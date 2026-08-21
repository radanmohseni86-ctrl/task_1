package repository

import (
	"context"
	"database/sql"
)

type UserRepository interface {
	Exists(ctx context.Context, username string) (bool, error)
	Create(ctx context.Context, username, password string) error
	GetPassword(ctx context.Context, username string) (string, error)
	Migrate(ctx context.Context) error
}

type postgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Exists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	return exists, err
}

func (r *postgresUserRepository) Create(ctx context.Context, username, password string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO users (username, password) VALUES ($1, $2)", username, password)
	return err
}

func (r *postgresUserRepository) GetPassword(ctx context.Context, username string) (string, error) {
	var password string
	err := r.db.QueryRowContext(ctx, "SELECT password FROM users WHERE username = $1", username).Scan(&password)
	return password, err
}

func (r *postgresUserRepository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS users (username TEXT PRIMARY KEY, password TEXT NOT NULL)")
	return err
}
