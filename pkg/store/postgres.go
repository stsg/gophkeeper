package postgres

import (
	"context"
	"errors"
	"fmt"

	log "github.com/go-pkgz/lgr"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stsg/gophkeeper/pkg/lib"
)

type Storage struct {
	cfg *Config
	db  *pgxpool.Pool
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

func (p *Storage) GetUserByLogin(ctx context.Context, login string) (User, error) {
	var user User

	err := p.db.QueryRow(
		ctx,
		"SELECT id, passw FROM identities WHERE id=$1", login).Scan(
		&user.Login,
		&user.Passw,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrNoExists
		}
		return User{}, err
	}
	return user, nil

}

func (p *Storage) GetUserByUUID(ctx context.Context, uid uuid.UUID) (User, error) {
	var user User

	err := p.db.QueryRow(
		ctx,
		"SELECT id, passw FROM users WHERE uid=$1", uid).Scan(
		&user.Login,
		&user.Passw,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return User{}, ErrNoExists
		}
		return User{}, err
	}
	return user, nil

}

func (p *Storage) CreateUser(ctx context.Context, user *User) (*User, error) {
	_, err := p.db.Exec(
		ctx,
		"INSERT INTO users (uid, login, password) VALUES ($1, $2, $3)",
		user.Login,
		user.Passw,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			log.Printf("[ERROR] user %s already exists %v", user.Login, err)
			return nil, ErrUserExists
		}
		log.Printf("[ERROR] cannot create user %s %v", user.Login, err)
		return nil, err
	}
	return user, nil
}
