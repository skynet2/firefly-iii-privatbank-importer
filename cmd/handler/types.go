package main

type Webhook struct {
	Message  Message `json:"message"`
	UpdateId int64   `json:"update_id"`
}

type Message struct {
	Date          int64          `json:"date"`
	ForwardOrigin *ForwardOrigin `json:"forward_from"`
	Text          string
	Chat          Chat `json:"chat"`
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

type Config struct {
	DbName string `env:"DB_NAME" envDefault:"firefly_iii"`
}
