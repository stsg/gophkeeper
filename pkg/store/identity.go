package postgres

import (
	"bufio"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrUserNotFound      = fmt.Errorf("user not found")
	ErrUserExists        = fmt.Errorf("user exists")
	ErrUserUnauthorized  = fmt.Errorf("user unauthorized")
	ErrUserWrong         = fmt.Errorf("user wrong")
	ErrUserWrongPassword = fmt.Errorf("user password wrong")

	ErrResourceNotFound = fmt.Errorf("resource not found")
)

const (
	keyLen                         = 32
	keyIter                        = 4096
	ResourceTypePiece ResourceType = iota + 1
	ResourceTypeBlob
)

type (
	ResourceID   int64
	ResourceType int
)

type Piece struct {
	Content []byte // Content of the piece.
	Meta    string // Meta info of the piece.
}

type Blob struct {
	Content io.ReadCloser // Content of the blob.
	Meta    string        // Meta info of the blob.
}

type Resource struct {
	ID   ResourceID
	Type ResourceType
	Meta string
}

type ComposedReadCloser struct {
	Reader io.Reader
	Closer io.Closer
}

func (rc *ComposedReadCloser) Read(p []byte) (int, error) {
	return rc.Reader.Read(p)
}

func (rc *ComposedReadCloser) Close() error {
	return rc.Closer.Close()
}

// StorePiece stores a piece of content in the database along with its metadata and owner information.
//
// Parameters:
// - ctx: The context.Context object for the function.
// - piece: The Piece struct containing the content and metadata of the piece.
// - c: The Creds struct containing the credentials of the owner.
//
// Returns:
// - ResourceID: The ID of the stored resource.
// - error: An error if the storage operation fails.
func (p *Storage) StorePiece(ctx context.Context, piece Piece, c Creds) (ResourceID, error) {
	if err := p.checkPass(ctx, c); err != nil {
		return -1, errors.Join(err, ErrUserUnauthorized)
	}

	var (
		salt []byte = make([]byte, 8)
		iv   []byte = make([]byte, 12)
		key  []byte
	)
	if _, err := rand.Read(salt); err != nil {
		return -1, err
	}
	if _, err := rand.Read(iv); err != nil {
		return -1, err
	}
	key = pbkdf2.Key(([]byte)(c.Passw), salt, keyIter, keyLen, sha256.New)
	var block, blockError = aes.NewCipher(key)
	if blockError != nil {
		return -1, blockError
	}
	var aesgcm, aesgcmError = cipher.NewGCM(block)
	if aesgcmError != nil {
		return -1, aesgcmError
	}
	var content = aesgcm.Seal(nil, iv, piece.Content, nil)

	var transaction, transactionError = p.db.Begin(ctx)
	if transactionError != nil {
		return -1, transactionError
	}
	defer transaction.Rollback(ctx)

	insertPieceResult := transaction.QueryRow(
		ctx,
		`INSERT INTO pieces(content, salt, iv) VALUES($1, $2, $3) RETURNING id`,
		content, salt, iv,
	)
	var id int
	if err := insertPieceResult.Scan(&id); err != nil {
		return -1, err
	}
	insertResourceResult := transaction.QueryRow(
		ctx,
		`INSERT INTO resources(meta, resource, type, owner) VALUES($1, $2, $3, $4) RETURNING id`,
		piece.Meta, id, (int)(ResourceTypePiece), c.Login,
	)
	var rid int64
	if err := insertResourceResult.Scan(&rid); err != nil {
		return -1, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return -1, err
	}

	return (ResourceID)(rid), nil
}

// RestorePiece retrieves a piece of content and its metadata from the database based on the provided resource ID and credentials.
//
// Parameters:
// - ctx: The context.Context object for the function.
// - rid: The resource ID of the piece to be restored.
// - c: The Creds struct containing the credentials of the owner.
//
// Returns:
// - Piece: The restored piece of content with its metadata.
// - error: An error if the restoration operation fails.
func (p *Storage) RestorePiece(ctx context.Context, rid ResourceID, c Creds) (Piece, error) {
	if err := p.checkPass(ctx, c); err != nil {
		return Piece{}, errors.Join(err, ErrUserUnauthorized)
	}

	var (
		meta    string
		content []byte
		iv      []byte
		salt    []byte
	)

	var queryResourceResult = p.db.QueryRow(
		ctx,
		`SELECT meta, resource FROM resources WHERE id = $1 AND owner = $2 AND type = $3`,
		(int64)(rid), c.Login, (int)(ResourceTypePiece),
	)
	var id int
	if err := queryResourceResult.Scan(&meta, &id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Piece{}, ErrResourceNotFound
		}
		return Piece{}, err
	}
	var queryPieceResult = p.db.QueryRow(
		ctx,
		`SELECT content, iv, salt FROM pieces WHERE id = $1`,
		id,
	)
	if err := queryPieceResult.Scan(&content, &iv, &salt); err != nil {
		return Piece{}, err
	}

	var key = pbkdf2.Key(([]byte)(c.Passw), salt, keyIter, keyLen, sha256.New)
	var block, blockError = aes.NewCipher(key)
	if blockError != nil {
		return Piece{}, blockError
	}
	var aesgcm, aesgcmError = cipher.NewGCM(block)
	if aesgcmError != nil {
		return Piece{}, aesgcmError
	}
	var decryptedContent, openError = aesgcm.Open(nil, iv, content, nil)
	if openError != nil {
		return Piece{}, openError
	}

	var piece = Piece{
		Meta:    meta,
		Content: decryptedContent,
	}
	return piece, nil
}

