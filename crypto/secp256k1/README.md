secp256k1-go
=======

golang secp256k1 library

Implements cryptographic operations for the secp256k1 ECDSA curve used by Bitcoin.

Installing
===

GMP library headers are required to build. On Debian-based systems, the package is called `libgmp-dev`.

```
sudo apt-get install libgmp-dev
```

Now compiles with cgo!

Test
===

To run tests do
```
go tests
```