4.0.0 / 2016-12-3
------------------
- `decode` now returns a `Buffer` again,  to avoid potential cryptographic errors. [Daniel Cousens / #21](https://github.com/cryptocoinjs/bs58/pull/21)

3.0.0 / 2015-08-18
------------------
- refactored module into generic [`base-x`](https://github.com/cryptocoinjs/base-x).

2.0.1 / 2014-12-23
------------------
- performance boost in `encode()` [#10](https://github.com/cryptocoinjs/bs58/pull/10)

2.0.0 / 2014-10-03
------------------
- `decode` now returns an `Array` instead of `Buffer` to keep things simple. [Daniel Cousens / #9](https://github.com/cryptocoinjs/bs58/pull/9)

1.2.1 / 2014-07-24
------------------
* speed optimizations [Daniel Cousens / #8](https://github.com/cryptocoinjs/bs58/pull/8)

1.2.0 / 2014-06-29
------------------
* removed `bigi` dep, implemented direct byte conversion [Jared Deckard / #6](https://github.com/cryptocoinjs/bs58/pull/6)

1.1.0 / 2014-06-26
------------------
* user `Buffer` internally for calculations, providing cleaner code and a performance increase. [Daniel Cousens](https://github.com/cryptocoinjs/bs58/commit/129c71de8bc1e36f113bce06da0616066f41c5ca)

1.0.0 / 2014-05-27
------------------
* removed `binstring` dep, `Buffer` now only input to `encode()` and output of `decode()`
* update `bigi` from `~0.3.0` to `^1.1.0`
* added travis-ci support
* added coveralls support
* modified tests and library to handle fixture style testing (thanks to bitcoinjs-lib devs and [Daniel Cousens](https://github.com/dcousens))


0.3.0 / 2014-02-24
------------------
* duck type input to `encode` and change output of `decode` to `Buffer`.


0.2.1 / 2014-02-24
------------------
* removed bower and component support. Closes #1
* convert from 4 spaces to 2


0.2.0 / 2013-12-07
------------------
* renamed from `cryptocoin-base58` to `bs58`


0.1.0 / 2013-11-20
------------------
* removed AMD support


0.0.1 / 2013-11-04
------------------
* initial release
