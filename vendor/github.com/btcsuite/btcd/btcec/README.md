btcec
=====

[![Build Status](https://travis-ci.org/btcsuite/btcd.png?branch=master)](https://travis-ci.org/btcsuite/btcec)
[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)
[![GoDoc](https://godoc.org/github.com/btcsuite/btcd/btcec?status.png)](http://godoc.org/github.com/btcsuite/btcd/btcec)

Package btcec implements elliptic curve cryptography needed for working with
Bitcoin (secp256k1 only for now). It is designed so that it may be used with the
standard crypto/ecdsa packages provided with go.  A comprehensive suite of test
is provided to ensure proper functionality.  Package btcec was originally based
on work from ThePiachu which is licensed under the same terms as Go, but it has
signficantly diverged since then.  The btcsuite developers original is licensed
under the liberal ISC license.

Although this package was primarily written for btcd, it has intentionally been
designed so it can be used as a standalone package for any projects needing to
use secp256k1 elliptic curve cryptography.

## Installation and Updating

```bash
$ go get -u github.com/btcsuite/btcd/btcec
```

## Examples

* [Sign Message](http://godoc.org/github.com/btcsuite/btcd/btcec#example-package--SignMessage)  
  Demonstrates signing a message with a secp256k1 private key that is first
  parsed form raw bytes and serializing the generated signature.

* [Verify Signature](http://godoc.org/github.com/btcsuite/btcd/btcec#example-package--VerifySignature)  
  Demonstrates verifying a secp256k1 signature against a public key that is
  first parsed from raw bytes.  The signature is also parsed from raw bytes.

* [Encryption](http://godoc.org/github.com/btcsuite/btcd/btcec#example-package--EncryptMessage)
  Demonstrates encrypting a message for a public key that is first parsed from
  raw bytes, then decrypting it using the corresponding private key.

* [Decryption](http://godoc.org/github.com/btcsuite/btcd/btcec#example-package--DecryptMessage)
  Demonstrates decrypting a message using a private key that is first parsed
  from raw bytes.

## GPG Verification Key

All official release tags are signed by Conformal so users can ensure the code
has not been tampered with and is coming from the btcsuite developers.  To
verify the signature perform the following:

- Download the public key from the Conformal website at
  https://opensource.conformal.com/GIT-GPG-KEY-conformal.txt

- Import the public key into your GPG keyring:
  ```bash
  gpg --import GIT-GPG-KEY-conformal.txt
  ```

- Verify the release tag with the following command where `TAG_NAME` is a
  placeholder for the specific tag:
  ```bash
  git tag -v TAG_NAME
  ```

## License

Package btcec is licensed under the [copyfree](http://copyfree.org) ISC License
except for btcec.go and btcec_test.go which is under the same license as Go.

