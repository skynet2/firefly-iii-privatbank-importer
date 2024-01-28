package main

import (
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/azcosmos"
	"github.com/caarlos0/env/v10"
	"github.com/gorilla/mux"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/repo"
)

var apiKey string

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}

	endpoint := "todo"
	client, err := azcosmos.NewClient(endpoint, credential, &azcosmos.ClientOptions{
		EnableContentResponseOnWrite: true,
	})
	if err != nil {
		panic(err)
	}

	db, err := client.NewDatabase(cfg.DbName)
	if err != nil {
		panic(err)
	}

	dataRepo := repo.NewCosmo(db)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()

	processorSvc := processor.NewProcessor(dataRepo, parser.NewParser())
	handle := NewHandler(processorSvc)
	r.Handle("/hook", handle)

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
