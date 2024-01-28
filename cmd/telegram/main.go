package main

import (
	"net/http"
	"os"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/gorilla/mux"
	"github.com/syumai/workers"

	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/parser"
	"github.com/skynet2/firefly-iii-privatbank-importer/pkg/processor"
	repo "github.com/skynet2/firefly-iii-privatbank-importer/pkg/repo"
)

const (
	DbName = "firefly_iii"
)

var apiKey string

func main() {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	dataRepo, err := repo.NewD1(cfg.DbName)
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
	workers.Serve(nil) // use http.DefaultServeMux
}
