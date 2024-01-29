package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
)

type Handler struct {
	processor MessageProcessor
}

func NewHandler(
	processor MessageProcessor,
) *Handler {
	return &Handler{
		processor: processor,
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

	return h.processor.ProcessMessage(ctx, processor.Message{
		ID:            strconv.FormatInt(webhook.UpdateId, 10),
		Date:          date,
		OriginalDate:  originalDate,
		ChatID:        webhook.Message.Chat.Id,
		Content:       webhook.Message.Text,
		ForwardedFrom: forwardedFrom,
		MessageID:     webhook.Message.MessageID,
	})
}

func (h *Handler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if apiKey != r.URL.Query().Get("api_key") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var webhook Webhook

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
