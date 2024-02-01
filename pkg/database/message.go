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

	ChatID    int64 `json:"chatId"`
	MessageID int64 `json:"messageId"`
}

type Transaction struct {
	ID   string
	Type TransactionType

	SourceAmount   decimal.Decimal
	SourceCurrency string

	DestinationAmount   decimal.Decimal
	DestinationCurrency string

	Date               time.Time
	Description        string
	SourceAccount      string
	DestinationAccount string
	DateFromMessage    string
	Raw                string

	InternalTransferDirectionTo bool
	DuplicateTransactions       []*Transaction

	OriginalMessage *Message `json:"-"`
}

type TransactionType int32

const (
	TransactionTypeUnknown          = TransactionType(0)
	TransactionTypeIncome           = TransactionType(1)
	TransactionTypeExpense          = TransactionType(2)
	TransactionTypeInternalTransfer = TransactionType(3)
	TransactionTypeRemoteTransfer   = TransactionType(4)
)
