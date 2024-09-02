package duplicatecleaner

import (
	"context"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Repo interface {
	IsDuplicateKeyExists(ctx context.Context, key string, source database.TransactionSource) (bool, error)
	AddDuplicateKey(ctx context.Context, key string, source database.TransactionSource) error
}
