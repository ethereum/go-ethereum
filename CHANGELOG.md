# Changelog

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
