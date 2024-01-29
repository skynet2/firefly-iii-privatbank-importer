.PHONY: build
build:
	@cd cmd/server && rm -rf dist && GOOS=linux go build -o dist/handler

.PHONY: azure-deploy
azure-deploy: build
	@cd cmd/server/githubmonkey/.azure && cp -a . ../dist/