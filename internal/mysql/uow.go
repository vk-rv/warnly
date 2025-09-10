package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vk-rv/warnly/internal/uow"
	"github.com/vk-rv/warnly/internal/warnly"
)

// unitOfWork implements uow.UnitOfWork interface.
type unitOfWork struct {
	messageStore    *MessageStore
	mentionStore    *MentionStore
	assingmentStore *AssingmentStore
	tx              *sql.Tx
	t               uow.Type
}

// key is an unexported type for keys defined in this package.
type key struct{}

// uowKey is the key for unitOfWork values in Contexts. It is used to retrieve the unitOfWork from the context.
var uowKey key

// NewUOW returns an implementation of the interface uow.StartUnitOfWork
// that will track all repositories.
func NewUOW(db *sql.DB, logger *slog.Logger) uow.StartUnitOfWork {
	return func(ctx context.Context, t uow.Type, uowFn uow.UnitOfWorkFn, storages ...any) (err error) {
		uw := &unitOfWork{t: t}
		if ctxOUW, ok := ctx.Value(uowKey).(*unitOfWork); ok {
			for i := range storages {
				if err := ctxOUW.add(storages[i]); err != nil {
					return fmt.Errorf("could not add repository: %w", err)
				}
			}
			ctx = context.WithValue(ctx, uowKey, ctxOUW)
			return uowFn(ctx, ctxOUW)
		}

		ctx = context.WithValue(ctx, uowKey, uw)
		err = uw.begin(ctx, db)
		if err != nil {
			return fmt.Errorf("could not initialize TX: %w", err)
		}
		defer func() {
			if r := recover(); r != nil {
				if rollBackErr := uw.rollback(); rollBackErr != nil {
					logger.Error("problem while trying to rollback after recover in transaction",
						slog.Any("error", rollBackErr))
				}
				panic(r)
			}

			rollbackErr := uw.rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = fmt.Errorf("failed to rollback TX: %w", rollbackErr)
			}
		}()

		for i := range storages {
			if err := uw.add(storages[i]); err != nil {
				return fmt.Errorf("could not add repository: %w", err)
			}
		}

		defer func() {
			if err == nil {
				commitErr := uw.commit()
				if commitErr != nil {
					err = fmt.Errorf("failed to commit TX: %w", commitErr)
				}
			}
		}()

		return uowFn(ctx, uw)
	}
}

//nolint:ireturn // temporary
func (uw *unitOfWork) Messages() warnly.MessageStore { return uw.messageStore }

//nolint:ireturn // temporary
func (uw *unitOfWork) Mentions() warnly.MentionStore { return uw.mentionStore }

//nolint:ireturn // temporary
func (uw *unitOfWork) Assignments() warnly.AssingmentStore { return uw.assingmentStore }

// add adds repository to the unitOfWork
// by setting its db field to the current transaction.
func (uw *unitOfWork) add(r any) error {
	switch rep := r.(type) {
	case *MentionStore:
		if uw.mentionStore == nil {
			r := *rep
			r.db = uw.tx
			uw.mentionStore = &r
		}
		return nil
	case *MessageStore:
		if uw.messageStore == nil {
			r := *rep
			r.db = uw.tx
			uw.messageStore = &r
		}
		return nil
	case *AssingmentStore:
		if uw.assingmentStore == nil {
			r := *rep
			r.db = uw.tx
			uw.assingmentStore = &r
		}
		return nil
	default:
		return fmt.Errorf("invalid repository of type: %T", rep)
	}
}

// commit commits the transaction.
func (uw *unitOfWork) commit() error { return uw.tx.Commit() }

// rollback rollbacks the transaction.
func (uw *unitOfWork) rollback() error { return uw.tx.Rollback() }

// begin starts a new transaction.
func (uw *unitOfWork) begin(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	uw.tx = tx
	return nil
}
