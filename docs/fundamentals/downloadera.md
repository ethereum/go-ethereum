---
title: Download Era
description: Instructions for downloading Era files
---

Era downloads are used to retrieve historical block bodies and receipts data that have been expired or pruned from a Geth node. Era files allow operators to efficiently reconstruct history without a full sync. Geth nodes with history expiry enabled can prune historical block bodies and receipts to significantly reduce storage requirements. However, in some cases operators may want to selectively restore some of this history for research, debugging, or compliance purposes. Era files provide an efficient way to retrieve historical data directly from trusted servers without needing to re-sync the entire chain. The geth `download-era` command enables targeted retrieval of this data.

## Requirements {#requirements}
Before downloading Era files:
 - Geth must be stopped during the era file download process.
 - Ensure sufficient disk space is available for storing the downloaded files.

Era files are indexed by block or epoch range. When downloading:
1. Geth queries the era server for the file corresponding to the requested range.
2. Downloaded files are automatically verified against known checksums.
3. Verified files are placed into the ancient store directory, ready for Geth to use.

## Download Era Command {#command}
```sh
geth download-era --server <url> [--block <range> | --epoch <range> | --all] --datadir <path>
```

| Flag        | Description                                              |
| ----------- | -------------------------------------------------------- |
| `--server`  | (Required) URL of the era server                         |
| `--block`   | Block number or range to download (e.g. `100000-200000`) |
| `--epoch`   | Epoch number or range to download (e.g. `100-200`)       |
| `--all`     | Download all available era files                         |
| `--datadir` | Geth datadir where era files will be stored              |


Range formats:
 - Single value: `500` → downloads only block or epoch `500`
 - Range: `100-200` → downloads inclusive range from `100`to `200`

Servers can be found at the following link: [Ethereum History Endpoints](https://eth-clients.github.io/history-endpoints/). This link will have the most updates list of clients serving era files. Currently, geth only supports era files of the `Era1` specification, so make sure download accordingly. 

## Example {#example}
Download epochs `100-300`:
```sh
geth download-era --server https://mainnet.era1.nimbus.team --epoch 100-300 --datadir /mnt/geth-data
```

Download all:
```sh
geth download-era --server https://mainnet.era1.nimbus.team --all --datadir /mnt/geth-data
```




