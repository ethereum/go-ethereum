# Running a simple private geth network with Docker

This document shows a minimal example of running a single-node private
geth network using Docker. It is intended for local experiments and
should not be used as-is in production.

## Example `docker-compose.yml`

```yaml
version: "3.8"

services:
  geth:
    image: ethereum/client-go:stable
    container_name: geth-private
    command:
      [
        "--http",
        "--http.addr=0.0.0.0",
        "--http.api=eth,net,web3,personal",
        "--dev",
        "--dev.period=0",
        "--verbosity=3"
      ]
    ports:
      - "8545:8545"
      - "30303:30303"
    volumes:
      - ./data:/root/.ethereum