// StoreBlob stores a blob in the storage.
//
// It takes the following parameters:
// - ctx: the context.Context object for controlling the execution flow.
// - blob: the Blob object containing the content to be stored.
// - c: the Creds object containing the credentials for authentication.
//
// It returns the ResourceID of the stored blob and an error if any.
func (p *Storage) StoreBlob(ctx context.Context, blob Blob, c Creds) (ResourceID, error) {
	defer blob.Content.Close()
	if err := p.checkPass(ctx, c); err != nil {
		return -1, errors.Join(err, ErrUserUnauthorized)
	}

	var salt []byte = make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return -1, err
	}

	var block, blockError = aes.NewCipher(
		pbkdf2.Key(([]byte)(c.Passw), salt, keyIter, keyLen, sha256.New),
	)
	if blockError != nil {
		return -1, blockError
	}

	var iv []byte = make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return -1, err
	}

	var location = path.Join(p.BlobsDir, uuid.New().String())
	var file, createError = os.Create(location)
	if createError != nil {
		return -1, createError
	}

	var (
		writer = cipher.StreamWriter{
			S: cipher.NewCTR(block, iv),
			W: file,
		}
		reader = bufio.NewReader(blob.Content)
	)
	if _, err := reader.WriteTo(writer); err != nil {
		log.Printf("failed to write file: %s\n", err.Error())
		if err := file.Close(); err != nil {
			log.Printf("failed to close file: %s\n", err.Error())
		}
		if err := os.Remove(location); err != nil {
			log.Printf("failed to remove file: %s\n", err.Error())
		}
		return -1, err
	}
	if err := file.Close(); err != nil {
		log.Printf("failed to close file: %s\n", err.Error())
		return -1, err
	}

	var transaction, transactionError = p.db.Begin(ctx)
	if transactionError != nil {
		return -1, transactionError
	}
	defer transaction.Rollback(ctx)

	var (
		blobID int
		rid    int64
	)

	var insertBlobResult = transaction.QueryRow(
		ctx,
		`INSERT INTO blobs(location, iv, salt) VALUES($1, $2, $3) RETURNING id`,
		location, iv, salt,
	)
	if err := insertBlobResult.Scan(&blobID); err != nil {
		return -1, err
	}

	var insertResourceResult = transaction.QueryRow(
		ctx,
		`INSERT INTO resources(meta, owner, type, resource) VALUES($1, $2, $3, $4) RETURNING id`,
		blob.Meta, c.Login, ResourceTypeBlob, blobID,
	)
	if err := insertResourceResult.Scan(&rid); err != nil {
		return -1, err
	}

	if err := transaction.Commit(ctx); err != nil {
		return -1, err
	}

	return (ResourceID)(rid), nil
}

