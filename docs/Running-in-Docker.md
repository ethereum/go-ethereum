# Running in Docker

We keep a Docker image with recent snapshot builds from the `develop` branch [on DockerHub](https://hub.docker.com/r/ethereum/client-go/). In addition to the container based on [Ubuntu](http://www.ubuntu.com) (158 MB), there is a smaller image using [Alpine Linux](https://alpinelinux.org) (35 MB). To use the alpine [tag](https://hub.docker.com/r/ethereum/client-go/tags), replace `ethereum/client-go` with `ethereum/client-go:alpine` in the examples below.

To pull the image, run this command:

```shell
docker pull ethereum/client-go
```

Start a node with:

```shell
docker run -it -p 30303:30303 ethereum/client-go
```

To start a node that runs the JSON-RPC interface on port **8545**, run:

```shell
docker run -it -p 8545:8545 -p 30303:30303 ethereum/client-go --rpc --rpcaddr "0.0.0.0"
```
**WARNING: This opens your container to external calls. "0.0.0.0" should _not_ be used when exposed to public networks**

To use the interactive JavaScript console, run:

```shell
docker run -it -p 30303:30303 ethereum/client-go console
```

## Using Data Volumes

To persist downloaded blockchain data between container starts, use Docker [data volumes](https://docs.docker.com/engine/tutorials/dockervolumes/#/mount-a-host-directory-as-a-data-volume). Replace `/path/on/host` with the location you want to store the data in.

    docker run -it -p 30303:30303 -v /path/on/host:/root/.ethereum ethereum/client-go

