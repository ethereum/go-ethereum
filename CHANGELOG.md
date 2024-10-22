# Changelog

## [1.11.1](https://github.com/taikoxyz/taiko-geth/compare/v1.11.0...v1.11.1) (2024-10-22)


### Bug Fixes

* **taiko-geth:** fix a mempool fetch issue ([#333](https://github.com/taikoxyz/taiko-geth/issues/333)) ([1340ded](https://github.com/taikoxyz/taiko-geth/commit/1340ded3811193b46d18241e5810c5b47083821f))
* **taiko-geth:** revert a `tx.Shift()` change ([#335](https://github.com/taikoxyz/taiko-geth/issues/335)) ([46576d2](https://github.com/taikoxyz/taiko-geth/commit/46576d27209194db9e02ba38b9ab6b919679fcbd))
* **taiko-geth:** stop using `RevertToSnapshot` when fetching mempool ([#336](https://github.com/taikoxyz/taiko-geth/issues/336)) ([1216d8d](https://github.com/taikoxyz/taiko-geth/commit/1216d8d6051ba6f73ee42b395e973dccf1d90cf9))

## [1.11.0](https://github.com/taikoxyz/taiko-geth/compare/v1.10.0...v1.11.0) (2024-10-16)


### Features

* **core:** update `MainnetOntakeBlock` ([#330](https://github.com/taikoxyz/taiko-geth/issues/330)) ([cd72c5b](https://github.com/taikoxyz/taiko-geth/commit/cd72c5bf056cce5870b685226ae70e0d2620dc5e))

## [1.10.0](https://github.com/taikoxyz/taiko-geth/compare/v1.9.0...v1.10.0) (2024-10-03)


### Features

* **all:** changes based on Taiko protocol ([7e1b8b6](https://github.com/taikoxyz/taiko-geth/commit/7e1b8b65a3f8b931a5f141281c6ff82ad17028d0))
* **consensus:** improve `VerifyHeaders` for `taiko` consensus ([#238](https://github.com/taikoxyz/taiko-geth/issues/238)) ([4f36879](https://github.com/taikoxyz/taiko-geth/commit/4f368792dc27d1e5c5d92f44b2d4b0a3f2986e02))
* **consensus:** update `ValidateAnchorTx` ([#289](https://github.com/taikoxyz/taiko-geth/issues/289)) ([8ff161f](https://github.com/taikoxyz/taiko-geth/commit/8ff161fb39b76ef15585d26033131433c4530a3e))
* **core:** changes based on the latest `block.extradata` format ([#295](https://github.com/taikoxyz/taiko-geth/issues/295)) ([a875cc8](https://github.com/taikoxyz/taiko-geth/commit/a875cc83b907b026b88da887ce0a0d46c91d6980))
* **core:** decode basefee params from `block.extraData` ([#290](https://github.com/taikoxyz/taiko-geth/issues/290)) ([83564ba](https://github.com/taikoxyz/taiko-geth/commit/83564ba6fc9c20b1fa28ff94d65d5e19211a1aa2))
* **core:** introduce `BasefeeSharingPctg` in `BlockMetadata` ([#287](https://github.com/taikoxyz/taiko-geth/issues/287)) ([e6487f0](https://github.com/taikoxyz/taiko-geth/commit/e6487f00ed74139fb4169cf4ccd70488d933a01a))
* **core:** update `ontakeForkHeight` to Sep 24, 2024 ([#309](https://github.com/taikoxyz/taiko-geth/issues/309)) ([4e05e58](https://github.com/taikoxyz/taiko-geth/commit/4e05e5893b18482a90b1560019f93e90745cc0e0))
* **eip1559:** remove `CalcBaseFeeOntake()` method ([#293](https://github.com/taikoxyz/taiko-geth/issues/293)) ([124fde7](https://github.com/taikoxyz/taiko-geth/commit/124fde7e025d6ba88c5cf796d6a0a5fd19c21a19))
* **eth:** add default gpo price flag ([#258](https://github.com/taikoxyz/taiko-geth/issues/258)) ([0fb7ce1](https://github.com/taikoxyz/taiko-geth/commit/0fb7ce1999e6b8f4d39e78787525e236e007948f))
* **miner:** change invalid transaction log level to `DEBUG` ([#224](https://github.com/taikoxyz/taiko-geth/issues/224)) ([286ffe2](https://github.com/taikoxyz/taiko-geth/commit/286ffe2cbfd6e1b234c9ab3976b4daa60c8a24ce))
* **miner:** compress the txlist bytes after checking the transaction is executable ([#269](https://github.com/taikoxyz/taiko-geth/issues/269)) ([aa70708](https://github.com/taikoxyz/taiko-geth/commit/aa70708a69d9612bf2dffd218db7e703de1654c1))
* **miner:** count last oversized transaction ([#273](https://github.com/taikoxyz/taiko-geth/issues/273)) ([451a668](https://github.com/taikoxyz/taiko-geth/commit/451a668d79bb9e41bb34dfb5fdbd1e0301977a9b))
* **miner:** improve `prepareWork()` ([#292](https://github.com/taikoxyz/taiko-geth/issues/292)) ([06b2903](https://github.com/taikoxyz/taiko-geth/commit/06b29039cbf1f72d6163c0c4f658053acfcc5c47))
* **miner:** move `TAIKO_MIN_TIP` check to `commitL2Transactions` ([#272](https://github.com/taikoxyz/taiko-geth/issues/272)) ([f3a7fb6](https://github.com/taikoxyz/taiko-geth/commit/f3a7fb6311e9d59ba2fb55799b9eab614d488095))
* **repo:** `geth/v1.14.11` upstream merge ([#313](https://github.com/taikoxyz/taiko-geth/issues/313)) ([5c84a20](https://github.com/taikoxyz/taiko-geth/commit/5c84a20827473cbe60ed16827df21b4ad395c9c2))
* **taiko_api:** reduce the frequency of `zlib` compression when fetching txpool content ([#323](https://github.com/taikoxyz/taiko-geth/issues/323)) ([27b4d6e](https://github.com/taikoxyz/taiko-geth/commit/27b4d6ebf9959b096fb6c6ed7f5910fa93a59df3))
* **taiko_genesis:** update genesis JSONs ([#305](https://github.com/taikoxyz/taiko-geth/issues/305)) ([73df1f1](https://github.com/taikoxyz/taiko-geth/commit/73df1f1a116bdb530c5a8bd7fc20b64b491f2f3c))
* **taiko_genesis:** update genesis JSONs ([#315](https://github.com/taikoxyz/taiko-geth/issues/315)) ([ae8a194](https://github.com/taikoxyz/taiko-geth/commit/ae8a194c517e39fda7a4c330cd6e5a49a8df3621))
* **taiko_genesis:** update interanl devnet genesis JSON for ontake hardfork ([#288](https://github.com/taikoxyz/taiko-geth/issues/288)) ([a748b91](https://github.com/taikoxyz/taiko-geth/commit/a748b914abb1b5bc2a25fe40de6e38bb70e4235a))
* **taiko_genesis:** update interanl devnet genesis JSON for ontake hardfork ([#291](https://github.com/taikoxyz/taiko-geth/issues/291)) ([217c9ec](https://github.com/taikoxyz/taiko-geth/commit/217c9ec0f42f4785b44b8d2dbc4c046eb43e1d02))
* **taiko_genesis:** update internal devnet genesis JSON ([#285](https://github.com/taikoxyz/taiko-geth/issues/285)) ([b137b2a](https://github.com/taikoxyz/taiko-geth/commit/b137b2ac113dfe899bc538220cbdadf45b24f133))
* **taiko_genesis:** update internal devnet genesis JSON ([#296](https://github.com/taikoxyz/taiko-geth/issues/296)) ([882a6cd](https://github.com/taikoxyz/taiko-geth/commit/882a6cd3294cd1c74eac37fbc37c54e64f0dc363))
* **taiko_miner:** add `BuildTransactionsListsWithMinTip` method ([#283](https://github.com/taikoxyz/taiko-geth/issues/283)) ([c777d24](https://github.com/taikoxyz/taiko-geth/commit/c777d24af16915030536564b8cb44346866ab0b1))
* **taiko_miner:** remove an unnecessary check ([#239](https://github.com/taikoxyz/taiko-geth/issues/239)) ([974b338](https://github.com/taikoxyz/taiko-geth/commit/974b338e20c3a2ff48ecfd0174c595d6cb02e935))
* **taiko_worker:** skip blob transactions ([#280](https://github.com/taikoxyz/taiko-geth/issues/280)) ([30a615b](https://github.com/taikoxyz/taiko-geth/commit/30a615b4c3aafd0d395309035d58b86ff53c8eb0))
* **txpool:** introduce `TAIKO_MIN_TIP` env ([#264](https://github.com/taikoxyz/taiko-geth/issues/264)) ([a29520e](https://github.com/taikoxyz/taiko-geth/commit/a29520e066809dda21af463272b6ec1ef1cdfcae))
* **txpool:** update `ValidateTransaction` ([#237](https://github.com/taikoxyz/taiko-geth/issues/237)) ([6cc43e1](https://github.com/taikoxyz/taiko-geth/commit/6cc43e1d9c1ef34cba5fff2db3735ced3ad0a3a0))
* **txpool:** update `ValidateTransaction` ([#255](https://github.com/taikoxyz/taiko-geth/issues/255)) ([87f4206](https://github.com/taikoxyz/taiko-geth/commit/87f42062d9d02fd99be1f8c318baf573ef08135f))
* **txpool:** update max fee check in `ValidateTransaction()` ([#259](https://github.com/taikoxyz/taiko-geth/issues/259)) ([ef40d46](https://github.com/taikoxyz/taiko-geth/commit/ef40d46c0efbda50f0a2b84987291a4b8f9f2a2d))
* **worker:** add `chainId` check in `worker` ([#228](https://github.com/taikoxyz/taiko-geth/issues/228)) ([4ebcf66](https://github.com/taikoxyz/taiko-geth/commit/4ebcf6656c507c3164722148c16e76f7766fe52e))


### Bug Fixes

* broken url link ([#28342](https://github.com/taikoxyz/taiko-geth/issues/28342)) ([a5544d3](https://github.com/taikoxyz/taiko-geth/commit/a5544d35f6746c93d01e9c54c5bc5ef6567463b3))
* **core/txpool:** fix typos ([a081130](https://github.com/taikoxyz/taiko-geth/commit/a0811300815f1d4e79881113a102e91fdfeecdb8))
* **core:** fix a transaction `Message` assembling issue ([#308](https://github.com/taikoxyz/taiko-geth/issues/308)) ([04d76e8](https://github.com/taikoxyz/taiko-geth/commit/04d76e8f012e8a3d89d04f38dabac08e758f5a00))
* **eth:** mark anchor transaction in `traceBlockParallel` ([#243](https://github.com/taikoxyz/taiko-geth/issues/243)) ([8622b2c](https://github.com/taikoxyz/taiko-geth/commit/8622b2cce09330fc4957e22be5bd4685675411d9))
* fix some (ST1005)go-staticcheck ([2814ee0](https://github.com/taikoxyz/taiko-geth/commit/2814ee0547cb49dddf182bad802f19100608d5f8))
* **flag:** one typo ([52234eb](https://github.com/taikoxyz/taiko-geth/commit/52234eb17299dbccb108f74cf9ac94cc44bc6d6a))
* **taiko_api:** fix an `EstimatedGasUsed` calculation issue ([#322](https://github.com/taikoxyz/taiko-geth/issues/322)) ([96296fb](https://github.com/taikoxyz/taiko-geth/commit/96296fb42e08da4f0db1c836efb9c427740c92e4))
* **taiko_genesis:** update devnet Ontake fork hight ([#302](https://github.com/taikoxyz/taiko-geth/issues/302)) ([d065dd2](https://github.com/taikoxyz/taiko-geth/commit/d065dd2c3d005fb01590ecc82cda9c91678dfd13))
* **taiko_miner:** fix a typo ([#299](https://github.com/taikoxyz/taiko-geth/issues/299)) ([5faa71b](https://github.com/taikoxyz/taiko-geth/commit/5faa71b531cc889fb66868380d9063e8c78c7646))
* **taiko_worker:** fix a `maxBytesPerTxList` check issue ([#282](https://github.com/taikoxyz/taiko-geth/issues/282)) ([f930382](https://github.com/taikoxyz/taiko-geth/commit/f930382f4bf789bdc6c6fae5a410758a9f9bed7c))
* **taiko_worker:** fix a size limit check in `commitL2Transactions` ([#245](https://github.com/taikoxyz/taiko-geth/issues/245)) ([7a75d5e](https://github.com/taikoxyz/taiko-geth/commit/7a75d5e6b42ee57fed4df8713049c71e9b08657a))
* **txpool:** basefee requires mintip to not be nil. ([#297](https://github.com/taikoxyz/taiko-geth/issues/297)) ([6315fd4](https://github.com/taikoxyz/taiko-geth/commit/6315fd49697701beb1f18b8c8c0a6bdf97e862d5))
* **txpool:** fix the unit in a log ([#266](https://github.com/taikoxyz/taiko-geth/issues/266)) ([9594e0a](https://github.com/taikoxyz/taiko-geth/commit/9594e0a6a87d14bdaa594b3a31eec116ce24c948))
* typo ([d8a351b](https://github.com/taikoxyz/taiko-geth/commit/d8a351b58f147fc8e1527695ff7a3d19e6f3420b))
* update link to trezor ([1a79089](https://github.com/taikoxyz/taiko-geth/commit/1a79089193f2046c0cab60954bc05be2f52a2a90))
* update outdated link to trezor docs ([#28966](https://github.com/taikoxyz/taiko-geth/issues/28966)) ([1a79089](https://github.com/taikoxyz/taiko-geth/commit/1a79089193f2046c0cab60954bc05be2f52a2a90))
* **wokrer:** fix an issue in `sealBlockWith` ([#240](https://github.com/taikoxyz/taiko-geth/issues/240)) ([02c6ee9](https://github.com/taikoxyz/taiko-geth/commit/02c6ee9672c1b47ac534ec7224f45d9ab0652cdf))

## [1.8.0](https://github.com/taikoxyz/taiko-geth/compare/v1.7.0...v1.8.0) (2024-09-09)


### Features

* **core:** update `ontakeForkHeight` to Sep 24, 2024 ([#309](https://github.com/taikoxyz/taiko-geth/issues/309)) ([4e05e58](https://github.com/taikoxyz/taiko-geth/commit/4e05e5893b18482a90b1560019f93e90745cc0e0))

## [1.7.0](https://github.com/taikoxyz/taiko-geth/compare/v1.6.1...v1.7.0) (2024-08-29)


### Features

* **taiko_genesis:** update genesis JSONs ([#305](https://github.com/taikoxyz/taiko-geth/issues/305)) ([73df1f1](https://github.com/taikoxyz/taiko-geth/commit/73df1f1a116bdb530c5a8bd7fc20b64b491f2f3c))


### Bug Fixes

* **core:** fix a transaction `Message` assembling issue ([#308](https://github.com/taikoxyz/taiko-geth/issues/308)) ([04d76e8](https://github.com/taikoxyz/taiko-geth/commit/04d76e8f012e8a3d89d04f38dabac08e758f5a00))

## [1.6.1](https://github.com/taikoxyz/taiko-geth/compare/v1.6.0...v1.6.1) (2024-08-28)


### Bug Fixes

* **taiko_genesis:** update devnet Ontake fork hight ([#302](https://github.com/taikoxyz/taiko-geth/issues/302)) ([d065dd2](https://github.com/taikoxyz/taiko-geth/commit/d065dd2c3d005fb01590ecc82cda9c91678dfd13))

## [1.6.0](https://github.com/taikoxyz/taiko-geth/compare/v1.5.0...v1.6.0) (2024-08-26)


### Features

* **consensus:** update `ValidateAnchorTx` ([#289](https://github.com/taikoxyz/taiko-geth/issues/289)) ([8ff161f](https://github.com/taikoxyz/taiko-geth/commit/8ff161fb39b76ef15585d26033131433c4530a3e))
* **core:** changes based on the latest `block.extradata` format ([#295](https://github.com/taikoxyz/taiko-geth/issues/295)) ([a875cc8](https://github.com/taikoxyz/taiko-geth/commit/a875cc83b907b026b88da887ce0a0d46c91d6980))
* **core:** decode basefee params from `block.extraData` ([#290](https://github.com/taikoxyz/taiko-geth/issues/290)) ([83564ba](https://github.com/taikoxyz/taiko-geth/commit/83564ba6fc9c20b1fa28ff94d65d5e19211a1aa2))
* **core:** introduce `BasefeeSharingPctg` in `BlockMetadata` ([#287](https://github.com/taikoxyz/taiko-geth/issues/287)) ([e6487f0](https://github.com/taikoxyz/taiko-geth/commit/e6487f00ed74139fb4169cf4ccd70488d933a01a))
* **eip1559:** remove `CalcBaseFeeOntake()` method ([#293](https://github.com/taikoxyz/taiko-geth/issues/293)) ([124fde7](https://github.com/taikoxyz/taiko-geth/commit/124fde7e025d6ba88c5cf796d6a0a5fd19c21a19))
* **miner:** improve `prepareWork()` ([#292](https://github.com/taikoxyz/taiko-geth/issues/292)) ([06b2903](https://github.com/taikoxyz/taiko-geth/commit/06b29039cbf1f72d6163c0c4f658053acfcc5c47))
* **taiko_genesis:** update interanl devnet genesis JSON for ontake hardfork ([#288](https://github.com/taikoxyz/taiko-geth/issues/288)) ([a748b91](https://github.com/taikoxyz/taiko-geth/commit/a748b914abb1b5bc2a25fe40de6e38bb70e4235a))
* **taiko_genesis:** update interanl devnet genesis JSON for ontake hardfork ([#291](https://github.com/taikoxyz/taiko-geth/issues/291)) ([217c9ec](https://github.com/taikoxyz/taiko-geth/commit/217c9ec0f42f4785b44b8d2dbc4c046eb43e1d02))
* **taiko_genesis:** update internal devnet genesis JSON ([#285](https://github.com/taikoxyz/taiko-geth/issues/285)) ([b137b2a](https://github.com/taikoxyz/taiko-geth/commit/b137b2ac113dfe899bc538220cbdadf45b24f133))
* **taiko_genesis:** update internal devnet genesis JSON ([#296](https://github.com/taikoxyz/taiko-geth/issues/296)) ([882a6cd](https://github.com/taikoxyz/taiko-geth/commit/882a6cd3294cd1c74eac37fbc37c54e64f0dc363))


### Bug Fixes

* **taiko_miner:** fix a typo ([#299](https://github.com/taikoxyz/taiko-geth/issues/299)) ([5faa71b](https://github.com/taikoxyz/taiko-geth/commit/5faa71b531cc889fb66868380d9063e8c78c7646))
* **txpool:** basefee requires mintip to not be nil. ([#297](https://github.com/taikoxyz/taiko-geth/issues/297)) ([6315fd4](https://github.com/taikoxyz/taiko-geth/commit/6315fd49697701beb1f18b8c8c0a6bdf97e862d5))

## [1.5.0](https://github.com/taikoxyz/taiko-geth/compare/v1.4.0...v1.5.0) (2024-07-03)


### Features

* **taiko_miner:** add `BuildTransactionsListsWithMinTip` method ([#283](https://github.com/taikoxyz/taiko-geth/issues/283)) ([c777d24](https://github.com/taikoxyz/taiko-geth/commit/c777d24af16915030536564b8cb44346866ab0b1))

## [1.4.0](https://github.com/taikoxyz/taiko-geth/compare/v1.3.0...v1.4.0) (2024-07-02)


### Features

* **miner:** count last oversized transaction ([#273](https://github.com/taikoxyz/taiko-geth/issues/273)) ([451a668](https://github.com/taikoxyz/taiko-geth/commit/451a668d79bb9e41bb34dfb5fdbd1e0301977a9b))
* **taiko_worker:** skip blob transactions ([#280](https://github.com/taikoxyz/taiko-geth/issues/280)) ([30a615b](https://github.com/taikoxyz/taiko-geth/commit/30a615b4c3aafd0d395309035d58b86ff53c8eb0))


### Bug Fixes

* **taiko_worker:** fix a `maxBytesPerTxList` check issue ([#282](https://github.com/taikoxyz/taiko-geth/issues/282)) ([f930382](https://github.com/taikoxyz/taiko-geth/commit/f930382f4bf789bdc6c6fae5a410758a9f9bed7c))

## [1.3.0](https://github.com/taikoxyz/taiko-geth/compare/v1.2.0...v1.3.0) (2024-06-06)


### Features

* **miner:** compress the txlist bytes after checking the transaction is executable ([#269](https://github.com/taikoxyz/taiko-geth/issues/269)) ([aa70708](https://github.com/taikoxyz/taiko-geth/commit/aa70708a69d9612bf2dffd218db7e703de1654c1))
* **miner:** move `TAIKO_MIN_TIP` check to `commitL2Transactions` ([#272](https://github.com/taikoxyz/taiko-geth/issues/272)) ([f3a7fb6](https://github.com/taikoxyz/taiko-geth/commit/f3a7fb6311e9d59ba2fb55799b9eab614d488095))

## [1.2.0](https://github.com/taikoxyz/taiko-geth/compare/v1.1.0...v1.2.0) (2024-06-05)


### Features

* **txpool:** introduce `TAIKO_MIN_TIP` env ([#264](https://github.com/taikoxyz/taiko-geth/issues/264)) ([a29520e](https://github.com/taikoxyz/taiko-geth/commit/a29520e066809dda21af463272b6ec1ef1cdfcae))


### Bug Fixes

* **txpool:** fix the unit in a log ([#266](https://github.com/taikoxyz/taiko-geth/issues/266)) ([9594e0a](https://github.com/taikoxyz/taiko-geth/commit/9594e0a6a87d14bdaa594b3a31eec116ce24c948))

## [1.1.0](https://github.com/taikoxyz/taiko-geth/compare/v1.0.0...v1.1.0) (2024-05-27)


### Features

* **eth:** add default gpo price flag ([#258](https://github.com/taikoxyz/taiko-geth/issues/258)) ([0fb7ce1](https://github.com/taikoxyz/taiko-geth/commit/0fb7ce1999e6b8f4d39e78787525e236e007948f))
* **txpool:** update max fee check in `ValidateTransaction()` ([#259](https://github.com/taikoxyz/taiko-geth/issues/259)) ([ef40d46](https://github.com/taikoxyz/taiko-geth/commit/ef40d46c0efbda50f0a2b84987291a4b8f9f2a2d))

## 1.0.0 (2024-05-22)


### Features

* **all:** changes based on Taiko protocol ([7e1b8b6](https://github.com/taikoxyz/taiko-geth/commit/7e1b8b65a3f8b931a5f141281c6ff82ad17028d0))
* **consensus:** improve `VerifyHeaders` for `taiko` consensus ([#238](https://github.com/taikoxyz/taiko-geth/issues/238)) ([4f36879](https://github.com/taikoxyz/taiko-geth/commit/4f368792dc27d1e5c5d92f44b2d4b0a3f2986e02))
* **miner:** change invalid transaction log level to `DEBUG` ([#224](https://github.com/taikoxyz/taiko-geth/issues/224)) ([286ffe2](https://github.com/taikoxyz/taiko-geth/commit/286ffe2cbfd6e1b234c9ab3976b4daa60c8a24ce))
* **taiko_miner:** remove an unnecessary check ([#239](https://github.com/taikoxyz/taiko-geth/issues/239)) ([974b338](https://github.com/taikoxyz/taiko-geth/commit/974b338e20c3a2ff48ecfd0174c595d6cb02e935))
* **txpool:** update `ValidateTransaction` ([#237](https://github.com/taikoxyz/taiko-geth/issues/237)) ([6cc43e1](https://github.com/taikoxyz/taiko-geth/commit/6cc43e1d9c1ef34cba5fff2db3735ced3ad0a3a0))
* **txpool:** update `ValidateTransaction` ([#255](https://github.com/taikoxyz/taiko-geth/issues/255)) ([87f4206](https://github.com/taikoxyz/taiko-geth/commit/87f42062d9d02fd99be1f8c318baf573ef08135f))
* **worker:** add `chainId` check in `worker` ([#228](https://github.com/taikoxyz/taiko-geth/issues/228)) ([4ebcf66](https://github.com/taikoxyz/taiko-geth/commit/4ebcf6656c507c3164722148c16e76f7766fe52e))


### Bug Fixes

* broken url link ([#28342](https://github.com/taikoxyz/taiko-geth/issues/28342)) ([a5544d3](https://github.com/taikoxyz/taiko-geth/commit/a5544d35f6746c93d01e9c54c5bc5ef6567463b3))
* **core/txpool:** fix typos ([a081130](https://github.com/taikoxyz/taiko-geth/commit/a0811300815f1d4e79881113a102e91fdfeecdb8))
* **eth:** mark anchor transaction in `traceBlockParallel` ([#243](https://github.com/taikoxyz/taiko-geth/issues/243)) ([8622b2c](https://github.com/taikoxyz/taiko-geth/commit/8622b2cce09330fc4957e22be5bd4685675411d9))
* fix some (ST1005)go-staticcheck ([2814ee0](https://github.com/taikoxyz/taiko-geth/commit/2814ee0547cb49dddf182bad802f19100608d5f8))
* **flag:** one typo ([52234eb](https://github.com/taikoxyz/taiko-geth/commit/52234eb17299dbccb108f74cf9ac94cc44bc6d6a))
* **taiko_worker:** fix a size limit check in `commitL2Transactions` ([#245](https://github.com/taikoxyz/taiko-geth/issues/245)) ([7a75d5e](https://github.com/taikoxyz/taiko-geth/commit/7a75d5e6b42ee57fed4df8713049c71e9b08657a))
* typo ([d8a351b](https://github.com/taikoxyz/taiko-geth/commit/d8a351b58f147fc8e1527695ff7a3d19e6f3420b))
* update link to trezor ([1a79089](https://github.com/taikoxyz/taiko-geth/commit/1a79089193f2046c0cab60954bc05be2f52a2a90))
* update outdated link to trezor docs ([#28966](https://github.com/taikoxyz/taiko-geth/issues/28966)) ([1a79089](https://github.com/taikoxyz/taiko-geth/commit/1a79089193f2046c0cab60954bc05be2f52a2a90))
* **wokrer:** fix an issue in `sealBlockWith` ([#240](https://github.com/taikoxyz/taiko-geth/issues/240)) ([02c6ee9](https://github.com/taikoxyz/taiko-geth/commit/02c6ee9672c1b47ac534ec7224f45d9ab0652cdf))
