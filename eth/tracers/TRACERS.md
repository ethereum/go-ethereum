# Ethereum Go Client Tracers

This document provides additional details about the tracers available in the Ethereum Go client.

## Prestate Tracer

The prestateTracer captures the state of accounts before transaction execution. It provides a detailed view of account states including balances, nonces, code, and storage.

### Important Behaviors

1. **Complete Account Capture**: When an account is accessed, the tracer captures its complete state by default, including code, even if only specific fields like balance were accessed during execution.

2. **Configuration Options**:
   - `diffMode`: If true, returns state modifications
   - `disableCode`: If true, excludes contract code from the output (useful to reduce response size or when code isn't needed)
   - `disableStorage`: If true, excludes contract storage from the output

### Example Usage with CLI tools

```bash
# Include code (default behavior)
cast rpc debug_traceCall \
  '{"to":"$CONTRACT_ADDR","data":"$DATA"}' \
  latest \
  '{"tracer":"prestateTracer", "tracerConfig": { "diffMode": false } }'

# Exclude code
cast rpc debug_traceCall \
  '{"to":"$CONTRACT_ADDR","data":"$DATA"}' \
  latest \
  '{"tracer":"prestateTracer", "tracerConfig": { "diffMode": false, "disableCode": true } }'
