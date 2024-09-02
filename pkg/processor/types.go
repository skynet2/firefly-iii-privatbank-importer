package processor

import (
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
)

type Message struct {
	ID                string
	Date              time.Time
	OriginalDate      time.Time
	ChatID            int64
	Content           string
	ForwardedFrom     string
	MessageID         int64
	TransactionSource database.TransactionSource
	FileID            string
}

type CommitResult struct {
	ExpectedReaction string
	Msg              *database.Message
	Tx               *firefly.MappedTransaction
}
