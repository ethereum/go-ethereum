// +build dummy

// Package c contains only a C file.
//
// This Go file is part of a workaround for `go mod vendor`.
// Please see the file dummy.go at the root of the module for more information.
package secp256k1

import (
	_ "github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1/include"
	_ "github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1/src"
	_ "github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1/src/modules/recovery"
)
