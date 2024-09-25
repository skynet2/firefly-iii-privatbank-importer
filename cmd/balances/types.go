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

func (s simpleAccountData) Equal(target simpleAccountData) bool {
	if s.ID == 0 {
		return false
	}

	if !s.Balance.Equal(target.Balance) {
		return false
	}

	if s.CurrencyID != target.CurrencyID {
		return false
	}

	return true
}

func (s simpleAccountData) TableName() string {
	return "simple_account_data_importer"
}

type simpleAccountDataDaily struct {
	ID         int `gorm:"primaryKey"`
	Balance    decimal.Decimal
	CurrencyID int
	UpdatedAt  time.Time

	Date time.Time `gorm:"type:date;primaryKey"`
}

func (s simpleAccountDataDaily) Equal(target simpleAccountData, dateNow time.Time) bool {
	if s.Date.Format(time.DateOnly) != dateNow.Format(time.DateOnly) {
		return false
	}

	if s.ID == 0 {
		return false
	}

	if !s.Balance.Equal(target.Balance) {
		return false
	}

	if s.CurrencyID != target.CurrencyID {
		return false
	}

	return true
}

func (s simpleAccountDataDaily) TableName() string {
	return "simple_account_data_importer_daily"
}
