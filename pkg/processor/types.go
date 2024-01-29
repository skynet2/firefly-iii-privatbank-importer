package processor

import "time"

type Message struct {
	ID            string
	Date          time.Time
	OriginalDate  time.Time
	ChatID        int64
	Content       string
	ForwardedFrom string
	MessageID     int64
}
