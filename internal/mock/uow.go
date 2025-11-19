// Package mock contains mock implementations of various interfaces.
package mock

import (
	"context"

	"github.com/vk-rv/warnly/internal/uow"
)

// StartUnitOfWork is a mock implementation of uow.StartUnitOfWork.
func StartUnitOfWork(_ context.Context, _ uow.Type, fn uow.UnitOfWorkFn, _ ...any) error {
	return nil
}
