# Bootnode

## Options

- ```listen-addr```: listening address of bootnode (<ip>:<port>) (default: 0.0.0.0:30303)

- ```v5```: Enable UDP v5 (default: false)

- ```verbosity```: Logging verbosity (5=trace|4=debug|3=info|2=warn|1=error|0=crit) (default: 3)

- ```log-level```: log level (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead (default: info)

- ```nat```: port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: none)

- ```node-key```: file or hex node key

- ```save-key```: path to save the ecdsa private key

- ```dry-run```: validates parameters and prints bootnode configurations, but does not start bootnode (default: false)