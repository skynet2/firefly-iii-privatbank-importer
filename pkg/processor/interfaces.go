package processor

import (
	"context"
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
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

type Firefly interface {
	ListAccounts(ctx context.Context) ([]*firefly.Account, error)
	MapTransactions(
		ctx context.Context,
		transactions []*database.Transaction,
	) ([]*firefly.MappedTransaction, error)
}

type NotificationSvc interface {
	React(
		ctx context.Context,
		chatID int64,
		messageID int64,
	) error

	SendMessage(
		ctx context.Context,
		chatID int64,
		text string,
	) error
}
