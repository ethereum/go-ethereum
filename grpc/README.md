This package provides a gRPC server as an entrypoint to the EVM.

Helpful commands (MacOS):
```bash
# install necessary dependencies
brew install leveldb

# build geth
make geth

# TODO - run beacon?

# run geth
./build/bin/geth --grpc --grpc.addr "[::1]" --grpc.port 50051
```
