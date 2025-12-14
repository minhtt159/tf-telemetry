#!/bin/bash
ES_HOST="http://localhost:9200"

echo "Creating Metrics Index..."
curl -X PUT "$ES_HOST/mobile-metrics-v1" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date", "format": "epoch_millis" },
      "platform": { "type": "keyword" },
      "customer_id": { "type": "integer" },
      "app_version": { "type": "keyword" },
      "installation_id": { "type": "keyword" },
      "journey_id": { "type": "keyword" },
      "network": { "type": "keyword" },
      "cpu_usage": { "type": "scaled_float", "scaling_factor": 100 },
      "battery_level": { "type": "scaled_float", "scaling_factor": 100 }
    }
  }
}'

echo "\nCreating Logs Index..."
curl -X PUT "$ES_HOST/mobile-logs-v1" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date", "format": "epoch_millis" },
      "platform": { "type": "keyword" },
      "level": { "type": "keyword" },
      "tag": { "type": "keyword" },
      "message": { "type": "text" },
      "stack_trace": { "type": "text" },
      "error_code": { "type": "integer" },
      "attributes": { "type": "object" }
    }
  }
}'
