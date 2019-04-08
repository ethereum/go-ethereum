# go-bip39
[![Build Status](https://travis-ci.org/tyler-smith/go-bip39.svg?branch=master)](https://travis-ci.org/tyler-smith/go-bip39)
[![license](https://img.shields.io/github/license/tyler-smith/go-bip39.svg?maxAge=2592000)](https://github.com/tyler-smith/go-bip39/blob/master/LICENSE)
[![Documentation](https://godoc.org/github.com/tyler-smith/go-bip39?status.svg)](http://godoc.org/github.com/tyler-smith/go-bip39)
[![Go Report Card](https://goreportcard.com/badge/github.com/tyler-smith/go-bip39)](https://goreportcard.com/report/github.com/tyler-smith/go-bip39)
[![GitHub issues](https://img.shields.io/github/issues/tyler-smith/go-bip39.svg)](https://github.com/tyler-smith/go-bip39/issues)


A golang implementation of the BIP0039 spec for mnemonic seeds

## Example

```go
package main

import (
  "github.com/tyler-smith/go-bip39"
  "github.com/tyler-smith/go-bip32"
  "fmt"
)

func main(){
  // Generate a mnemonic for memorization or user-friendly seeds
  entropy, _ := bip39.NewEntropy(256)
  mnemonic, _ := bip39.NewMnemonic(entropy)

  // Generate a Bip32 HD wallet for the mnemonic and a user supplied password
  seed := bip39.NewSeed(mnemonic, "Secret Passphrase")

  masterKey, _ := bip32.NewMasterKey(seed)
  publicKey := masterKey.PublicKey()

  // Display mnemonic and keys
  fmt.Println("Mnemonic: ", mnemonic)
  fmt.Println("Master private key: ", masterKey)
  fmt.Println("Master public key: ", publicKey)
}
```

## Credits

Wordlists are from the [bip39 spec](https://github.com/bitcoin/bips/tree/master/bip-0039).

Test vectors are from the standard Python BIP0039 implementation from the
Trezor team: [https://github.com/trezor/python-mnemonic](https://github.com/trezor/python-mnemonic)
