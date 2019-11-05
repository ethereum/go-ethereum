---
title: Backup & Restore
sort_key: C
---

Most important info first: **REMEMBER YOUR PASSWORD** and **BACKUP YOUR KEYSTORE**.

## Data Directory

Everything `geth` persists gets written inside its data directory. The default data
directory locations are platform specific:

* Mac: `~/Library/Ethereum`
* Linux: `~/.ethereum`
* Windows: `%APPDATA%\Ethereum`

Accounts are stored in the `keystore` subdirectory. The contents of this directories
should be transportable between nodes, platforms, implementations (C++, Go, Python).

To configure the location of the data directory, the `--datadir` parameter can be
specified. See [CLI Options](../interface/command-line-options) for more details.

Note the [ethash dag](../interface/mining) is stored at `~/.ethash` (Mac/Linux) or
`%APPDATA%\Ethash` (Windows) so that it can be reused by all clients. You can store this
in a different location by using a symbolic link.

## Cleanup

Geth's blockchain and state databases can be removed with:

```
geth removedb
```

This is useful for deleting an old chain and sync'ing to a new one. It only affects data
directories that can be re-created on synchronisation and does not touch the keystore.

## Blockchain Import/Export

Export the blockchain in binary format with:

```
geth export <filename>
```

Or if you want to back up portions of the chain over time, a first and last block can be
specified. For example, to back up the first epoch:

```
geth export <filename> 0 29999
```

Note that when backing up a partial chain, the file will be appended rather than
truncated.

Import binary-format blockchain exports with:

```
geth import <filename>
```

_See https://github.com/ethereum/wiki/wiki/Blockchain-import-export for more info_


And finally: **REMEMBER YOUR PASSWORD** and **BACKUP YOUR KEYSTORE**
