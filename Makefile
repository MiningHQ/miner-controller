#
# A Makefile to build, run and test Go code
#

.PHONY: default build fmt lint run run_race test clean vet docker_build docker_run docker_clean

# This makes the APP_NAME be the name of the current directory
# Ex. in path /home/dev/app/awesome-app the APP_NAME will be set to awesome-app
APP_NAME := $(notdir $(CURDIR))

default: build ## Build the binary

build: ## Build the binary
	go build -o ./bin/${APP_NAME} ./src/*.go

run: build ## Build and run the binary
	# Add your environment variable here
	LOG_LEVEL=debug \
	LOG_FORMAT=text \
	./bin/${APP_NAME}

run_race: ## Run the binary with race condition checking enabled
	# Add your environment variable here
	LOG_LEVEL=debug \
	LOG_FORMAT=text \
	go run -race ./src/*.go

fmt: ## Format the code using `go fmt`
	go fmt ./...

test: ## Run the tests
	go test ./...

test_cover: ## Run tests with a coverage report
	go test ./... -v -cover -covermode=count -coverprofile=./coverage.out

clean: ## Remove compiled binaries from bin/
	rm ./bin/*

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
