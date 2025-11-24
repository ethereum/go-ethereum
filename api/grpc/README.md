# gRPC Trading API

Low-latency binary API for high-frequency trading and MEV operations on Mandarin (Geth fork).

## Overview

This package provides a Protocol Buffer-based gRPC API that replaces slow JSON-RPC for performance-critical operations. Key features:

- **Binary protocol**: 10x performance improvement over JSON-RPC
- **Bundle operations**: Submit and simulate transaction bundles with profit calculation
- **Batch operations**: Read multiple storage slots in a single call
- **Low latency**: Direct integration with miner and state database
- **Type safety**: Strongly-typed interfaces via Protocol Buffers

## Architecture

- `trader.proto`: Protocol Buffer definitions
- `trader.pb.go` / `trader_grpc.pb.go`: Generated Go code
- `server.go`: gRPC server implementation
- `service.go`: Node lifecycle integration
- `example_client.go`: Client library and usage examples

## Configuration

Enable gRPC in your node configuration:

```toml
[Eth]
EnableGRPC = true
GRPCHost = "localhost"
GRPCPort = 9090
```

Or via command-line flags:

```bash
geth --grpc --grpc.addr localhost --grpc.port 9090
```

## Usage Example

```go
import grpcapi "github.com/ethereum/go-ethereum/api/grpc"

// Connect to gRPC server
client, err := grpcapi.NewClient("localhost", 9090)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Batch read Uniswap pool state
poolAddr := common.HexToAddress("0x...")
slots := []common.Hash{
    common.HexToHash("0x0"), // slot0
    common.HexToHash("0x1"), // feeGrowthGlobal0X128
}
values, err := client.GetStorageBatch(ctx, poolAddr, slots, nil)

// Simulate bundle
simResult, err := client.SimulateBundle(ctx, txs, &grpcapi.BundleOptions{
    TargetBlock: &blockNum,
})

profit := new(big.Int).SetBytes(simResult.Profit)
fmt.Printf("Bundle profit: %s wei\n", profit)

// Submit if profitable
if profit.Sign() > 0 {
    bundleHash, err := client.SubmitBundle(ctx, txs, opts)
    fmt.Printf("Bundle submitted: %s\n", bundleHash.Hex())
}
```

## API Methods

### SimulateBundle
Simulate transaction bundle execution without submitting to mempool.

**Request:**
- `transactions`: RLP-encoded transactions
- `min_timestamp`, `max_timestamp`: Optional timing constraints
- `target_block`: Specific block number target
- `reverting_txs`: Indices of transactions allowed to revert

**Response:**
- `success`: Whether all transactions succeeded
- `gas_used`: Total gas consumed
- `profit`: Coinbase profit in wei
- `coinbase_balance`: Final coinbase balance
- `tx_results`: Per-transaction results with gas, errors, return values

### SubmitBundle
Submit bundle for inclusion in future blocks.

**Request:** Same as SimulateBundle

**Response:**
- `bundle_hash`: Unique bundle identifier

### GetStorageBatch
Read multiple storage slots in a single call. **10-100x faster** than multiple `eth_getStorageAt` calls.

**Request:**
- `contract`: Contract address
- `slots`: Array of storage slot hashes
- `block_number`: Optional block height

**Response:**
- `values`: Array of storage values (same order as request)

### GetPendingTransactions
Retrieve pending transactions from mempool.

**Request:**
- `min_gas_price`: Optional filter for minimum gas price

**Response:**
- `transactions`: Array of RLP-encoded transactions

### CallContract
Execute contract call (equivalent to `eth_call`).

**Request:**
- `from`, `to`, `data`: Standard call parameters
- `gas`, `gas_price`, `value`: Optional parameters
- `block_number`: Optional block height

**Response:**
- `return_data`: Call result
- `gas_used`: Gas consumed
- `success`: Whether call succeeded
- `error`: Error message if failed

## Performance Characteristics

Based on Phase 2 benchmarks:

- **Transaction feed latency**: ~2.5μs average, 21μs max
- **Storage batch reads**: 10-100x faster than JSON-RPC (single call vs N round-trips)
- **Bundle simulation**: Direct EVM access, no JSON marshaling overhead
- **gRPC overhead**: ~100-500μs vs 1-5ms for JSON-RPC

For a 10-transaction bundle simulation:
- JSON-RPC: ~50ms (encoding + network + decoding)
- gRPC: ~5ms (binary encoding, single RTT)

## Development

### Regenerating Protocol Buffers

If you modify `trader.proto`:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       api/grpc/trader.proto
```

### Testing

Integration tests require a running node:

```bash
# Terminal 1: Start node with gRPC enabled
geth --dev --grpc --grpc.port 9090

# Terminal 2: Run tests
go test -v ./api/grpc/...
```

### Benchmarking

Compare gRPC vs JSON-RPC performance:

```bash
go test -bench=. -benchtime=10s ./api/grpc/...
```

## Security Considerations

- **Network exposure**: gRPC server should only listen on localhost or trusted networks
- **Authentication**: Add mTLS or API keys for production deployments
- **Rate limiting**: Not currently implemented, add as needed
- **Bundle privacy**: Bundles are visible to node operator

## Future Enhancements

1. **Streaming APIs**: Real-time transaction feed via server-side streaming
2. **Hot state cache**: Dedicated cache for frequently-accessed DeFi contracts
3. **Parallel simulation**: Multiple bundle simulations in concurrent goroutines
4. **State deltas**: Export compact state diffs instead of full state
5. **Shared memory**: Zero-copy data sharing for co-located bots
6. **Authentication**: mTLS, JWT, or API key support
7. **Metrics**: Prometheus integration for latency and throughput monitoring

## License

LGPL-3.0-or-later (same as go-ethereum)

