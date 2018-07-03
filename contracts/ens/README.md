# Swarm ENS interface

## Usage

Full documentation for the Ethereum Name Service [can be found as EIP 137](https://github.com/ethereum/EIPs/issues/137).
This package offers a simple binding that streamlines the registration of arbitrary UTF8 domain names to swarm content hashes.

## Development

The SOL file in contract subdirectory implements the ENS root registry, a simple
first-in, first-served registrar for the root namespace, and a simple resolver contract;
they're used in tests, and can be used to deploy these contracts for your own purposes.

The solidity source code can be found at [github.com/arachnid/ens/](https://github.com/arachnid/ens/).

The go bindings for ENS contracts are generated using `abigen` via the go generator:

```shell
go generate ./contracts/ens
```
