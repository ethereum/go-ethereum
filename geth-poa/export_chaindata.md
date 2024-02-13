# Blockchain Data Export Script README

## Overview

This script is designed to export blockchain data from a specific range of blocks. It stops a running Docker container that runs a blockchain node, exports the specified block range data into a file, and then restarts the container. This process is crucial for backing up blockchain data or analyzing the blockchain at a specific point in time.

## Prerequisites

- Docker installed on your machine.
- A running Docker container hosting a blockchain node.
- Geth (Go Ethereum) or a compatible Ethereum client installed for blockchain operations.
- Sufficient disk space for the export data.

## Usage

The script requires five arguments to run:

1. `containerId`: The ID or name of the Docker container running the blockchain node.
2. `chaindata`: The path to the blockchain data directory (`datadir`).
3. `exportdata`: The path and filename where the exported data will be saved.
4. `startBlock`: The starting block number from which to begin the export.
5. `endBlock`: The ending block number where the export will stop.

### Command Syntax

```sh
./export-chaindata.sh <containerId> <chaindata> <exportdata> <startBlock> <endBlock>
```

### Example

```sh
./export-chaindata.sh ethereum-node /path/to/chaindata /path/to/exportedData 0 2000
```

This command will stop the ethereum-node (better to stop member node) container, export blocks 0 to 2000 from the specified chain data directory to the specified file, and then restart the ethereum-node container.
