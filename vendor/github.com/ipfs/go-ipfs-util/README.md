# go-ipfs-util

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](http://ipn.io)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](http://ipfs.io/)
[![](https://img.shields.io/badge/freenode-%23ipfs-blue.svg?style=flat-square)](http://webchat.freenode.net/?channels=%23ipfs)
[![standard-readme compliant](https://img.shields.io/badge/standard--readme-OK-green.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)
[![](https://img.shields.io/badge/discussion_repo-go_to_issues-brightgreen.svg?style=flat-square)](https://github.com/ipfs/NAME/issues)

> Common utilities used by go-ipfs and other related go packages

## Install

This is a Go module which can be installed with `go get github.com/ipfs/go-ipfs-util`. `go-ipfs-util` is however packaged with Gx, so it is recommended to use Gx to install it (see Usage section).

## Usage

This module is packaged with [Gx](https://github.com/whyrusleeping/gx).
In order to use it in your own project do:

```
go get -u github.com/whyrusleeping/gx
go get -u github.com/whyrusleeping/gx-go
cd <your-project-repository>
gx init
gx import github.com/ipfs/go-ipfs-util
gx install --global
gx-go --rewrite
```

Please check [Gx](https://github.com/whyrusleeping/gx) and [Gx-go](https://github.com/whyrusleeping/gx-go) documentation for more information.


## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/go-ipfs-util/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/contributing.md)

## License

MIT
