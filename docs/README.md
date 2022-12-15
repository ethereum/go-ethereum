
# Documentation

[The new command line interface (CLI)](./cli) in this version of Bor aims to give users more control over the codebase when interacting with and starting a node. We have made every effort to keep most of the flags similar to the old CLI, except for a few notable changes. One major change is the use of the --config flag, which previously represented fields without available flags. It now represents all flags available to the user, and will overwrite any other flags if provided. As a node operator, you still have the flexibility to modify flags as needed. Please note that this change does not affect the internal functionality of the node, and it remains compatible with Geth and the Ethereum Virtual Machine (EVM).

## Additional notes

- The new entrypoint to run the Bor client is ```server```.

  ```
  $ bor server <flags>
  ```

  See [here](./cli/server.md) for more flag details.

- The `bor dumpconfig` sub-command prints the default configurations, in the TOML format, on the terminal. One can `pipe (>)` this to a file (say `config.toml`) and use it to start bor. 

- A toml file now can be used instead of flags and can contain all configuration for the node to run. To simply run bor with a configuration file, the following command can be used. 

  ```
  $ bor server --config <path_to_config.toml>
  ```

- You can find an example config file [here](./cli/example_config.toml) to know more about what each flag is used for, what are the defaults and recommended values for different networks. 

- Toml files used earlier (with `--config` flag) to configure additional fields (like static and trusted nodes) are being deprecated and have been converted to flags. 
