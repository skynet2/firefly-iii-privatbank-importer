package duplicatecleaner

import (
	"context"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

//go:generate mockgen -destination interfaces_mocks_test.go -package duplicatecleaner_test -source=interfaces.go

type Repo interface {
	IsDuplicateKeyExists(ctx context.Context, key string, source database.TransactionSource) (bool, error)
	AddDuplicateKey(ctx context.Context, key string, source database.TransactionSource) error
}
