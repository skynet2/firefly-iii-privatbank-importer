package firefly

import "github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"

type GenericApiResponse[T any] struct {
	Data T `json:"data"`
}

type Account struct {
	Id         string            `json:"id"`
	Attributes AccountAttributes `json:"attributes"`
}

type AccountAttributes struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	CurrencyCode  string `json:"currency_code"`
	Iban          string `json:"iban"`
	Bic           string `json:"bic"`
	AccountNumber string `json:"account_number"`
	Active        bool   `json:"active"`
}

type MappedTransaction struct {
	Original            *database.Transaction
	Transaction         *Transaction
	FireflyMappingError error
	IsCommitted         bool
}

type Transaction struct {
	Type                string `json:"type"`
	Date                string `json:"date"`
	Amount              string `json:"amount"`
	Description         string `json:"description"`
	CurrencyCode        string `json:"currency_code"`
	SourceID            string `json:"source_id"`
	SourceName          string `json:"-"`
	DestinationID       string `json:"destination_id,omitempty"`
	DestinationName     string `json:"-"`
	Notes               string `json:"notes"`
	ForeignAmount       string `json:"foreign_amount,omitempty"`
	ForeignCurrencyCode string `json:"foreign_currency_code,omitempty"`
}
