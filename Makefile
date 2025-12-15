BINARY ?= tf-telemetry
BUILD_DIR ?= bin

.PHONY: build test docker-build

build:
	@mkdir -p $(BUILD_DIR)
	GOTOOLCHAIN=auto go build -o $(BUILD_DIR)/$(BINARY) ./cmd/app

test:
	GOTOOLCHAIN=auto go test ./...

docker-build:
	docker build -f build/Dockerfile -t $(BINARY):local .
