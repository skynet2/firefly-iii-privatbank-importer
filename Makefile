.PHONY: build
build:
	@cd cmd/server && rm -rf dist && GOOS=linux CGO_ENABLED=0 go build -o dist/handler

.PHONY: azure-deploy
azure-deploy: build
	@cd cmd/server/.azure && cp -a . ../dist/