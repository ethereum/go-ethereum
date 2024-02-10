# Signer Node Setup Script

## Overview

This script facilitates the setup and registration of a signer node within a blockchain network using Docker and Geth. It exports several environment variables essential for configuring the signer node, initializes the Docker containers required for the node's operation, and proposes the node as a signer to the network.

## Prerequisites

- Docker and Docker Compose installed on your machine.
- Geth (Go Ethereum) client installed within the Docker containers.
- A running Ethereum network that supports the Clique consensus mechanism (Proof of Authority).

## Configuration

The script uses several environment variables for configuration, which can be set directly in the script or passed as environment variables before execution. Defaults are provided for each variable:

- `SIGNER_NODE_ADDRESS`: The Ethereum address of the signer node. Default is `0x57949c3552159532c324c6fa8b102696cf4504bc`.
- `SIGNER_NODE_PRIVATE_KEY`: The private key associated with the signer node's address. Default is `0x0300633b02bab7305e17a2eabc6477f5caa3bc705994d2e19f55e8427c38536e`.
- `SIGNER_NODE_VOLUME`: The Docker volume name for storing node data. Default is `geth-data-signer`.
- `SIGNER_NODE_PORT`: The port on which the signer node will listen. Default is `60605`.
- `SIGNER_NODE_IP`: The IP address of the signer node. Default is `172.29.0.102`.

## Usage

1. **Prepare Docker Compose File**: Ensure you have a `docker-compose-add-signer.yml` file configured for your Docker environment. This file should define the services required for your signer node, including the Ethereum client (Geth).

2. **Set Environment Variables (Optional)**: If you wish to use custom values for any of the configuration variables, export them before running the script:

```sh
   export SIGNER_NODE_ADDRESS="0xYourSignerAddress"
   export SIGNER_NODE_PRIVATE_KEY="0xYourPrivateKey"
   export SIGNER_NODE_VOLUME="YourVolumeName"
   export SIGNER_NODE_PORT=YourPortNumber
   export SIGNER_NODE_IP="YourNodeIP"
```

3. **Execute the Script**: Run the script by passing the names of the containers where you want to propose the signer node as an argument. Ensure you have the necessary permissions to execute the script:

```sh
   export ./add_signer.sh <container_id_1> <container_id_2> ...
```

4. **Verify Signer Node**: After execution, verify that the signer node has been successfully proposed and added to your Ethereum network as a signer. This can usually be done by checking the logs of your Ethereum client or using Geth's console.
