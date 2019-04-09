.DEFAULT_GOAL := help

tests: ## Run tests with coverage
	go test -v -cover ./...

profile_tests: ## Run tests and output coverage profiling
	go test -v -coverprofile=coverage.out .
	go tool cover -html=coverage.out

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
