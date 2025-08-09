package parser

import (
	"strings"
	"unicode"
)

func toLines(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")

	return strings.Split(input, "\n")
}

func stripAccountPrefix(account string) string {
	account = strings.ToLower(account)
	var accountStriped strings.Builder

	for idx, l := range account {
		if !unicode.IsLetter(l) && idx == 0 {
			return account
		}

		if unicode.IsLetter(l) {
			continue
		}
		accountStriped.WriteRune(l)
	}

	return accountStriped.String()
}
