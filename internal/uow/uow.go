// Package uow provides an interface in which the "repositories" that participate on it
// are asure that the functions/actions that are called will be rollback if the Unit of Work
// fails at some point.
// So it's not necessary to care about removing the already created data if an error raises
// on the middle of the Unit of Work. It's basically an interface to emulate a Transaction
// which is a more common word for it.
package uow

import (
	"context"

	"github.com/vk-rv/warnly/internal/warnly"
)

// Type is the type of the UniteOfWork.
type Type int

const (
	// Read is the type of UoW that only reads data.
	Read Type = iota

	// Write is the type of UoW that Reads and Writes data.
	Write
)

// UnitOfWork is the interface that any UnitOfWork has to follow
// the only methods it as are to return Repositories that work
// together to achieve a common purpose/work.
type UnitOfWork interface {
	Mentions() warnly.MentionStore
	Messages() warnly.MessageStore
	Assignments() warnly.AssingmentStore
}

// StartUnitOfWork is a function that starts a UnitOfWork (e.g. database transaction).
type StartUnitOfWork func(ctx context.Context, t Type, uowFn UnitOfWorkFn, storages ...any) error

// UnitOfWorkFn is the callback of the StartUnitOfWork.
type UnitOfWorkFn func(ctx context.Context, uw UnitOfWork) error
