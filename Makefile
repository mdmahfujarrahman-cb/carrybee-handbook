.DEFAULT_GOAL := help
.PHONY: help run build check test fmt clean

help: ## List the available targets
	@grep -E '^[a-z]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

run: ## Serve the handbook with the Sync button at http://127.0.0.1:4321
	@echo "Handbook: http://127.0.0.1:4321"
	go run ./cmd/handbook serve

build: ## Scan the apps and write the static site to dist/
	go run ./cmd/handbook build

check: ## Validate content references and list modules that need a description
	go run ./cmd/handbook check

test: ## Run go vet and the test suite
	go vet ./... && go test -count=1 ./...

fmt: ## Format Go code with gofmt
	gofmt -w .

clean: ## Remove dist, dist.old and .dist-* build folders
	rm -rf dist dist.old .dist-*
