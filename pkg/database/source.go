package database

type TransactionSource string

const (
	PrivatBank = TransactionSource("privatbank")
	Paribas    = TransactionSource("paribas")
	Revolut    = TransactionSource("revolut")
	Zen        = TransactionSource("zen")
	Mono       = TransactionSource("mono")
)
