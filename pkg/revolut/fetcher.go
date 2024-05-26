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

func (f *Fetcher) Fetch(ctx context.Context, req *FetchRequest) error {
	to := int64(0)
	var resultTxs []*transaction

	for ctx.Err() == nil {
		httpReq := f.cl.NewRequest().
			SetURL("https://app.revolut.com/api/retail/user/current/transactions/last?count=50")

		if to > 0 {
			httpReq = httpReq.SetQueryParam("to", fmt.Sprint(to))
		}

		resp := httpReq.Do(ctx)
		if resp.Err != nil {
			return resp.Err
		}

		if resp.IsErrorState() {
			return fmt.Errorf("error response: %v and body %s", resp.StatusCode, resp.String())
		}

		var txs []*transaction
		if err := resp.UnmarshalJson(&txs); err != nil {
			return err
		}

		if len(txs) == 0 {
			break
		}

		var addedTxs []*transaction
		for _, tx := range txs {
			parsedDate := time.Unix(tx.CreatedDate, 0)

			if parsedDate.After(req.After) {
				addedTxs = append(addedTxs, tx)
			}
		}

		if len(addedTxs) == 0 {
			break
		}

		resultTxs = append(resultTxs, addedTxs...)

		sort.Slice(addedTxs, func(i, j int) bool {
			return addedTxs[i].CreatedDate > addedTxs[j].CreatedDate
		})

		to = addedTxs[len(addedTxs)-1].CreatedDate
	}
	return nil
}