// RestoreBlob retrieves a blob from the storage based on the provided resource ID and credentials.
//
// Parameters:
// - ctx: the context.Context object for controlling the execution flow.
// - rid: the ResourceID of the blob to be restored.
// - c: the Creds object containing the credentials for authentication.
//
// Returns:
// - Blob: the restored blob.
// - error: an error if the blob retrieval fails.
func (p *Storage) RestoreBlob(ctx context.Context, rid ResourceID, c Creds) (Blob, error) {
	if err := p.checkPass(ctx, c); err != nil {
		return Blob{}, errors.Join(err, ErrUserUnauthorized)
	}

	var (
		iv       []byte
		salt     []byte
		location string
		meta     string
	)

	var selectResourceResult = p.db.QueryRow(
		ctx,
		`SELECT meta, resource FROM resources WHERE id = $1 AND owner = $2`,
		(int64)(rid), c.Login,
	)
	var blobID int
	if err := selectResourceResult.Scan(&meta, &blobID); err != nil {
		return Blob{}, err
	}

	var selectBlobResult = p.db.QueryRow(
		ctx,
		`SELECT location, iv, salt FROM blobs WHERE id = $1`,
		blobID,
	)
	if err := selectBlobResult.Scan(&location, &iv, &salt); err != nil {
		return Blob{}, err
	}

	var file, fileError = os.Open(location)
	if fileError != nil {
		return Blob{}, fileError
	}

	var block, blockError = aes.NewCipher(
		pbkdf2.Key(([]byte)(c.Passw), salt, keyIter, keyLen, sha256.New),
	)
	if blockError != nil {
		return Blob{}, blockError
	}

	var blob = Blob{
		Meta: meta,
		Content: &ComposedReadCloser{
			Reader: cipher.StreamReader{
				S: cipher.NewCTR(block, iv),
				R: file,
			},
			Closer: file,
		},
	}
	return blob, nil
}

// Delete deletes a resource from the database.
//
// It takes the following parameters:
// - ctx: the context.Context object for the function.
// - rid: the ResourceID of the resource to be deleted.
// - c: the Creds object containing the login information of the owner of the resource.
//
// It returns an error if there was a problem deleting the resource.
func (p *Storage) Delete(ctx context.Context, rid ResourceID, c Creds) error {
	var transaction, transactionError = p.db.Begin(ctx)
	if transactionError != nil {
		return transactionError
	}
	defer transaction.Rollback(ctx)

	var deleteResourceResult = transaction.QueryRow(
		ctx,
		`DELETE FROM resources WHERE id = $1 AND owner = $2 RETURNING type, resource`,
		(int64)(rid), c.Login,
	)
	var (
		resourceType int
		resourceID   int
	)
	if err := deleteResourceResult.Scan(&resourceType, &resourceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrResourceNotFound
		}
		return err
	}

	switch (ResourceType)(resourceType) {
	case ResourceTypePiece:
		_, err := transaction.Exec(
			ctx,
			`DELETE FROM pieces WHERE id = $1`,
			resourceID,
		)
		if err != nil {
			return err
		}
	case ResourceTypeBlob:
		var deleteResult = transaction.QueryRow(
			ctx,
			`DELETE FROM blobs WHERE id = $1 RETURNING location`,
			resourceID,
		)
		var location string
		if err := deleteResult.Scan(&location); err != nil {
			return err
		}
		if err := os.Remove(location); err != nil {
			return err
		}
	default:
		log.Fatalf("unknown resource type: %d", resourceType)
	}

	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// List retrieves a list of resources owned by the given credentials from the storage.
//
// ctx: The context.Context object for the request.
// c: The Creds object containing the login of the owner.
// []Resource: A slice of Resource objects representing the resources owned by the owner.
// error: An error object if there was an issue retrieving the resources.
func (p *Storage) List(ctx context.Context, c Creds) ([]Resource, error) {
	var selectResourcesResult, selectResourcesResultError = p.db.Query(
		ctx,
		`SELECT id, type, meta FROM resources WHERE owner = $1`,
		c.Login,
	)
	if selectResourcesResultError != nil {
		log.Fatal(selectResourcesResultError)
		return nil, selectResourcesResultError
	}
	defer selectResourcesResult.Close()
	var resources []Resource
	for selectResourcesResult.Next() {
		if err := selectResourcesResult.Err(); err != nil {
			log.Fatal(err)
			return nil, err
		}
		var resource Resource
		if err := selectResourcesResult.Scan(&resource.ID, &resource.Type, &resource.Meta); err != nil {
			log.Fatal(err)
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// checkPass checks the password for the given credentials in the Storage.
//
// ctx: The context.Context object for the request.
// c: The Creds object containing the login and password to be checked.
// error: An error object if there was an issue checking the password.
func (p *Storage) checkPass(ctx context.Context, c Creds) error {
	var row = p.db.QueryRow(
		ctx,
		`SELECT password FROM identities WHERE username = $1`,
		c.Login,
	)
	var encodedPassword string
	if err := row.Scan(&encodedPassword); err != nil {
		var pgerr pgconn.PgError
		if errors.As(err, (any)(&pgerr)) {
			return ErrUserUnauthorized
		}
		return err
	}

	var decodedPassword, decodePasswordError = p.EncdP.DecodeString(encodedPassword)
	if decodePasswordError != nil {
		return decodePasswordError
	}
	if err := bcrypt.CompareHashAndPassword(decodedPassword, ([]byte)(c.Passw)); err != nil {
		return errors.Join(ErrUserUnauthorized, err)
	}
	return nil
}
