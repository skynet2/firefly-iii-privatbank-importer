.PHONY: build
build:
	@cd cmd/server && rm -rf dist && GOOS=linux CGO_ENABLED=0 go build -o dist/handler

.PHONY: azure-deploy
azure-deploy: build
	@cd cmd/server/.azure && cp -a . ../dist/


.PHONY: generate
generate:
	go generate ./...

.PHONY: deploy-balances
deploy-balances:
	@docker build -t skydev/firefly-balances:0.0.1 -f images/balances/Dockerfile .

.PHONY: lint
lint: generate
	golangci-lint run