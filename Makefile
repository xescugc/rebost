.PHONY: help 
help: ## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: ci
ci:	lint vet test 	## Run all the CI targets

.PHONY: test
test:
	@go test ./...

.PHONY: vet
vet:
	@go vet ./...

.PHONY: lint
lint: install-lint
	@go list ./... | xargs golint -set_exit_status

.PHONY: install-lint
install-lint:
	@go get -u golang.org/x/lint/golint
