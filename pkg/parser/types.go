package parser

import (
	"time"

	"github.com/google/uuid"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
)

type Record struct {
	Message *database.Message
	Data    []byte
}

type revolutTransaction struct {
	Id            uuid.UUID          `json:"id"`
	Type          string             `json:"type"`
	State         string             `json:"state"`
	Currency      string             `json:"currency"`
	Amount        int32              `json:"amount"`
	Description   string             `json:"description"`
	Tag           string             `json:"tag"`
	Comment       string             `json:"comment"`
	Account       revolutAccount     `json:"account"`
	Beneficiary   revolutBeneficiary `json:"beneficiary"`
	Sender        revolutSender      `json:"sender"`
	Recipient     revolutSender      `json:"recipient"`
	StartedDate   int64              `json:"startedDate"`
	CreatedDate   int64              `json:"createdDate"`
	CompletedDate int64              `json:"completedDate"`
	Counterpart   revolutCounterPart `json:"counterpart"`
}

type revolutCounterPart struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

func (m revolutTransaction) StartedAt() time.Time {
	return time.Unix(0, m.StartedDate*int64(time.Millisecond)).UTC()
}

type revolutSender struct {
	Account revolutAccount `json:"account"`
}

type revolutAccount struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type revolutBeneficiary struct {
	ID      string `json:"id"`
	Country string `json:"country"`
}
