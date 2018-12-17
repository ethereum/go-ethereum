**DO NOT FORGET YOUR PASSWORD** and **BACKUP YOUR KEYSTORE**

# Backup & restore

## Data directory

Everything `geth` persists gets written inside its data directory (except for the PoW Ethash DAG, see note below).
The default data directory locations are platform specific:

* Mac: `~/Library/Ethereum`
* Linux: `~/.ethereum`
* Windows: `%APPDATA%\Ethereum`

Accounts are stored in the `keystore` subdirectory. The contents of this directories should be transportable between nodes, platforms, implementations (C++, Go, Python).

To configure the location of the data directory, the `--datadir` parameter can be specified. See [CLI Options](https://github.com/ethereum/go-ethereum/wiki/Command-Line-Options) for more details.

_**Note:** The [Ethash DAG](https://github.com/ethereum/go-ethereum/wiki/Mining#ethash-dag) is stored at `~/.ethash` (Mac/Linux) or `%APPDATA%\Ethash` (Windows) so that it can be reused by all clients. You can store this in a different location by using a symbolic link._

## Upgrades

Sometimes the internal database formats need updating (for example, when upgrade from before 0.9.20). This can be run with the following command (geth should not be otherwise running):

```
geth upgradedb
```

## Cleanup

Geth's blockchain and state databases can be removed with:

```
geth removedb
```

This is useful for deleting an old chain and sync'ing to a new one. It only affects data directories that can be re-created on synchronisation and does not touch the keystore.

## Blockchain import/export

Export the blockchain in binary format with:
```
geth export <filename>
```

Or if you want to back up portions of the chain over time, a first and last block can be specified. For example, to back up the first epoch:

```
geth export <filename> 0 29999
```

Note that when backing up a partial chain, the file will be appended rather than truncated.

Import binary-format blockchain exports with:
```
geth import <filename>
```

_See https://github.com/ethereum/wiki/wiki/Blockchain-import-export for more info_


And finally: **DO NOT FORGET YOUR PASSWORD** and **BACKUP YOUR KEYSTORE**