package processor

import (
	"context"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	parser2 "github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
)

//go:generate mockgen -destination interfaces_mocks_test.go -package processor_test -source=interfaces.go

type Repo interface {
	AddMessage(ctx context.Context, messages []database.Message) error
	GetLatestMessages(ctx context.Context, source database.TransactionSource) ([]*database.Message, error)
	Clear(ctx context.Context, transactionSource database.TransactionSource) error
	UpdateMessages(ctx context.Context, message []*database.Message) error
}

type Printer interface {
	Dry(
		ctx context.Context,
		mappedTx []*firefly.MappedTransaction,
		errArr []error,
	) string

	Commit(
		ctx context.Context,
		mappedTx []*firefly.MappedTransaction,
		errArr []error,
	) string

	Stat(
		_ context.Context,
		mappedTx []*firefly.MappedTransaction,
		errArr []error,
	) string

	Duplicates(
		_ context.Context,
		mappedTx []*firefly.MappedTransaction,
		errArr []error,
	) string

	Errors(
		_ context.Context,
		mappedTx []*firefly.MappedTransaction,
		errArr []error,
	) string
}

type Parser interface {
	ParseMessages(
		ctx context.Context,
		raw []*parser2.Record,
	) ([]*database.Transaction, error)
	Type() database.TransactionSource

	SplitExcel(
		_ context.Context,
		data []byte,
	) ([][]byte, error)
}

type Firefly interface {
	ListAccounts(ctx context.Context) ([]*firefly.Account, error)
	MapTransactions(
		ctx context.Context,
		transactions []*database.Transaction,
	) ([]*firefly.MappedTransaction, error)
	CreateTransactions(
		ctx context.Context,
		tx *firefly.Transaction,
		errorOnDuplicate bool,
	) (*firefly.Transaction, error)
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

	GetFile(ctx context.Context, fileID string) ([]byte, error)
}

type DuplicateCleaner interface {
	IsDuplicate(
		ctx context.Context,
		key string,
		txSource database.TransactionSource,
	) error

	AddDuplicateKey(
		ctx context.Context,
		key string,
		txSource database.TransactionSource,
	) error
}
