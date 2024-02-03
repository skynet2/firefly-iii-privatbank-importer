package main

type Webhook struct {
	Message  Message `json:"message"`
	UpdateId int64   `json:"update_id"`
}

type Message struct {
	Date          int64          `json:"date"`
	ForwardOrigin *ForwardOrigin `json:"forward_origin"`
	Document      *Document      `json:"document"`
	Text          string
	Chat          Chat  `json:"chat"`
	MessageID     int64 `json:"message_id"`
}

type Document struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
}

type Chat struct {
	Id int64 `json:"id"`
}

type ForwardOrigin struct {
	Date       int64      `json:"date"`
	SenderUser SenderUser `json:"sender_user"`
}

type SenderUser struct {
	UserName string `json:"username"`
	Id       int64  `json:"id"`
	IsBot    bool   `json:"is_bot"`
}
