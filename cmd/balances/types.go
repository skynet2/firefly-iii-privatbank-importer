package main

import (
	"time"

	"github.com/shopspring/decimal"
)

type genericResponse[T any] struct {
	Data T `json:"data"`
}

type accountResponse struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Attributes accountAttributes `json:"attributes"`
}

type accountAttributes struct {
	Active         bool            `json:"active"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	CurrencyID     string          `json:"currency_id"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
}

type simpleAccountData struct {
	ID         int
	Balance    decimal.Decimal
	CurrencyID int
	UpdatedAt  time.Time
}

func (s *simpleAccountData) TableName() string {
	return "simple_account_data_importer"
}
