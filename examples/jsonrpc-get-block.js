#!/usr/bin/env node

/**
 * Minimal JSON-RPC example that fetches the latest block number
 * from a running geth node using only Node.js built-ins.
 *
 * Usage:
 *   node examples/jsonrpc-get-block.js
 *
 * Optionally override RPC URL:
 *   RPC_URL=http://127.0.0.1:8545 node examples/jsonrpc-get-block.js
 */

const http = require("http");

const rpcUrl = process.env.RPC_URL || "http://127.0.0.1:8545";
const { hostname, port, pathname } = new URL(rpcUrl);

const payload = JSON.stringify({
  jsonrpc: "2.0",
  method: "eth_blockNumber",
  params: [],
  id: 1,
});

const options = {
  hostname,
  port,
  path: pathname || "/",
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(payload),
  },
};

const req = http.request(options, (res) => {
  let data = "";

  res.on("data", (chunk) => {
    data += chunk;
  });

  res.on("end", () => {
    try {
      const json = JSON.parse(data);
      console.log("RPC response:", json);
    } catch (e) {
      console.error("Failed to parse response:", e);
      console.error("Raw response:", data);
    }
  });
});

req.on("error", (err) => {
  console.error("Request error:", err);
});

req.write(payload);
req.end();
