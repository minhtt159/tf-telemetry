// Telemetry JavaScript Library
// Handles sending metrics and logs to the telemetry server
// with localStorage queue for offline support

const telemetry = (function () {
  "use strict";

  const STORAGE_KEY = "telemetry_queue";
  const MAX_QUEUE_SIZE = 100;

  // Generate UUID v7
  function generateUUIDv7String() {
    return "tttttttt-tttt-7xxx-yxxx-xxxxxxxxxxxx"
      .replace(/[xy]/g, function (c) {
        const r = Math.trunc(Math.random() * 16);
        const v = c == "x" ? r : (r & 0x3) | 0x8;
        return v.toString(16);
      })
      .replace(/^[t]{8}-[t]{4}/, function () {
        const unixtimestamp = Date.now().toString(16).padStart(12, "0");
        return unixtimestamp.slice(0, 8) + "-" + unixtimestamp.slice(8);
      });
  }

  // Convert UUID string to base64 bytes (for protobuf)
  function uuidStringToBase64(uuidStr) {
    const hex = uuidStr.replace(/-/g, "");
    const bytes = new Uint8Array(
      hex.match(/.{1,2}/g).map((byte) => parseInt(byte, 16)),
    );
    return btoa(String.fromCharCode.apply(null, bytes));
  }

  // Generate client metadata (with bytes for server, string for display)
  function generateMetadata() {
    const installationIdStr =
      localStorage.getItem("installation_id") || generateUUIDv7String();
    localStorage.setItem("installation_id", installationIdStr);

    const journeyIdStr =
      sessionStorage.getItem("journey_id") || generateUUIDv7String();
    sessionStorage.setItem("journey_id", journeyIdStr);

    // Log installation ID for debugging/identification purposes
    console.log("Installation ID:", installationIdStr);

    return {
      platform: "WEB",
      installation_id: uuidStringToBase64(installationIdStr),
      journey_id: uuidStringToBase64(journeyIdStr),
      sdk_version_packed: 10001, // version 1.0.1
      host_app_version: "1.0.0",
      host_app_name: "telemetry-demo",
      device_hardware: {
        physical_cores: navigator.hardwareConcurrency || 4,
        logical_cpus: navigator.hardwareConcurrency || 4,
        l1_cache_kb: 32,
        l2_cache_kb: 256,
        l3_cache_kb: 8192,
        total_physical_memory: performance.memory
          ? performance.memory.jsHeapSizeLimit
          : 2147483648,
      },
    };
  }

  // Detect network type
  function getNetworkType() {
    if (!navigator.onLine) {
      return "NET_OFFLINE";
    }

    // Use Network Information API if available
    const connection =
      navigator.connection ||
      navigator.mozConnection ||
      navigator.webkitConnection;
    if (connection) {
      const type = connection.effectiveType;
      if (type === "4g") return "NET_CELLULAR_4G";
      if (type === "3g") return "NET_CELLULAR_3G";
      if (type === "2g") return "NET_CELLULAR_2G";
      if (type === "slow-2g") return "NET_CELLULAR_2G";
    }

    return "NET_WIFI"; // Default assumption for web
  }

  // Get battery level if available
  async function getBatteryLevel() {
    if ("getBattery" in navigator) {
      try {
        const battery = await navigator.getBattery();
        return battery.level * 100;
      } catch (e) {
        return 100; // Default
      }
    }
    return 100;
  }

  // Generate CPU usage (simulated for web)
  function generateCpuDetail() {
    const cores = navigator.hardwareConcurrency || 4;
    const coreUsages = [];
    for (let i = 0; i < cores; i++) {
      coreUsages.push(Math.random() * 100);
    }

    return {
      total_usage_percent: coreUsages.reduce((a, b) => a + b, 0) / cores,
      core_usage_percent: coreUsages,
    };
  }

  // Generate memory detail
  function generateMemoryDetail() {
    const mem = {
      app_resident_bytes: 0,
      app_virtual_bytes: 0,
      system_free_bytes: 0,
      system_active_bytes: 0,
      system_inactive_bytes: 0,
      system_wired_bytes: 0,
    };

    if (performance.memory) {
      mem.app_resident_bytes = performance.memory.usedJSHeapSize;
      mem.app_virtual_bytes = performance.memory.totalJSHeapSize;
      mem.system_free_bytes =
        performance.memory.jsHeapSizeLimit - performance.memory.usedJSHeapSize;
      mem.system_active_bytes = performance.memory.usedJSHeapSize;
    } else {
      // Simulated values
      mem.app_resident_bytes = Math.floor(Math.random() * 100000000) + 50000000;
      mem.app_virtual_bytes = Math.floor(Math.random() * 200000000) + 100000000;
      mem.system_free_bytes =
        Math.floor(Math.random() * 1000000000) + 500000000;
      mem.system_active_bytes =
        Math.floor(Math.random() * 500000000) + 200000000;
    }

    return mem;
  }

  // Generate metrics payload
  async function generateMetricsPayload() {
    const batteryLevel = await getBatteryLevel();

    return {
      points: [
        {
          client_timestamp_ms: Date.now(),
          network_type: getNetworkType(),
          battery_level_percent: batteryLevel,
          cpu: generateCpuDetail(),
          memory: generateMemoryDetail(),
        },
      ],
    };
  }

  // Generate logs payload
  function generateLogsPayload() {
    const logLevels = ["DEBUG", "INFO", "WARN", "ERROR"];
    const tags = ["network", "ui", "auth", "storage", "analytics"];
    const messages = [
      "User action completed successfully",
      "Network request to API endpoint",
      "Cache hit for resource",
      "Component rendered",
      "State updated",
      "Authentication token refreshed",
      "Data synced to server",
      "Performance metric recorded",
    ];

    const entries = [];
    const numEntries = Math.floor(Math.random() * 3) + 1; // 1-3 log entries

    for (let i = 0; i < numEntries; i++) {
      const level = logLevels[Math.floor(Math.random() * logLevels.length)];
      const entry = {
        client_timestamp_ms: Date.now() - Math.floor(Math.random() * 1000),
        network_type: getNetworkType(),
        level: level,
        tag: tags[Math.floor(Math.random() * tags.length)],
        message: messages[Math.floor(Math.random() * messages.length)],
        context: {
          user_agent: navigator.userAgent,
          url: window.location.href,
          screen_width: window.screen.width.toString(),
          screen_height: window.screen.height.toString(),
        },
      };

      // Add stack trace for ERROR level
      if (level === "ERROR") {
        entry.stack_trace = new Error().stack || "No stack trace available";
      }

      entries.push(entry);
    }

    return { entries };
  }

  // Send telemetry packet to server
  // Note: This uses HTTP/JSON. For native gRPC (port 50051), use a proper gRPC-web client
  async function sendTelemetryPacket(serverUrl, username, password, packet) {
    const endpoint = serverUrl + "/v1/telemetry";

    const headers = {
      "Content-Type": "application/json",
      "Access-Control-Request-Private-Network": "true",
    };

    // Add basic auth if credentials provided
    if (username && password) {
      const credentials = btoa(username + ":" + password);
      headers["Authorization"] = "Basic " + credentials;
    }

    const body = JSON.stringify(packet);
    const bodySize = new TextEncoder().encode(body).length;

    try {
      const response = await fetch(endpoint, {
        method: "POST",
        headers: headers,
        body: body,
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(
          `Server responded with ${response.status}: ${errorText}`,
        );
      }

      return {
        success: true,
        message: "Telemetry sent successfully!",
        packetSize: bodySize,
      };
    } catch (error) {
      // Queue packet for retry if server is unavailable
      queuePacket(packet);
      throw error;
    }
  }

  // Queue packet in localStorage
  function queuePacket(packet) {
    try {
      let queue = JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");

      // Limit queue size
      if (queue.length >= MAX_QUEUE_SIZE) {
        queue.shift(); // Remove oldest
      }

      queue.push({
        packet: packet,
        timestamp: Date.now(),
        retries: 0,
      });

      localStorage.setItem(STORAGE_KEY, JSON.stringify(queue));
    } catch (e) {
      console.error("Failed to queue packet:", e);
    }
  }

  // Get queued packets
  function getQueuedPackets() {
    try {
      return JSON.parse(localStorage.getItem(STORAGE_KEY) || "[]");
    } catch (e) {
      console.error("Failed to get queued packets:", e);
      return [];
    }
  }

  // Retry queued packets
  async function retryQueuedPackets(serverUrl, username, password) {
    const queue = getQueuedPackets();

    if (queue.length === 0) {
      return {
        success: true,
        message: "Queue is empty, nothing to retry",
      };
    }

    let successCount = 0;
    let failCount = 0;
    const newQueue = [];

    for (const item of queue) {
      try {
        await sendTelemetryPacket(serverUrl, username, password, item.packet);
        successCount++;
      } catch (error) {
        item.retries = (item.retries || 0) + 1;
        // Keep in queue if less than 5 retries
        if (item.retries < 5) {
          newQueue.push(item);
        }
        failCount++;
      }
    }

    localStorage.setItem(STORAGE_KEY, JSON.stringify(newQueue));

    return {
      success: successCount > 0,
      message: `Retry complete: ${successCount} sent, ${failCount} failed, ${newQueue.length} remaining in queue`,
    };
  }

  // Clear queue
  function clearQueue() {
    localStorage.removeItem(STORAGE_KEY);
  }

  // Public API
  return {
    generateMetadata,
    generateMetricsPayload,
    generateLogsPayload,
    getQueuedPackets,
    clearQueue,

    async sendMetrics(serverUrl, username, password) {
      const packet = {
        schema_version: 1,
        metadata: generateMetadata(),
        metrics: await generateMetricsPayload(),
      };

      try {
        return await sendTelemetryPacket(serverUrl, username, password, packet);
      } catch (error) {
        return {
          success: false,
          message: `Failed to send metrics: ${error.message}. Packet queued for retry.`,
        };
      }
    },

    async sendLogs(serverUrl, username, password) {
      const packet = {
        schema_version: 1,
        metadata: generateMetadata(),
        logs: generateLogsPayload(),
      };

      try {
        return await sendTelemetryPacket(serverUrl, username, password, packet);
      } catch (error) {
        return {
          success: false,
          message: `Failed to send logs: ${error.message}. Packet queued for retry.`,
        };
      }
    },

    async sendBoth(serverUrl, username, password) {
      const packet = {
        schema_version: 1,
        metadata: generateMetadata(),
        metrics: await generateMetricsPayload(),
        logs: generateLogsPayload(),
      };

      try {
        return await sendTelemetryPacket(serverUrl, username, password, packet);
      } catch (error) {
        return {
          success: false,
          message: `Failed to send telemetry: ${error.message}. Packet queued for retry.`,
        };
      }
    },

    retryQueuedPackets,
  };
})();
