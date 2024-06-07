package revolut

import (
	"time"
)

type FetchRequest struct {
	Cookies  string
	DeviceID string
	After    time.Time
}

type Transaction struct {
	Id          string  `json:"id"`
	Type        string  `json:"type"`
	State       string  `json:"state"`
	Currency    string  `json:"currency"`
	Amount      int64   `json:"amount"`
	Description string  `json:"description"`
	Tag         string  `json:"tag"`
	Comment     string  `json:"comment"`
	Account     account `json:"account"`

	FromAccount account `json:"fromAccount"`
	ToAccount   account `json:"toAccount"`

	Beneficiary   revolutBeneficiary `json:"beneficiary"`
	Sender        revolutSender      `json:"sender"`
	Recipient     revolutSender      `json:"recipient"`
	StartedDate   int64              `json:"startedDate"`
	CreatedDate   int64              `json:"createdDate"`
	CompletedDate int64              `json:"completedDate"`
	Counterpart   revolutCounterPart `json:"counterpart"`
	CountryCode   string             `json:"countryCode"`
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
	ID        string `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Code      string `json:"code"`
}

type revolutCounterPart struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}
