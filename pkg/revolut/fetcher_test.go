package revolut_test

import (
	"context"
	_ "embed"
	"net/http"
	"testing"
	"time"

	"github.com/imroc/req/v3"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/revolut"
)

//go:embed testdata/last.json
var last string

//go:embed testdata/last2.json
var last2 string

//go:embed testdata/last3.json
var last3 string

//go:embed testdata/last4.json
var last4 string

func TestFetch(t *testing.T) {
	cl := req.DefaultClient()

	httpmock.ActivateNonDefault(cl.GetClient())
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://app.revolut.com/api/retail/user/current/transactions/last?count=50",
		func(request *http.Request) (*http.Response, error) {

			return httpmock.NewStringResponse(200, last), nil
		})

	httpmock.RegisterResponder("GET", "https://app.revolut.com/api/retail/user/current/transactions/last?count=50&to=1712138204748",
		func(request *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, last2), nil
		})

	httpmock.RegisterResponder("GET", "https://app.revolut.com/api/retail/user/current/transactions/last?count=50&to=1703158638000",
		func(request *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, last3), nil
		})

	httpmock.RegisterResponder("GET", "https://app.revolut.com/api/retail/user/current/transactions/last?count=50&to=1117407790804",
		func(request *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, last4), nil
		})

	fetcher := revolut.NewFetcher(cl)

	resp, err := fetcher.Fetch(context.TODO(), &revolut.FetchRequest{
		Cookies: "abcd",
		After:   time.Now().AddDate(-1, 0, 0),
	})
	assert.NoError(t, err)

	assert.Len(t, resp, 7)
}
