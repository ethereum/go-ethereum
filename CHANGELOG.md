# Changelog

## [1.15.0](https://github.com/taikoxyz/taiko-geth/compare/v1.14.1...v1.15.0) (2025-04-01)


### Features

* **miner:** improve `pruneTransactions` ([#411](https://github.com/taikoxyz/taiko-geth/issues/411)) ([7a019ea](https://github.com/taikoxyz/taiko-geth/commit/7a019ea44bc98be082a1a4dfa0c6975b30939196))
* **miner:** reduce the number compression attempts when fetching transactions list ([#406](https://github.com/taikoxyz/taiko-geth/issues/406)) ([9e6edc5](https://github.com/taikoxyz/taiko-geth/commit/9e6edc51dbb37b3f9b280c95a031c0a2f68af53a))


### Chores

* **eth:** always use the latest block number for pending state in RPC calls ([#410](https://github.com/taikoxyz/taiko-geth/issues/410)) ([6822358](https://github.com/taikoxyz/taiko-geth/commit/682235849b5df653c4108f2a4099ee39b8cde6b6))

## [1.14.1](https://github.com/taikoxyz/taiko-geth/compare/v1.14.0...v1.14.1) (2025-03-21)


### Bug Fixes

* **repo:** fix workflow to use configs ([#402](https://github.com/taikoxyz/taiko-geth/issues/402)) ([177750c](https://github.com/taikoxyz/taiko-geth/commit/177750c73ee40cb32f10b4e4d2276f1a3b0cad3b))
* **taiko-client:** fix an issue in `encodeAndCompressTxList` ([#404](https://github.com/taikoxyz/taiko-geth/issues/404)) ([8d5d308](https://github.com/taikoxyz/taiko-geth/commit/8d5d308dfbc465d111e044b0c4f245e3b1ef5c3a))

## [1.14.0](https://github.com/taikoxyz/taiko-geth/compare/v1.13.0...v1.14.0) (2025-03-21)


### Features

* **taiko_genesis:** update `TaikoGenesisBlock` configs ([#400](https://github.com/taikoxyz/taiko-geth/issues/400)) ([139e562](https://github.com/taikoxyz/taiko-geth/commit/139e56205075ba5897c2a4ca707a52b096a3f200))

## [1.13.0](https://github.com/taikoxyz/taiko-geth/compare/v1.12.0...v1.13.0) (2025-03-15)


### Features

* **consensus:** introduce `AnchorV3GasLimit` ([#378](https://github.com/taikoxyz/taiko-geth/issues/378)) ([a0b97be](https://github.com/taikoxyz/taiko-geth/commit/a0b97be30cc01a93cebbd2d7188d28b0dcc5989a))
* **consensus:** introduce cache for the payloads ([#380](https://github.com/taikoxyz/taiko-geth/issues/380)) ([36430eb](https://github.com/taikoxyz/taiko-geth/commit/36430eb4f53a1455eb331f58c3ad2e88d8f40ecf))
* **consensus:** update `TaikoAnchor.anchorV3` selector ([#379](https://github.com/taikoxyz/taiko-geth/issues/379)) ([1e948cf](https://github.com/taikoxyz/taiko-geth/commit/1e948cff4c83e7a5cb0d8a4db27cbe59ce2a8884))
* **core:** align the upstream `types.Header` ([#393](https://github.com/taikoxyz/taiko-geth/issues/393)) ([573f8fc](https://github.com/taikoxyz/taiko-geth/commit/573f8fc144670d7221b387661f1f18dcd0935fe1))
* **eth:** changes based on protocol `Pacaya` fork ([#367](https://github.com/taikoxyz/taiko-geth/issues/367)) ([7bf5c0d](https://github.com/taikoxyz/taiko-geth/commit/7bf5c0d259f60f9d62d481c873053548c87b6fb5))
* **miner:** use `[]*ethapi.RPCTransaction` in RPC response body ([#391](https://github.com/taikoxyz/taiko-geth/issues/391)) ([afd09af](https://github.com/taikoxyz/taiko-geth/commit/afd09afd03871f9c66231a92bba706f4c491b877))
* **repo:** `go-ethereum` v1.15.5 upstream merge  ([#395](https://github.com/taikoxyz/taiko-geth/issues/395)) ([364acd0](https://github.com/taikoxyz/taiko-geth/commit/364acd00d1f2b45a07b7b1d20ec9b0f77be50b91))
* **repo:** add do-not-merge and rename files ([#390](https://github.com/taikoxyz/taiko-geth/issues/390)) ([3792f93](https://github.com/taikoxyz/taiko-geth/commit/3792f9356bfdc391fc8e112fccc7bf31d9edbb63))
* **taiko_genesis:** update devnet genesis JSONs for new `AnchorV3` method ([#382](https://github.com/taikoxyz/taiko-geth/issues/382)) ([2448fb9](https://github.com/taikoxyz/taiko-geth/commit/2448fb97a8b873c7bd7c0051cd83aaea339050e0))


### Bug Fixes

* **eth:** write `L1Origin` even if the payload is already in the cache ([#396](https://github.com/taikoxyz/taiko-geth/issues/396)) ([43a60b3](https://github.com/taikoxyz/taiko-geth/commit/43a60b36ea53a416ba01f271749d20f1251ce607))
