GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./mock/*")

.PHONY: help
help: ## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: ci
ci:	lint vet fmt test ## Run all the CI targets

.PHONY: test
test: ## Run the tests
	@go test ./...

.PHONY: vet
vet: ## Run the vet
	@go vet ./...

.PHONY: fmt
fmt: install-goimports ## Run the goimports
	@if [ "$(shell goimports -l $(GO_FILES) | wc -l)" != "0" ]; then \
		echo "--- CHECK FAIL: Some files did not pass goimports $(shell goimports -l $(GO_FILES))"; exit 2; \
	fi

.PHONY: lint
lint: install-lint ## Run the golint
	@go list ./... | xargs golint -set_exit_status

.PHONY: install-lint
install-lint: ## Install the golint
	@go get -u golang.org/x/lint/golint

.PHONY: install-goimports
install-goimports: ## Intall the goimports
	@go get golang.org/x/tools/cmd/goimports
