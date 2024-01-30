package database

import (
	"time"

	"github.com/shopspring/decimal"
)

type Message struct {
	ID          string     `json:"id"`
	CreatedAt   time.Time  `json:"createdAt"`
	ProcessedAt *time.Time `json:"processedAt"`
	IsProcessed bool       `json:"isProcessed"`
	Content     string     `json:"content"`
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
	DateFromMessage    string
	Raw                string

	FireflyTransaction  *FireflyTransaction
	FireflyMappingError error

	InternalTransferDirectionTo bool
	DuplicateTransactions       []*Transaction
}

type FireflyTransaction struct {
	Type            string
	SourceID        string
	SourceName      string
	DestinationID   string
	DestinationName string
	Description     string
	Notes           string
}

type TransactionType int32

const (
	TransactionTypeUnknown          = TransactionType(0)
	TransactionTypeIncome           = TransactionType(1)
	TransactionTypeExpense          = TransactionType(2)
	TransactionTypeInternalTransfer = TransactionType(3)
	TransactionTypeRemoteTransfer   = TransactionType(4)
)
