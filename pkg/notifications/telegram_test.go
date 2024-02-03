package notifications_test

import (
	"context"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/notifications"
)

func TestGetFile(t *testing.T) {
	cl := req.DefaultClient()
	httpmock.ActivateNonDefault(cl.GetClient())
	defer httpmock.DeactivateAndReset()

	tg := notifications.NewTelegram("123:xxx", cl)

	httpmock.RegisterResponder("GET", "https://api.telegram.org/bot123:xxx/getFile?file_id=xxx-yyy-ggg",
		httpmock.NewStringResponder(200, `{"ok":true,"result":{"file_id":"xxx-yyy-ggg","file_unique_id":"123","file_size":123,"file_path":"photos/file_123.jpg"}}`))

	httpmock.RegisterResponder("GET", "https://api.telegram.org/file/bot123:xxx/photos/file_123.jpg",
		httpmock.NewStringResponder(200, `file content`))

	resp, err :=
		tg.GetFile(context.TODO(), "xxx-yyy-ggg")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "file content", string(resp))
}

func TestSendMessage(t *testing.T) {
	cl := req.DefaultClient()
	httpmock.ActivateNonDefault(cl.GetClient())
	defer httpmock.DeactivateAndReset()

	tg := notifications.NewTelegram("123:xxx", cl)

	httpmock.RegisterResponder("POST", "https://api.telegram.org/bot123:xxx/sendMessage",
		httpmock.NewStringResponder(200, `{"ok":true,"result":{"message_id":123,"from":{"id":123,"is_bot":true,"first_name":"test","username":"test"},"chat":{"id":123,"first_name":"test","username":"test","type":"private"},"date":123,"text":"test"}}`))

	err :=
		tg.SendMessage(context.TODO(), 123, "test")
	assert.NoError(t, err)
}

func TestReact(t *testing.T) {
	cl := req.DefaultClient()
	httpmock.ActivateNonDefault(cl.GetClient())
	defer httpmock.DeactivateAndReset()

	tg := notifications.NewTelegram("123:xxx", cl)

	httpmock.RegisterResponder("POST", "https://api.telegram.org/bot123:xxx/setMessageReaction",
		httpmock.NewStringResponder(200, `{"ok":true,"result":{"message_id":123,"from":{"id":123,"is_bot":true,"first_name":"test","username":"test"},"chat":{"id":123,"first_name":"test","username":"test","type":"private"},"date":123,"text":"test"}}`))
	err :=
		tg.React(context.TODO(), 123, 123, "test")
	assert.NoError(t, err)
}
