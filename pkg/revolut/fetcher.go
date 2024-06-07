package revolut

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/imroc/req/v3"
)

type Fetcher struct {
	cl *req.Client
}

func NewFetcher(
	client *req.Client,
) *Fetcher {
	return &Fetcher{
		cl: client,
	}
}

func (f *Fetcher) Fetch(ctx context.Context, req *FetchRequest) ([]*Transaction, error) {
	to := int64(0)
	var resultTxs []*Transaction

	for ctx.Err() == nil {
		httpReq := f.cl.NewRequest().
			SetHeader("X-Browser-Application", "WEB_CLIENT").
			SetHeader("X-Client-Version", "100.0").
			SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36").
			SetHeader("Referer", "https://app.revolut.com/transactions").
			SetHeader("X-Device-Id", req.DeviceID).
			SetHeader("Cookies", req.Cookies).
			SetURL("https://app.revolut.com/api/retail/user/current/transactions/last?count=50")

		if to > 0 {
			httpReq = httpReq.SetQueryParam("to", fmt.Sprint(to))
		}

		resp := httpReq.Do(ctx)
		if resp.Err != nil {
			return nil, resp.Err
		}

		if resp.IsErrorState() {
			return nil, fmt.Errorf("error response: %v and body %s", resp.StatusCode, resp.String())
		}

		var txs []*Transaction
		if err := resp.UnmarshalJson(&txs); err != nil {
			return nil, err
		}

		if len(txs) == 0 {
			break
		}

		var addedTxs []*Transaction
		for _, tx := range txs {
			parsedDate := time.UnixMilli(tx.StartedDate)

			if parsedDate.After(req.After) {
				addedTxs = append(addedTxs, tx)
			}
		}

		if len(addedTxs) == 0 {
			break
		}

		resultTxs = append(resultTxs, addedTxs...)

		sort.Slice(txs, func(i, j int) bool {
			return txs[i].StartedDate > txs[j].StartedDate
		})

		to = txs[len(txs)-1].StartedDate
	}

	return resultTxs, nil
}
