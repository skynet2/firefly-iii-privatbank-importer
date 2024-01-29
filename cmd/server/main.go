package main

import (
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/gorilla/mux"
	"github.com/imroc/req/v3"

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

	db, err := client.NewDatabase(os.Getenv("COSMO_DB_NAME"))
	if err != nil {
		panic(err)
	}

	dataRepo := repo.NewCosmo(db)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()

	tgNotifier := notifications.NewTelegram(
		os.Getenv("TELEGRAM_BOT_TOKEN"),
		req.DefaultClient(),
	)
	processorSvc := processor.NewProcessor(
		dataRepo,
		parser.NewParser(),
		tgNotifier,
	)
	handle := NewHandler(processorSvc)
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
