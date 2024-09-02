package duplicatecleaner

import "github.com/cockroachdb/errors"

var DuplicateTransactionError = errors.New("duplicate transaction")
