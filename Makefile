.PHONY: lint
lint: ## Run linters
	golangci-lint run

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: generate
generate: ## Generate code
	go generate ./...

.DEFAULT_GOAL := help

.PHONY: help
help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'