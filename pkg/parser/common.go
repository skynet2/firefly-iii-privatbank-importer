package parser

import "strings"

func toLines(input string) []string {
	input = strings.ReplaceAll(input, "\r\n", "\n")

	return strings.Split(input, "\n")
}
