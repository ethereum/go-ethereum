# Mandarin Roadmap

## Objective
Transform Mandarin into a low-latency execution client optimized for co-located trading operations, targeting microsecond-scale improvements over standard Geth.

## Differentiation
- **Erigon**: Database efficiency, archive node performance, disk I/O optimization
- **Mandarin**: Real-time latency optimization for live trading and MEV operations

## Phase 1: Foundation (2-4 weeks)

### Custom Transaction Ordering
- Modify `miner/worker.go` to accept external ordering functions
- Add bundle support with revert protection
- Export `OrderingStrategy` interface for pluggable algorithms
- Risk: Low, isolated to block building

### Binary RPC Layer
- Implement gRPC service in `api/grpc/`
- Core endpoints: `GetStorageBatch`, `SimulateBundles`, `SubmitBundle`, `GetPendingTransactions`
- Target: 10x latency improvement over JSON-RPC
- Risk: Low, runs alongside existing RPC

### Benchmarking Framework
- Establish baseline metrics vs vanilla Geth
- Automated latency tracking: tx propagation, block import, simulation throughput, API latency
- Continuous integration testing against mainnet replays

## Phase 2: Mempool Fast Path (4-6 weeks)

### Lock-Free Transaction Feed
- Implement ring buffer in `core/txpool/fastfeed/`
- Hook into txpool at `insertFeed.Send()` points
- Binary layout for zero-copy access
- Target: Sub-100μs tx propagation to consumers
- Risk: Medium, touches critical path

### Shared Memory Interface
- Expose transaction events via mmap'd regions
- Selective filtering by contract address or method selector
- Support multiple concurrent consumers
- Risk: Medium, IPC complexity

## Phase 3: Hot State Cache (6-8 weeks)

### DeFi Contract State Pinning
- Maintain in-memory cache of top 100 pool/vault states
- Pre-decode reserves, fees, and critical parameters
- Update atomically on block import
- Target: Sub-microsecond state access for hot contracts
- Risk: High, state consistency is critical

### State Delta Export
- After each block, export structured diffs of watched addresses
- Binary format for rapid consumption by trading strategies
- Include pre and post-execution state
- Risk: Medium

### Integration Points
- `core/blockchain.go`: Hook into `ProcessBlock()`
- `core/state_processor.go`: Intercept state changes during EVM execution
- Deploy in shadow mode initially to verify correctness

## Phase 4: Native API (4-6 weeks)

### Shared Library Interface
- Build Mandarin as `libmandarin.so` with C ABI
- Export simulation, state access, and bundle submission functions
- Enable in-process trading strategies
- Target: Sub-100μs API calls
- Risk: Medium, ABI stability and versioning concerns

### Safety Mechanisms
- Process isolation boundaries
- Memory protection
- API rate limiting and resource quotas

## Key Metrics

### Target Improvements vs Baseline Geth
- Transaction propagation (P99): 1.2ms → 45μs
- Block import (avg): 120ms → 68ms
- Simulation throughput: 200/sec → 15,000/sec
- API latency (P50): 8ms → 12μs

### Monitoring
- Real-time latency percentiles (P50, P99, P999)
- Memory overhead tracking
- State consistency validation
- Regression detection in CI

## Out of Scope

### Not Prioritized
- Full kernel-bypass networking (DPDK/AF_XDP): Complex, limited benefit
- Custom P2P protocol: Breaks compatibility
- Core EVM rewrites: High maintenance burden, modest gains

## Initial Validation

### Quick Wins (First 2 Weeks)
1. Bundle simulation endpoint using existing `eth_call` infrastructure
2. gRPC implementation for 5 critical APIs
3. Benchmark suite comparing to vanilla Geth
4. Publish latency improvements to validate approach

### Success Criteria
- Measurable 5-10x latency improvements in Phase 1
- Zero state consistency issues in hot cache
- Production stability matching Geth reliability standards

