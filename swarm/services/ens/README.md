# Swarm ENS interface

## Usage

Full documentation for the Ethereum Name Service  [can be found as EIP 137](https://github.com/ethereum/EIPs/issues/137)
Swarm offers a simple phase-one interface that streamlines the registration of swarm content  hashes to arbitrary utf8 domain names.
The interface is offered through the ENS RPC API module under the `ens` namespace. The API can be used via the console, rpc or web3.js.

### `ens.register(name)`

Sends a transaction that registers a name as a top-level domain, giving ownership to the current account, and automatically deploying a simple resolver contract for it.

This line registers the top-level domain `swarm`:

```
ens.register("swarm")
// {}
```

### `ens.setContentHash(name, hash)`

Sends a transaction that sets a content hash to a name using a (sub-domain) resolver:

```js
ens.setContentHash("swarm","0x7420f14a28e276dd39da6a967dec332a932256717451155fbd3870b202b561c4")
// {}
```

To function, the domain must have a resolver deployed that supports the `setContent` function, and the sending account must have permission to call that function. `ens.register()` deploys a resolver for you that meets this requirement.

The content hash argument is the bzz hash as returned by a swarm upload, e.g., using `bzz.upload`. The name here will allow the name setting, if the top-level domain is 'swarm', it dispatches to the resolver just registered.

```js
bzz.upload("path/to/my/directory", "index.html")
```

Here the second argument to `bzz.upload` is the relative path to the asset mapped on to the root hash of entire collection, effectively the landing page served on the bare hash (and therefore the root of the domain registered with that hash) as url.

`setContentHash` can also be used on subdomains:

```js
ens.setContentHash("album.swarm","7a59235e5f9c23bf74deb4838e24f75a77f786163f404c8004d79b5674625db0")
// {}
```



### `ens.resolve(name)`

To query the ENS, this read-only free call is provided.

```js
ens.resolve("swarm")
// "7420f14a28e276dd39da6a967dec332a932256717451155fbd3870b202b561c4"
```

This same backend method is used within swarm to resolve hostnames in urls, hence you can use the set names of registered domains in a bzz-scheme url `bzz://swarm` or normal http using a swarm http proxy: `http://localhost:32200/bzz:/swarm`.

## Development

The SOL file in contract subdirectory implements the ENS root registry, a simple first-in-first-served registrar for the root namespace, and a simple resolver contract; they're used in tests, and can be used to deploy these contracts for your own purposes.

The solidity source code can be found at [github.com/arachnid/ens/](https://github.com/arachnid/ens/).

The go bindings for ENS contracts are generated using `abigen` via the go generator:

```shell
godep go generate ./swarm/services/ens
```

see the preprocessor directives in leading comments of ens.go and ens_test.go
