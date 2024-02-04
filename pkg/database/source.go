package database

type TransactionSource string

const (
	PrivatBank = TransactionSource("privatbank")
	Paribas    = TransactionSource("paribas")
)
