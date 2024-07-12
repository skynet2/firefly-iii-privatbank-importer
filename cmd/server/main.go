package main

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/gorilla/mux"
	"github.com/imroc/req/v3"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/database"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/firefly"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/notifications"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/repo"
)

var apiKey string

func main() {
	client, err := azcosmos.NewClientFromConnectionString(os.Getenv("COSMO_DB_CONNECTION_STRING"), nil)
	if err != nil {
		panic(err)
	}

	var fireflyAdditionalHeaders map[string]string
	if v, ok := os.LookupEnv("FIREFLY_ADDITIONAL_HEADERS"); ok {
		if err = json.Unmarshal([]byte(v), &fireflyAdditionalHeaders); err != nil {
			panic(err)
		}
	}

	httpClient := req.DefaultClient()
	fireflyClient := firefly.NewFirefly(
		os.Getenv("FIREFLY_TOKEN"),
		os.Getenv("FIREFLY_URL"),
		httpClient,
		fireflyAdditionalHeaders,
	)

	chatMap := map[string]database.TransactionSource{}
	if err = json.Unmarshal([]byte(os.Getenv("CHAT_MAP")), &chatMap); err != nil {
		panic(err)
	}

	dataRepo, err := repo.NewCosmo(client, os.Getenv("COSMO_DB_NAME"))
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()

	tgNotifier := notifications.NewTelegram(
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		httpClient,
	)

	parserConfig := &processor.Config{
		Repo:            dataRepo,
		Parsers:         map[database.TransactionSource]processor.Parser{},
		NotificationSvc: tgNotifier,
		FireflySvc:      fireflyClient,
	}

	for _, p := range []processor.Parser{
		parser.NewParser(),
		parser.NewParibas(),
		parser.NewZen(),
	} {
		parserConfig.Parsers[p.Type()] = p
	}

	processorSvc := processor.NewProcessor(parserConfig)
	handle := NewHandler(processorSvc, chatMap)
	r.Handle("/api/github/webhook", handle)

	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         listenAddr,
		WriteTimeout: 60 * time.Second,
		ReadTimeout:  60 * time.Second,
	}

	panic(srv.ListenAndServe())
}
