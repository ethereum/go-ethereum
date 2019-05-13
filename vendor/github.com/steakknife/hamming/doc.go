//
// Package hamming distance calculations in Go
//
// https://github.com/steakknife/hamming
//
// Copyright © 2014, 2015, 2016, 2018 Barry Allard
//
// MIT license
//
//
// Usage
//
// For functions named CountBits.+s?.  The plural forms are for slices.
// The CountBits.+ forms are Population Count only, where the bare-type
// forms are Hamming distance (number of bits different) between two values.
//
// Optimized assembly .+PopCnt forms are available on amd64, and operate just
// like the regular forms (Must check and guard on HasPopCnt() first before
// trying to call .+PopCnt functions).
//
//    import 'github.com/steakknife/hamming'
//
//    // ...
//
//    // hamming distance between values
//    hamming.Byte(0xFF, 0x00) // 8
//    hamming.Byte(0x00, 0x00) // 0
//
//    // just count bits in a byte
//    hamming.CountBitsByte(0xA5), // 4
//
// Got rune? use int32
// Got uint8? use byte
//
package hamming
