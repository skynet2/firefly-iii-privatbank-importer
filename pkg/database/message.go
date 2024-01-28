package database

import (
	"time"

	"github.com/shopspring/decimal"
)

type Message struct {
	ID          string
	CreatedAt   time.Time
	ProcessedAt *time.Time
	Content     string
}

type Transaction struct {
	ID                 string
	Type               TransactionType
	Amount             decimal.Decimal
	Currency           string
	Date               time.Time
	Description        string
	SourceAccount      string
	DestinationAccount string

	InternalTransferDirectionTo bool
}

type TransactionType int32

const (
	TransactionTypeUnknown          = TransactionType(0)
	TransactionTypeIncome           = TransactionType(1)
	TransactionTypeExpense          = TransactionType(2)
	TransactionTypeInternalTransfer = TransactionType(3)
	TransactionTypeRemoteTransfer   = TransactionType(3)
)
