---
title: Running in Docker
---

We maintain a Docker image with recent snapshot builds from our `develop` branch on DockerHub. In addition to the container based on Ubuntu (158 MB), there is a smaller image using Alpine Linux (35 MB). To use the alpine [tag](https://hub.docker.com/r/ethereum/client-go/tags), replace `ethereum/client-go` with `ethereum/client-go:alpine` in the examples below.

To pull the image and start a node, run these commands:

```shell
docker pull ethereum/client-go
docker run -it -p 30303:30303 ethereum/client-go
```

To start a node that runs the JSON-RPC interface on port **8545**, run:

```shell
docker run -it -p 8545:8545 -p 30303:30303 ethereum/client-go --rpc --rpcaddr "0.0.0.0"
```

**WARNING: This opens your container to external calls. You should not use "0.0.0.0" when exposed to public networks.**

To use the interactive JavaScript console, run:

```shell
docker run -it -p 30303:30303 ethereum/client-go console
```

## Using Data Volumes

To persist downloaded blockchain data between container starts, use Docker [data volumes](https://docs.docker.com/engine/tutorials/dockervolumes/#/mount-a-host-directory-as-a-data-volume). Replace `/path/on/host` with the location you want to store the data.

```shell
docker run -it -p 30303:30303 -v /path/on/host:/root/.ethereum ethereum/client-go
```

## Different image versions

We maintain four different docker images for running the latest stable or development versions of Geth.

-   `ethereum/client-go:latest` is the latest development version of Geth
-   `ethereum/client-go:stable` is the latest stable version of Geth
-   `ethereum/client-go:{version}` is the stable version of Geth at a specific version number
-   `ethereum/client-go:release-{version}` is the latest stable version of Geth at a specific version family

The images have the following ports automatically exposed:

-   `8545` TCP, used by the HTTP based JSON RPC API
-   `8546` TCP, used by the WebSocket based JSON RPC API
-   `30303` TCP and UDP, used by the P2P protocol running the network
-   `30304` UDP, used by the P2P protocol's new peer discovery overlay
