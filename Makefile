GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./mock/*")

.PHONY: help
help: ## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/:.*##/:##/' | column -t -s '##'

.PHONY: ci
ci: staticcheck test ## Run all the CI targets

.PHONY: test
test: ## Run the tests
	@go test ./...

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@go tool staticcheck ./...

.PHONY: generate
generate: ## Generates the code generators
	@go generate ./...
