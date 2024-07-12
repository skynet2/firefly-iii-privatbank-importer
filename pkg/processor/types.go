package processor

import (
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
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
}
