# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.TBD.0]

### Added

- Add support for the Berachain mainnet (`--berachain`) and Bepolia networks
  (`--bepolia`)
- Add support for configuring the Prague1 fork on Berachain networks.

### Changed

- BRIP-2: Allowed configuration of the `MinimumBaseFeeWei`. For Berachain, this will be set as 1 gwei. (https://github.com/berachain/bera-geth/pull/2)
- BRIP-2: Allowed configuration of the `BaseFeeChangeDenominator`. For Berachain, this will be set as 48 (a 6x increase corresponding to 6x faster block times). (https://github.com/berachain/bera-geth/pull/2)
- [WIP] Updated release GH workflow to publish tarballs and push built images to GHCR.
- Removed support for appveyor, gitea, travis, circleci.
