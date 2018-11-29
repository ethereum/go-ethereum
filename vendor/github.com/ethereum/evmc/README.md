# EVMC

[![chat: on gitter][gitter badge]][Gitter]
[![readme style: standard][readme style standard badge]][standard readme]

> Ethereum Client-VM Connector API

The EVMC is the low-level ABI between Ethereum Virtual Machines (EVMs) and
Ethereum Clients. On the EVM side it supports classic EVM1 and [ewasm].
On the Client-side it defines the interface for EVM implementations
to access Ethereum environment and state.

## Usage

Please visit the [documentation].

## Related projects

### EVMs

- [aleth-interpreter]
- [evmjit]
- [Hera]

### Clients

- [aleth]
- [nim-evmc]
- [go-ethereum] (in progress)
- [pyevm] (in progress)
- [pyethereum] (abandoned)

## Contribute

[![chat: on gitter][gitter badge]][Gitter]

Talk with us on the [EVMC Gitter chat][Gitter].

## Maintainers

- Alex Beregszaszi [@axic]
- Paweł Bylica [@chfast]

See also the list of [EVMC Authors](AUTHORS.md).

## License

Licensed under the [MIT License](LICENSE).


[@axic]: https://github.com/axic
[@chfast]: https://github.com/chfast
[documentation]: https://ethereum.github.io/evmc
[ewasm]: https://github.com/ewasm/design
[evmjit]: https://github.com/ethereum/evmjit
[Hera]: https://github.com/ewasm/hera
[Gitter]: https://gitter.im/ethereum/evmc
[aleth-interpreter]: https://github.com/ethereum/aleth/tree/master/libaleth-interpreter
[aleth]: https://github.com/ethereum/aleth
[nim-evmc]: https://github.com/status-im/nim-evmc
[go-ethereum]: https://github.com/ethereum/go-ethereum/pull/17050
[pyevm]: https://github.com/ethereum/py-evm
[pyethereum]: https://github.com/ethereum/pyethereum/pull/406
[standard readme]: https://github.com/RichardLitt/standard-readme

[gitter badge]: https://img.shields.io/gitter/room/ethereum/evmc.svg?style=flat-square
[readme style standard badge]: https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square
