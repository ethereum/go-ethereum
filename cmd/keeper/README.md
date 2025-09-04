# Keeper - geth as a zkvm guest

Keeper command is a specialized tool for validating stateless execution of Ethereum blocks. It's designed to run as a zkvm guest.

## Overview

The keeper reads an RLP-encoded payload containing:
- A block to execute
- A witness with the necessary state data

It then executes the block statelessly and validates that the computed state root and receipt root match the values in the block header.

## Architecture

The keeper uses build tags to compile platform-specific input methods:

```
cmd/keeper/
├── main.go                 # Main execution logic
├── getpayload_example.go     # Example implementation
└── README.md              # This file
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
go build -tags myplatform ./cmd/keeper
```

## Payload Structure

The payload is an RLP-encoded structure containing:

```go
type Payload struct {
    Block   *types.Block
    Witness *stateless.Witness
}
```

## Example Implementation

See `getinput_example.go` for a complete example that contains a payload. To build the example:

```bash
go build -tags example ./cmd/keeper
```
