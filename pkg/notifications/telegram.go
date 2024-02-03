package notifications

import (
	"context"
	"fmt"
	"os"

	"github.com/imroc/req/v3"
)

type Telegram struct {
	client   *req.Client
	apiToken string
}

func NewTelegram(
	apiToken string,
	cl *req.Client,
) *Telegram {
	return &Telegram{
		client:   cl,
		apiToken: apiToken,
	}
}

func (t *Telegram) GetFile(ctx context.Context, fileID string) ([]byte, error) {
	var fileResp getFileResponse

	resp, err := t.client.R().
		SetContext(ctx).
		SetSuccessResult(&fileResp).
		EnableDumpTo(os.Stdout).
		Get(fmt.Sprintf("https://api.telegram.org/bot%v/getFile?file_id=%v", t.apiToken, fileID))
	if err != nil {
		return nil, err
	}

	if resp.IsErrorState() {
		return nil, fmt.Errorf("unexpected status code: %v and message %v", resp.StatusCode, resp.String())
	}

	resp, err = t.client.R().
		SetContext(ctx).
		Get(fmt.Sprintf("https://api.telegram.org/file/bot%v/%v", t.apiToken, fileResp.Result.FilePath))
	if err != nil {
		return nil, err
	}

	return resp.Bytes(), nil
}

func (t *Telegram) SendMessage(
	ctx context.Context,
	chatID int64,
	text string,
) error {
	if len(text) > 4096 {
		text = text[:4090]
	}
	resp, err := t.client.R().
		SetBody(map[string]interface{}{
			"chat_id": chatID,
			"text":    text,
		}).
		SetContext(ctx).
		Post(fmt.Sprintf("https://api.telegram.org/bot%v/sendMessage", t.apiToken))

	if err != nil {
		return err
	}

	if resp.IsErrorState() {
		return fmt.Errorf("unexpected status code: %v and message %v", resp.StatusCode, resp.String())
	}

	return nil
}

func (t *Telegram) React(
	ctx context.Context,
	chatID int64,
	messageID int64,
	reaction string,
) error {
	resp, err := t.client.R().
		SetBody(map[string]interface{}{
			"chat_id":    chatID,
			"message_id": messageID,
			"reaction": []map[string]interface{}{
				{
					"type":  "emoji",
					"emoji": reaction,
				},
			},
		}).
		SetContext(ctx).
		Post(fmt.Sprintf("https://api.telegram.org/bot%v/setMessageReaction", t.apiToken))

	if err != nil {
		return err
	}

	if resp.IsErrorState() {
		return fmt.Errorf("unexpected status code: %v and message %v", resp.StatusCode, resp.String())
	}

	return nil
}
