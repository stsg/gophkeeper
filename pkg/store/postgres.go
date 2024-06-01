package postgres

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/golang-jwt/jwt"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stsg/gophkeeper/pkg/lib"
)

type Creds struct {
	Login string `json:"username"`
	Passw string `json:"password"`
}

type Storage struct {
	cfg      *Config
	db       *pgxpool.Pool
	EncdP    *base64.Encoding
	BlobsDir string
	Secret   []byte
	LifeSpan time.Duration
}

func (p *Storage) Close() {
	p.db.Close()
}

// Ping checks if the database connection is still alive by sending a ping request.
//
// ctx: The context.Context object for controlling the request's lifetime.
// Returns an error if the ping request fails.
func (p *Storage) Ping(ctx context.Context) error {
	return p.db.Ping(ctx)
}

// New creates a new Storage instance with the given configuration.
//
// It establishes a connection to the PostgreSQL database using the provided
// connection string and connection timeout. If the connection fails, an error
// is returned.
//
// If the "identities" table does not exist in the database, it runs the
// migration to create the table.
//
// Parameters:
//   - cfg: The configuration object containing the connection string,
//     connection timeout, and migration version.
//
// Returns:
//   - *Storage: A pointer to the newly created Storage instance.
//   - error: An error if the connection to the database fails or the migration
//     fails.
func New(cfg *Config) (*Storage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	if !lib.IsTableExist(pool, "identities") {
		if err := migrate(pool, cfg.MigrationVersion); err != nil {
			return nil, err
		}
	}

	return &Storage{cfg: cfg, db: pool}, nil
}

func (p *Storage) GetIdentity(ctx context.Context, login string) (Creds, error) {
	var c Creds

	err := p.db.QueryRow(
		ctx,
		"SELECT id, passw FROM identities WHERE id=$1", login).Scan(
		&c.Login,
		&c.Passw,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Creds{}, ErrNoExists
		}
		return Creds{}, err
	}
	return c, nil
}

func (p *Storage) Register(ctx context.Context, c Creds) error {
	_, err := p.db.Exec(
		ctx,
		"INSERT INTO identities (id, passw) VALUES ($1, $2)",
		c.Login,
		c.Passw,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			log.Printf("[ERROR] user %s already exists %v", c.Login, err)
			return ErrUniqueViolation
		}
		log.Printf("[ERROR] cannot create user %s %v", c.Login, err)
		return err
	}

	return nil
}

func (p *Storage) Authenticate(ctx context.Context, c Creds) (t string, err error) {

	if err := p.checkPass(ctx, c); err != nil {
		return "", err
	}

	var rawToken = jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"exp": time.Now().Add(p.LifeSpan).Unix(),
			"sub": c.Passw,
		},
	)
	var token, signTokenError = rawToken.SignedString(p.Secret)
	if signTokenError != nil {
		return "", signTokenError
	}
	return token, nil
}

func (p *Storage) Identity(ctx context.Context, t string) (c Creds, err error) {
	var parsedToken, parseTokenError = jwt.Parse(
		t,
		func(t *jwt.Token) (interface{}, error) {
			return p.Secret, nil
		},
	)
	if parseTokenError != nil {
		return Creds{}, ErrUserUnauthorized
	}

	var claims = parsedToken.Claims.(jwt.MapClaims)
	if claims.Valid() != nil {
		return Creds{}, ErrUserUnauthorized
	}

	sub := claims["sub"].(string)

	var username string
	if sub != "" {
		username = sub
	} else {
		return Creds{}, ErrUserUnauthorized
	}

	return Creds{Login: username, Passw: ""}, nil
}
