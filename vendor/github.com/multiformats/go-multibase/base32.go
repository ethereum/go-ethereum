package multibase

import (
	b32 "github.com/multiformats/go-base32"
)

var base32StdLowerPad = b32.NewEncodingCI("abcdefghijklmnopqrstuvwxyz234567")
var base32StdLowerNoPad = base32StdLowerPad.WithPadding(b32.NoPadding)

var base32StdUpperPad = b32.NewEncodingCI("ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")
var base32StdUpperNoPad = base32StdUpperPad.WithPadding(b32.NoPadding)

var base32HexLowerPad = b32.NewEncodingCI("0123456789abcdefghijklmnopqrstuv")
var base32HexLowerNoPad = base32HexLowerPad.WithPadding(b32.NoPadding)

var base32HexUpperPad = b32.NewEncodingCI("0123456789ABCDEFGHIJKLMNOPQRSTUV")
var base32HexUpperNoPad = base32HexUpperPad.WithPadding(b32.NoPadding)
