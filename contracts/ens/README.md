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

## Fallback contract support

In order to better support content resolution on different service providers (such as Swarm and IPFS), [EIP-1577](https://eips.ethereum.org/EIPS/eip-1577)
was introduced and with it changes that allow applications to know _where_ content hashes are stored (i.e. if the
requested hash resides on Swarm or IPFS).

The code under `contracts/ens/contract` reflects the new Public Resolver changes and the code under `fallback_contract` allows
us to support the old contract resolution in cases where the ENS name owner did not update her Resolver contract, until the migration
period ends (date arbitrarily set to June 1st, 2019).
