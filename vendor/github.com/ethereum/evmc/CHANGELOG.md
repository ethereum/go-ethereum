# Changelog

## [6.0.0] - 2018-10-24

- Added: [[#116](https://github.com/ethereum/evmc/pull/116)]
  [EVMC Host implementation example](https://github.com/ethereum/evmc/blob/master/examples/example_host.cpp).
- Added: [[#127](https://github.com/ethereum/evmc/pull/127)]
  Support for Constantinople SSTORE net gas metering.
- Added: [[#133](https://github.com/ethereum/evmc/pull/133)]
  Support for Constantinople CREATE2 salt in Go bindings.
- Added: [[#144](https://github.com/ethereum/evmc/pull/144)]
  A VM can now report its **capabilities** (i.e. EVM and/or ewasm).
- Added: [[#159](https://github.com/ethereum/evmc/pull/159)]
  [EVMC Host implementation guide](https://ethereum.github.io/evmc/hostguide.html).
- Added: [[#160](https://github.com/ethereum/evmc/pull/160)]
  [EVMC VM implementation guide](https://ethereum.github.io/evmc/vmguide.html).
- Changed: [[#119](https://github.com/ethereum/evmc/pull/119)]
  EVMC loader symbol searching has been generalized.
- Changed: [[#125](https://github.com/ethereum/evmc/pull/125)]
  The `evmc_context_fn_table` renamed to `evmc_host_interface`.
- Changed: [[#128](https://github.com/ethereum/evmc/pull/128)]
  The `evmc_message` fields reordered.
- Changed: [[#136](https://github.com/ethereum/evmc/pull/136)]
  The `evmc_set_option()` now returns more information about the failure cause.
- Changed: [[#138](https://github.com/ethereum/evmc/pull/138)], [[#140](https://github.com/ethereum/evmc/pull/140)]
  In C the `bool` type is used instead of `int` for true/false flags.
- Changed: [[#152](https://github.com/ethereum/evmc/pull/152)]
  Introduction of the `evmc_bytes32` type.
- Changed: [[#154](https://github.com/ethereum/evmc/pull/154)]
  Simplification of signatures of Host methods.

## [5.2.0] - 2018-08-28

- Feature: [[#81](https://github.com/ethereum/evmc/pull/81)]
  Use also "evmc_create" function name for loading EVMC DLLs.
- Fix: [[#92](https://github.com/ethereum/evmc/pull/92)]
  The evmc.h header compatibility with C++98 fixed.
- Fix: [[#93](https://github.com/ethereum/evmc/pull/93)], [[#103](https://github.com/ethereum/evmc/pull/103)]
  Compilation and build configuration fixes.
- Improved: [[#97](https://github.com/ethereum/evmc/pull/97)], [[#107](https://github.com/ethereum/evmc/pull/107)]
  Documentation improvements, including documentation for the VM Tester.

## [5.1.0] - 2018-08-23

- Feature: [[#41](https://github.com/ethereum/evmc/pull/41)]
  Go language bindings for EVMC.
- Feature: [[#56](https://github.com/ethereum/evmc/pull/56), [#62](https://github.com/ethereum/evmc/pull/62)]
  New error codes.
- Feature: [[#67](https://github.com/ethereum/evmc/pull/67), [#68](https://github.com/ethereum/evmc/pull/68), [#70](https://github.com/ethereum/evmc/pull/70)]
  More helper functions.
- Fix: [[#72](https://github.com/ethereum/evmc/pull/72)]
  Go bindings: Properly handle unknown error codes.
- Improved: [[#58](https://github.com/ethereum/evmc/pull/58)]
  Documentation has been extended.
- Improved: [[#59](https://github.com/ethereum/evmc/pull/59)]
  Optional Result Storage helper module has been separated.
- Improved: [[#75](https://github.com/ethereum/evmc/pull/75)]
  Cable upgraded to 0.2.11.
- Improved: [[#77](https://github.com/ethereum/evmc/pull/77)]
  The license changed from MIT to Apache 2.0.

## [5.0.0] - 2018-08-10

- Feature: [[#23](https://github.com/ethereum/evmc/pull/23), [#24](https://github.com/ethereum/evmc/pull/24)]
  List of status codes extended and reordered.
- Feature: [[#32](https://github.com/ethereum/evmc/pull/32)]
  VM Tracing API.
- Feature: [[#33](https://github.com/ethereum/evmc/pull/33), [#34](https://github.com/ethereum/evmc/pull/34)]
  The support library with metrics tables for EVM1 instructions.
- Feature: [[#35](https://github.com/ethereum/evmc/pull/35)]
  Ability to create EVMC CMake package.
- Feature: [[#40](https://github.com/ethereum/evmc/pull/40)]
  The loader support library for VM dynamic loading.
- Feature: [[#45](https://github.com/ethereum/evmc/pull/45)]
  Constantinople: Support for `CREATE2` instruction.
- Feature: [[#49](https://github.com/ethereum/evmc/pull/49)]
  Constantinople: Support for `EXTCODEHASH` instruction.
- Feature: [[#52](https://github.com/ethereum/evmc/pull/52)]
  Constantinople: Storage status is reported back from `evmc_set_storage()`.


[6.0.0]: https://github.com/ethereum/evmc/releases/tag/v6.0.0
[5.2.0]: https://github.com/ethereum/evmc/releases/tag/v5.2.0
[5.1.0]: https://github.com/ethereum/evmc/releases/tag/v5.1.0
[5.0.0]: https://github.com/ethereum/evmc/releases/tag/v5.0.0