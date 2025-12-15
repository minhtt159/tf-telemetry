# TF Telemetry

TF Telemetry exposes gRPC and HTTP endpoints to collect metrics and logs that are forwarded to Elasticsearch.

## Build and test

```bash
make build   # builds ./bin/tf-telemetry
make test    # runs go test ./...
```

## Container image

Build the image and run it locally:

```bash
docker build -f build/Dockerfile -t tf-telemetry:local .
docker run --rm -p 8080:8080 -p 50051:50051 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  tf-telemetry:local
```

The HTTP endpoint accepts telemetry at `POST /v1/telemetry` and basic health is available at `/healthz`. Enable basic auth in `config.yaml` and provide credentials with `curl -u user:pass ...` when needed.
