package firefly_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
)

func TestFirefly(t *testing.T) {
	cl := req.DefaultClient()
	httpmock.ActivateNonDefault(cl.GetClient())
	defer httpmock.DeactivateAndReset()

	apiKey := "test-api-key"

	ff := firefly.NewFirefly(
		apiKey,
		"https://example.com",
		cl,
	)

	httpmock.RegisterResponder(
		"GET",
		"https://example.com/api/v1/accounts",
		func(request *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer "+apiKey, request.Header.Get("Authorization"))
			acc := []*firefly.Account{
				{
					Id: "1",
					Attributes: firefly.AccountAttributes{
						Name: "test-account-1",
					},
				},
				{
					Id: "2",
					Attributes: firefly.AccountAttributes{
						Name: "test-account-2",
					},
				},
			}

			return httpmock.NewJsonResponse(200, firefly.GenericApiResponse[[]*firefly.Account]{
				Data: acc,
			})
		})

	resp, err := ff.ListAccounts(context.TODO())
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.Equal(t, 2, len(resp))
	assert.Equal(t, "1", resp[0].Id)
	assert.Equal(t, "test-account-1", resp[0].Attributes.Name)

	assert.Equal(t, "2", resp[1].Id)
	assert.Equal(t, "test-account-2", resp[1].Attributes.Name)
}
