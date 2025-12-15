# Stage 1: Build
FROM golang:1.22-alpine AS builder

ENV GOTOOLCHAIN=auto
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically
RUN CGO_ENABLED=0 GOOS=linux go build -o tf-telemetry ./cmd/app

# Stage 2: Run
FROM gcr.io/distroless/static-debian12

WORKDIR /app
COPY --from=builder /app/tf-telemetry .
# Copy config (optional default, usually mounted)
COPY config.yaml .

EXPOSE 50051 8080

CMD ["./tf-telemetry"]
