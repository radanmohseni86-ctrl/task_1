package repository

import (
	"context"
	"database/sql"
)

type CalcHistoryRepository interface {
	Insert(ctx context.Context, num1, num2, answer float64, operation, name string) error
	Migrate(ctx context.Context) error
}

type postgresCalcHistoryRepository struct {
	db *sql.DB
}

func NewPostgresCalcHistoryRepository(db *sql.DB) CalcHistoryRepository {
	return &postgresCalcHistoryRepository{db: db}
}

func (r *postgresCalcHistoryRepository) Insert(ctx context.Context, num1, num2, answer float64, operation, name string) error {
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO calc_history (num1, num2, operation, answer, name) VALUES ($1, $2, $3, $4, $5)",
		num1, num2, operation, answer, name)
	return err
}

func (r *postgresCalcHistoryRepository) Migrate(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS calc_history (id SERIAL PRIMARY KEY, num1 DOUBLE PRECISION, num2 DOUBLE PRECISION, operation TEXT, answer DOUBLE PRECISION, name TEXT)"); err != nil {
		return err
	}
	if _, err := r.db.ExecContext(ctx, "ALTER TABLE calc_history ADD COLUMN IF NOT EXISTS name TEXT"); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, "ALTER TABLE calc_history ALTER COLUMN num1 TYPE DOUBLE PRECISION, ALTER COLUMN num2 TYPE DOUBLE PRECISION, ALTER COLUMN answer TYPE DOUBLE PRECISION")
	return err
}
