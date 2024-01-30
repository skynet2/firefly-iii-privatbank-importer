package firefly

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

type Transaction struct {
	Type          string `json:"type"`
	Date          string `json:"date"`
	Amount        string `json:"amount"`
	Description   string `json:"description"`
	CurrencyCode  string `json:"currency_code"`
	SourceID      string `json:"source_id"`
	DestinationID string `json:"destination_id,omitempty"`
	Notes         string `json:"notes"`
}
