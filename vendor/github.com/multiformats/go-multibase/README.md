# go-multibase

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-multiformats-blue.svg?style=flat-square)](https://github.com/multiformats/multiformats)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](https://webchat.freenode.net/?channels=%23ipfs)
[![](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![Travis CI](https://img.shields.io/travis/multiformats/go-multibase.svg?style=flat-square&branch=master)](https://travis-ci.org/multiformats/go-multibase)
[![codecov.io](https://img.shields.io/codecov/c/github/multiformats/go-multibase.svg?style=flat-square&branch=master)](https://codecov.io/github/multiformats/go-multibase?branch=master)

> Implementation of [multibase](https://github.com/multiformats/multibase) -self identifying base encodings- in Go.


## Install

`go-multibase` is a standard Go module which can be installed with:

```sh
go get github.com/multiformats/go-multibase
```

Note that `go-multibase` is packaged with Gx, so it is recommended to use Gx to install and use it (see Usage section).

## Usage

This module is packaged with [Gx](https://github.com/whyrusleeping/gx). In order to use it in your own project it is recommended that you:

```sh
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/multiformats/go-multibase
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.

## Contribute

Contributions welcome. Please check out [the issues](https://github.com/multiformats/go-multibase/issues).

Check out our [contributing document](https://github.com/multiformats/multiformats/blob/master/contributing.md) for more information on how we work, and about contributing in general. Please be aware that all interactions related to multiformats are subject to the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

Small note: If editing the README, please conform to the [standard-readme](https://github.com/RichardLitt/standard-readme) specification.

## License

[MIT](LICENSE) © 2016 Protocol Labs Inc.
