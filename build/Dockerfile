# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install protoc (if you generate inside docker)
# For this example, we assume generated .pb.go files are already present or committed
# If not, install protoc and protoc-gen-go here.

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically
RUN CGO_ENABLED=0 GOOS=linux go build -o observability-server ./cmd/app

# Stage 2: Run
FROM busybox:latest

WORKDIR /app
COPY --from=builder /app/observability-server .
# Copy config (optional default, usually mounted)
COPY config.yaml . 

EXPOSE 50051 8080

CMD ["./observability-server"]
