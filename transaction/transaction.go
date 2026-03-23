package transaction

import (
	"context"
	"go-project-template/logger"
)

type Transaction interface {
	Commit() error
	Rollback() error
}

type Beginner interface {
	Begin(ctx context.Context) (Transaction, error)
}

func ExecuteUnderTransaction(
	ctx context.Context,
	beginner Beginner,
	log *logger.Logger,
	fn func(tx Transaction) error,
) (err error) {
	log.Info(ctx, "transaction.begin")

	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log.Error(ctx, "transaction.rollback_panic")
			_ = tx.Rollback()
			panic(recovered)
		}

		if err != nil {
			log.Error(ctx, "transaction.rollback", "error", err.Error())
			_ = tx.Rollback()
			return
		}
		log.Info(ctx, "transaction.commit")
		if commitErr := tx.Commit(); commitErr != nil {
			log.Error(ctx, "transaction.commit_failed", "error", commitErr.Error())
			err = commitErr
		}
	}()

	err = fn(tx)
	return
}
