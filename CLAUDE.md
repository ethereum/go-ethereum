# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is **bera-geth**, Berachain's fork of go-ethereum that implements the Berachain blockchain network. Key differences from upstream:

- **Custom base fee mechanism**: 1 gwei minimum with 6x faster adjustments via Prague1 fork
- **Networks**: Berachain mainnet (Chain ID 80094) and Bepolia testnet (Chain ID 80069)
- **Optimized for faster block times** while maintaining Ethereum compatibility

## Development Commands

### Building
```bash
make geth                    # Build main geth binary
make all                     # Build all executables
make evm                     # Build EVM utility
```

### Testing
```bash
make test                    # Run all tests
go test ./path/to/package    # Run specific package tests
go test -run TestName        # Run specific test
```

### Code Quality
```bash
make lint                    # Run linters
make fmt                     # Format code with gofmt
```

### Development Tools
```bash
make devtools                # Install development dependencies
```

### Cleaning
```bash
make clean                   # Clean build artifacts
```

## Codebase Architecture

### Core Components

**Core Blockchain Logic** (`/core/`)
- `blockchain.go` - Main blockchain state management
- `state_processor.go` - Transaction execution
- `types/` - Core data structures (blocks, transactions, receipts)
- `vm/` - Ethereum Virtual Machine implementation
- `txpool/` - Transaction pool management

**Consensus Layer** (`/consensus/`)
- `beacon/` - Proof-of-stake consensus (primary)
- `misc/eip1559/` - Base fee calculation with Berachain modifications
- `clique/` - Proof-of-authority for testing

**Network Protocol** (`/eth/`)
- `backend.go` - Main Ethereum service
- `downloader/` - Block/state synchronization
- `protocols/` - Network protocol handlers

**P2P Networking** (`/p2p/`)
- `discover/` - Node discovery protocols
- `rlpx/` - Encrypted communication

### Key Entry Points

**Main Binary**: `/cmd/geth/main.go`
- CLI application setup
- Network selection logic
- Node initialization

**Other Executables**: `/cmd/`
- `clef/` - Account management
- `abigen/` - ABI binding generator
- `evm/` - Standalone EVM executor

### Configuration Management

**Chain Configs**: `/params/config.go`
- Network parameters and fork schedules
- Berachain/Bepolia network definitions
- Prague1 fork configuration

**Network Discovery**: `/params/bootnodes.go`
- Bootstrap nodes for peer discovery

### Berachain-Specific Modifications

**Base Fee Changes** (`/consensus/misc/eip1559/eip1559.go:116-121`)
- Prague1 fork enforces 1 gwei minimum base fee
- BaseFeeChangeDenominator set to 48 (6x faster adjustment)

**Network Configurations** (`/params/config.go:77-99, 107-129`)
- BerachainChainConfig with Chain ID 80094
- BepoliaChainConfig with Chain ID 80069
- Custom Prague1 fork activation parameters

### Test Organization

Tests follow Go conventions with `*_test.go` files alongside source code:
- Unit tests in package directories
- Integration tests in `/tests/`
- Consensus tests for Ethereum specification compliance
- EVM operation tests in `/core/vm/testdata/`

### Build System

**Primary Build**: `/build/ci.go`
- Cross-platform compilation
- Docker image building
- Release packaging

**Simple Builds**: `/Makefile`
- Development-focused targets
- Requires Go 1.23+ and C compiler

## Development Workflow

1. **Architecture Understanding**: The codebase follows Ethereum's layered architecture (consensus → state → networking → application)
2. **Testing Strategy**: Always run tests after changes with `make test`
3. **Code Style**: Use `make fmt` and `make lint` before committing
4. **Network Testing**: Use `--bepolia` flag for testnet development
5. **Debugging**: Use `./build/bin/geth --help` for comprehensive CLI options

## Network-Specific Usage

### Berachain Mainnet
```bash
./build/bin/geth --berachain
```

### Bepolia Testnet
```bash
./build/bin/geth --bepolia
```

### Development Mode
```bash
./build/bin/geth --dev
```