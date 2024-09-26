package common

import "github.com/cockroachdb/errors"

var (
	ErrDuplicate             = errors.New("duplicate transaction")
	ErrOperationNotSupported = errors.New("income transactions are not supported")
)
