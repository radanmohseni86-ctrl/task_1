package repository

import (
	"context"
	"database/sql"
	"time"
)

type LoggedEventRepository interface {
	Insert(ctx context.Context, user string, timestamp time.Time) error
	Migrate(context.Context) error
}

type postgresLoggedEventRepository struct {
	db *sql.DB
}

func NewPostgresLoggedEventRepository(db *sql.DB) LoggedEventRepository {
	return &postgresLoggedEventRepository{db: db}
}

func (r *postgresLoggedEventRepository) Insert(ctx context.Context, user string, timestamp time.Time) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO logged_events (username, occurred_at) VALUES ($1, $2)",
		user, timestamp)
	return err
}

func (r *postgresLoggedEventRepository) Migrate(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS logged_events (id SERIAL PRIMARY KEY, username TEXT, occurred_at TIMESTAMPTZ)")
	return err
}
