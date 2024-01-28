package processor

import (
	"context"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Repo interface {
	AddMessage(ctx context.Context, message string)
	GetLatestMessages(ctx context.Context) ([]database.Message, error)
	Clear(ctx context.Context) error
}
