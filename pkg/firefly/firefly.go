package firefly

import (
	"context"
	"os"

	"github.com/cockroachdb/errors"
	"github.com/imroc/req/v3"
)

type Firefly struct {
	cl         *req.Client
	apiKey     string
	fireflyURL string
}

func NewFirefly(
	apiKey string,
	fireflyURL string,
	cl *req.Client,
) *Firefly {
	return &Firefly{
		cl:         cl,
		fireflyURL: fireflyURL,
		apiKey:     apiKey,
	}
}

func (f *Firefly) ListAccounts(ctx context.Context) ([]*Account, error) {
	var apiResp GenericApiResponse[[]*Account]

	resp, err := f.cl.R().
		SetContext(ctx).
		SetBearerAuthToken(f.apiKey).
		SetSuccessResult(&apiResp).
		EnableDumpTo(os.Stdout).
		SetQueryParam("limit", "100500").
		Get(f.fireflyURL + "/api/v1/accounts")
	if err != nil {
		return nil, err
	}

	if resp.IsErrorState() {
		return nil, errors.Newf("got error response: %s", resp.String())
	}

	return apiResp.Data, nil
}
