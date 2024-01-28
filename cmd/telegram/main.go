package main

import (
	"net/http"

	"github.com/caarlos0/env/v10"
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

	processorSvc := processor.NewProcessor(dataRepo, parser.NewParser())
	handle := NewHandler(processorSvc)
	http.Handle("/hook", handle)
	workers.Serve(nil) // use http.DefaultServeMux
}
