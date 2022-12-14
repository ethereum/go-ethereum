# Bootnode

## Options

- ```listen-addr```: listening address of bootnode (<ip>:<port>) (default: 0.0.0.0:30303)

- ```v5```: Enable UDP v5 (default: false)

- ```log-level```: Log level (trace|debug|info|warn|error|crit) (default: info)

- ```nat```: port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: none)

- ```node-key```: file or hex node key

- ```save-key```: path to save the ecdsa private key

- ```dry-run```: validates parameters and prints bootnode configurations, but does not start bootnode (default: false)