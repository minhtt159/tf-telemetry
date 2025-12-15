# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically
RUN CGO_ENABLED=0 GOOS=linux go build -o tf-telemetry ./cmd/app

# Stage 2: Run
# Distroless is preferred over scratch to retain CA certificates for Elasticsearch TLS.
FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /app/tf-telemetry .
# Copy config (optional default, usually mounted)
COPY config.yaml .

EXPOSE 50051 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["./tf-telemetry", "healthcheck"]

CMD ["./tf-telemetry"]
