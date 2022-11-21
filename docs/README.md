
# Documentation

- [Command-line-interface](./cli)

- [Configuration file](./config.md)

## Additional notes

- The new entrypoint to run the Bor client is ```server```.

  ```
  $ bor server <flags>
  ```

- Toml files used earlier just to configure static/trusted nodes are being deprecated. Instead, a toml file now can be used instead of flags and can contain all configuration for the node to run. The link to a sample config file is given above. To simply run bor with a configuration file, the following command can be used. 

  ```
  $ bor server --config <path_to_config.toml>
  ```
