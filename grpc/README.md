This package provides a gRPC server as an entrypoint to the EVM.

## Build and run from source:
```bash
# install necessary dependencies
brew install leveldb

# build geth
make geth

# generating protobuf files
buf generate buf.build/astria/astria --path "astria/execution"
```

See [private_network.md](../private_network.md) for running a local geth node.

### Running with remote Docker image:
```bash
docker run --rm \
  -p 8545:8545 -p 30303:30303 -p 50051:50051 \
  ghcr.io/astriaorg/go-ethereum --goerli \
  --grpc --grpc.addr "0.0.0.0" --grpc.port 50051
```

### Local Docker workflow:
```bash
# build local docker image
docker build \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg VERSION=0.1 \
  --build-arg BUILDNUM=1 \
  --tag ghcr.io/astriaorg/go-ethereum:local .

# run local docker image
docker run --rm \
  -p 8545:8545 -p 30303:30303 -p 50051:50051 \
  ghcr.io/astriaorg/go-ethereum:local --goerli \
  --grpc --grpc.addr "0.0.0.0" --grpc.port 50051

# build and push to remote from local (as opposed to gh action)
docker build \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg VERSION=0.1 \
  --build-arg BUILDNUM=1 \
  --tag ghcr.io/astriaorg/go-ethereum:latest .
echo $CR_PAT | docker login ghcr.io -u astriaorg --password-stdin
docker push ghcr.io/astriaorg/go-ethereum:latest
```
