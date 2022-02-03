
# Documentation

- [Command-line-interface](./cli)

- [Configuration file](./config.md)

## Deprecation notes

- The new entrypoint to run the Bor client is ```server```.

```
$ bor server
```

- Toml files to configure nodes are being deprecated. Currently, we only allow for static and trusted nodes to be configured using toml files.

```
$ bor server --config ./legacy.toml
```

- ```Admin```, ```Personal``` and account related endpoints in ```Eth``` are being removed from the JsonRPC interface. Some of this functionality will be moved to the new GRPC server for operational tasks.
