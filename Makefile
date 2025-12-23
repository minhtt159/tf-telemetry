BINARY ?= tf-telemetry
BUILD_DIR ?= bin
SWAGGER_DIR ?= docs/swagger

.PHONY: build test lint dev clean swagger proto docker-build docker-dev help

## Build
build:
	@mkdir -p $(BUILD_DIR)
	GOTOOLCHAIN=auto go build -o $(BUILD_DIR)/$(BINARY) ./cmd/app

## Test
test:
	GOTOOLCHAIN=auto go test ./...

## Lint (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## Development server with Air hot reload
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Run: go install github.com/air-verse/air@latest"; \
	fi

## Generate Swagger documentation
swagger:
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/app/main.go -o $(SWAGGER_DIR) --parseDependency --parseInternal; \
	else \
		echo "swag not installed. Run: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

## Generate protobuf files
proto:
	@if command -v protoc >/dev/null 2>&1; then \
		protoc --go_out=. --go_opt=paths=source_relative \
			--go-grpc_out=. --go-grpc_opt=paths=source_relative \
			-I api/proto api/proto/*.proto; \
	else \
		echo "protoc not installed"; \
	fi

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR) tmp $(SWAGGER_DIR)

## Docker build (production)
docker-build:
	docker build -f build/Dockerfile -t $(BINARY):local .

## Docker development with hot reload
docker-dev:
	docker compose up --build

## Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests"
	@echo "  lint         - Run linter"
	@echo "  dev          - Start development server with Air"
	@echo "  swagger      - Generate Swagger documentation"
	@echo "  proto        - Generate protobuf files"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-dev   - Start Docker development environment"
	@echo "  help         - Show this help"
