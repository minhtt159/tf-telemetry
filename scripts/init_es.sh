#!/bin/bash
ES_HOST="${ES_HOST:-http://localhost:9200}"

echo "Creating Metrics Index..."
curl -X PUT "$ES_HOST/mobile-metrics-v1" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date", "format": "epoch_millis" },
      "platform": { "type": "keyword" },
      "installation_id": { "type": "keyword" },
      "journey_id": { "type": "keyword" },
      "sdk_version": { "type": "keyword" },
      "host_app_name": { "type": "keyword" },
      "host_app_version": { "type": "keyword" },
      "network": { "type": "keyword" },
      "battery_level": { "type": "scaled_float", "scaling_factor": 100 },
      "device_hardware": {
        "type": "object",
        "properties": {
          "physical_cores": { "type": "integer" },
          "logical_cpus": { "type": "integer" },
          "l1_cache_kb": { "type": "integer" },
          "l2_cache_kb": { "type": "integer" },
          "l3_cache_kb": { "type": "integer" },
          "total_physical_bytes": { "type": "long" }
        }
      },
      "cpu": {
        "type": "object",
        "properties": {
          "total_usage_percent": { "type": "scaled_float", "scaling_factor": 100 },
          "core_usage_percent": { "type": "scaled_float", "scaling_factor": 100 }
        }
      },
      "memory": {
        "type": "object",
        "properties": {
          "app_resident_bytes": { "type": "long" },
          "app_virtual_bytes": { "type": "long" },
          "system_free_bytes": { "type": "long" },
          "system_active_bytes": { "type": "long" },
          "system_inactive_bytes": { "type": "long" },
          "system_wired_bytes": { "type": "long" }
        }
      }
    }
  }
}'

echo -e "\nCreating Logs Index..."
curl -X PUT "$ES_HOST/mobile-logs-v1" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "timestamp": { "type": "date", "format": "epoch_millis" },
      "platform": { "type": "keyword" },
      "installation_id": { "type": "keyword" },
      "journey_id": { "type": "keyword" },
      "sdk_version": { "type": "keyword" },
      "host_app_name": { "type": "keyword" },
      "host_app_version": { "type": "keyword" },
      "network": { "type": "keyword" },
      "level": { "type": "keyword" },
      "tag": { "type": "keyword" },
      "message": { "type": "text" },
      "stack_trace": { "type": "text" },
      "context": {
        "type": "object",
        "enabled": true,
        "dynamic": true
      }
    }
  }
}'

echo -e "\nDone!"
