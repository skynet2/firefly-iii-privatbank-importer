package revolut

import (
	"time"
)

type FetchRequest struct {
	Token string
	After time.Time
}

type transaction struct {
	Id            string             `json:"id"`
	Type          string             `json:"type"`
	State         string             `json:"state"`
	Currency      string             `json:"currency"`
	Amount        int32              `json:"amount"`
	Description   string             `json:"description"`
	Tag           string             `json:"tag"`
	Comment       string             `json:"comment"`
	Account       account            `json:"account"`
	Beneficiary   revolutBeneficiary `json:"beneficiary"`
	Sender        revolutSender      `json:"sender"`
	Recipient     revolutSender      `json:"recipient"`
	StartedDate   int64              `json:"startedDate"`
	CreatedDate   int64              `json:"createdDate"`
	CompletedDate int64              `json:"completedDate"`
	Counterpart   revolutCounterPart `json:"counterpart"`
}

type account struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type revolutBeneficiary struct {
	ID      string `json:"id"`
	Country string `json:"country"`
}

type revolutSender struct {
	Account account `json:"account"`
}

type revolutCounterPart struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
