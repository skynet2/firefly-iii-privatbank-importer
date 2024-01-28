package processor

import (
	"context"
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Repo interface {
	AddMessage(ctx context.Context, message database.Message) error
	GetLatestMessages(ctx context.Context) ([]*database.Message, error)
	Clear(ctx context.Context) error
}

type Parser interface {
	ParseMessages(
		ctx context.Context,
		raw string,
		date time.Time,
	) (*database.Transaction, error)
}
