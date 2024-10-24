package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

type Handler struct {
	processor MessageProcessor
	chatMap   map[string]database.TransactionSource
}

func NewHandler(processor MessageProcessor, chatMap map[string]database.TransactionSource) *Handler {
	return &Handler{
		processor: processor,
		chatMap:   chatMap,
	}
}

func (h *Handler) ProcessWebhook(
	ctx context.Context,
	webhook Webhook,
) error {
	date := time.Unix(webhook.Message.Date, 0)
	originalDate := date
	forwardedFrom := ""

	if webhook.Message.ForwardOrigin != nil {
		originalDate = time.Unix(webhook.Message.ForwardOrigin.Date, 0)
		forwardedFrom = webhook.Message.ForwardOrigin.SenderUser.UserName
	}

	source := h.chatMap[fmt.Sprint(webhook.Message.Chat.Id)]

	_ = h.processor.ProcessMessage(ctx, processor.Message{
		ID:                strconv.FormatInt(webhook.UpdateId, 10),
		Date:              date,
		OriginalDate:      originalDate,
		ChatID:            webhook.Message.Chat.Id,
		Content:           webhook.Message.Text,
		ForwardedFrom:     forwardedFrom,
		MessageID:         webhook.Message.MessageID,
		FileID:            webhook.Message.Document.FileID,
		TransactionSource: source,
	})

	return nil
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if apiKey != r.URL.Query().Get("api_key") {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
		return
	}

	var webhook Webhook

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if err = json.Unmarshal(b, &webhook); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	if err = h.ProcessWebhook(r.Context(), webhook); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}
