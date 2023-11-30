# Bootnode

## Options

- ```dry-run```: validates parameters and prints bootnode configurations, but does not start bootnode (default: false)

- ```listen-addr```: listening address of bootnode (<ip>:<port>) (default: 0.0.0.0:30303)

- ```log-level```: log level (trace|debug|info|warn|error|crit), will be deprecated soon. Use verbosity instead (default: info)

- ```metrics```: Enable metrics collection and reporting (default: true)

- ```nat```: port mapping mechanism (any|none|upnp|pmp|extip:<IP>) (default: none)

- ```node-key```: file or hex node key

- ```prometheus-addr```: listening address of bootnode (<ip>:<port>) (default: 127.0.0.1:7071)

- ```save-key```: path to save the ecdsa private key

- ```v5```: Enable UDP v5 (default: false)

- ```verbosity```: Logging verbosity (5=trace|4=debug|3=info|2=warn|1=error|0=crit) (default: 3)