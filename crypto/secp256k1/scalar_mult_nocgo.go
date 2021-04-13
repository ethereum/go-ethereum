// Copyright 2015 Jeffrey Wilcke, Felix Lange, Gustav Simonsson. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

// +build gofuzz !cgo

package secp256k1

import (
	"math/big"

	"github.com/btcsuite/btcd/btcec"
)

func ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int) {
	return btcec.S256().ScalarMult(Bx, By, scalar)
}
