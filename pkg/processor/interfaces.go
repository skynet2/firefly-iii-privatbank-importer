package processor

import (
	"context"
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
)

type Repo interface {
	AddMessage(ctx context.Context, message database.Message) error
	GetLatestMessages(ctx context.Context, source database.TransactionSource) ([]*database.Message, error)
	Clear(ctx context.Context) error
	UpdateMessage(ctx context.Context, message *database.Message) error
}

type Parser interface {
	ParseMessages(
		ctx context.Context,
		raw [][]byte,
		date time.Time,
	) (*database.Transaction, error)
	Type() database.TransactionSource
}

type Firefly interface {
	ListAccounts(ctx context.Context) ([]*firefly.Account, error)
	MapTransactions(
		ctx context.Context,
		transactions []*database.Transaction,
	) ([]*firefly.MappedTransaction, error)
	CreateTransactions(ctx context.Context, tx *firefly.Transaction) (*firefly.Transaction, error)
}

type NotificationSvc interface {
	React(
		ctx context.Context,
		chatID int64,
		messageID int64,
		reaction string,
	) error

	SendMessage(
		ctx context.Context,
		chatID int64,
		text string,
	) error
}
