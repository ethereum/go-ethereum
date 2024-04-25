## VM Runtime Benchmark
This benchmark is designed to measure the runtime performance of the VM and the execution of the contract bytecode

This is a standalone tool with the following features:
- The tool can be compiled for any platform and run without the need for a full node or the source code
- Benchmark parameters are controlled by the command line arguments
- Ability to run multiple iterations, or samples, of the benchmark
- Output is in a format that can be easily parsed by other tools
- Minimal overhead for benchmarking

### Sample Usage

To run it simply execute the following command:
```bash
go run vm-runtime-bench.go -bytecode 61FFFF600020 -samples 10
```

Or if you have the release version:
```bash
./vm-runtime-benchmark -bytecode 61FFFF600020 -samples 10
```

### Output format

The default output is in CSV format that can be easily parsed by other tools. For more human-readable output, use the `-csv=false` flag:
```bash
go run vm-runtime-bench.go -bytecode 61FFFF600020 -csv=false
```

### State Management

All the state is managed in memory for lower overhead. By default, the state is not preserved between samples. This assumes that the bytecode does not modify the state. 

For the bytecode that modifies the state (e.g. includes the `SSTORE` opcode), use the `-preserveState` flag to preserve the state and reset it between samples:
```bash
go run vm-runtime-bench.go -bytecode 604260005260206000F3 -preserveState
```

### Calldata

If your contract requires calldata, you can specify it using the `-calldata` flag:
```bash
go run vm-runtime-bench.go -bytecode 604260005260206000F3 -calldata 0102030405060708090A0B0C0D0E0F
```


