package sqldb

import (
	"context"
	"fmt"
	"go-project-template/transaction"

	"github.com/jmoiron/sqlx"
)

type dbBeginner struct {
	db *DB
}

func NewBeginner(db *DB) transaction.Beginner {
	return &dbBeginner{db: db}
}

func (b *dbBeginner) Begin(ctx context.Context) (transaction.Transaction, error) {
	tx, err := b.db.SQL().BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin postgres transaction failed: %w", err)
	}
	return tx, nil
}

func GetExtContext(tx transaction.Transaction) (sqlx.ExtContext, error) {
	ext, ok := tx.(sqlx.ExtContext)
	if !ok {
		return nil, fmt.Errorf("unsupported transaction type %T", tx)
	}
	return ext, nil
}
