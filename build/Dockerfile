# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Build statically
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o tf-telemetry ./cmd/app

# Stage 2: Run
FROM gcr.io/distroless/static-debian12:nonroot

# Image labels (OCI standard)
LABEL org.opencontainers.image.title="tf-telemetry"
LABEL org.opencontainers.image.description="Telemetry collection server for mobile and web applications"
LABEL org.opencontainers.image.url="https://github.com/minhtt159/tf-telemetry"
LABEL org.opencontainers.image.source="https://github.com/minhtt159/tf-telemetry"
LABEL org.opencontainers.image.vendor="hnimn.art"
LABEL org.opencontainers.image.authors="minhtt159 <4847344+minhtt159@users.noreply.github.com>"
LABEL org.opencontainers.image.licenses="MIT"

WORKDIR /app

# Copy from builder
COPY --from=builder /app/tf-telemetry .
# Copy config (optional default, usually mounted)
COPY --chown=nonroot:nonroot config.yaml .

# Use non-root user (distroless:nonroot uses UID 65532)
USER nonroot:nonroot

EXPOSE 50051 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 CMD ["./tf-telemetry", "healthcheck"]

# Read-only root filesystem recommendation - handled at runtime via docker run --read-only
CMD ["./tf-telemetry"]
