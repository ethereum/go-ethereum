# Keeper - geth as a zkvm guest

Keeper command is a specialized tool for validating stateless execution of Ethereum blocks. It's designed to run as a zkvm guest.

## Overview

The keeper reads an RLP-encoded payload containing:
- A block to execute
- A witness with the necessary state data

It then executes the block statelessly and validates that the computed state root and receipt root match the values in the block header.

## Architecture

The keeper uses build tags to compile platform-specific input methods and chain configurations:

```
cmd/keeper/
├── main.go                    # Main execution logic
├── getpayload_example.go      # Example implementation with embedded data
├── getpayload_ziren.go        # Ziren zkVM implementation
├── chainconfig_mainnet.go     # Mainnet chain configuration
├── chainconfig_sepolia.go     # Sepolia chain configuration
├── chainconfig_hoodi.go       # Hoodi chain configuration
└── README.md                  # This file
```

## Creating a Custom Platform Implementation

To add support for a new platform (e.g., "myplatform"), create a new file with the appropriate build tag:

### 1. Create `getinput_myplatform.go`

```go
//go:build myplatform

package main

import (
    "github.com/ethereum/go-ethereum/params"
    // ... other imports as needed
)

// getChainConfig returns the chain configuration for this platform
func getChainConfig() *params.ChainConfig {
    // Return the appropriate chain config for your platform
    // Examples: params.MainnetChainConfig, params.SepoliaChainConfig, 
    // or a custom configuration
    return params.MainnetChainConfig
}

// getInput returns the RLP-encoded payload
func getInput() []byte {
    // Your platform-specific code to retrieve the RLP-encoded payload
    // This might read from:
    // - Memory-mapped I/O
    // - Hardware registers  
    // - Serial port
    // - Network interface
    // - File system
    
    // The payload must be RLP-encoded and contain:
    // - Block with transactions
    // - Witness with parent headers and state data
    
    return encodedPayload
}
```

### 2. Build for Your Platform

```bash
# Build with specific platform and chain configuration
go build -tags "myplatform mainnet" ./cmd/keeper

# Available chain configurations:
# - mainnet: Ethereum mainnet
# - sepolia: Sepolia testnet
# - hoodi: Hoodi testnet
# If no chain tag is specified, defaults to mainnet
```

## Payload Structure

The payload is an RLP-encoded structure containing:

```go
type Payload struct {
    Block   *types.Block
    Witness *stateless.Witness
}
```

## Build Examples

### Example Implementation
See `getpayload_example.go` for a complete example with embedded Hoodi block data:

```bash
# Build example with different chain configurations
go build -tags "example hoodi" ./cmd/keeper    # Example with Hoodi config (default for example)
go build -tags "example mainnet" ./cmd/keeper  # Example with mainnet config
go build -tags "example sepolia" ./cmd/keeper  # Example with Sepolia config
```

### Ziren zkVM Implementation
Build for the Ziren zkVM platform:

```bash
# For MIPS architecture (typical for zkVM)
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -tags "ziren mainnet" ./cmd/keeper
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -tags "ziren sepolia" ./cmd/keeper
GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -tags "ziren hoodi" ./cmd/keeper
```
