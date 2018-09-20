// +build arm amd64,!appengine,!gccgo

// Written in 2012 by Dmitry Chestnykh.
//
// To the extent possible under law, the author have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
// http://creativecommons.org/publicdomain/zero/1.0/

// This file contains a function definition for use with assembly implementations of Hash()

package siphash

//go:noescape

// Hash returns the 64-bit SipHash-2-4 of the given byte slice with two 64-bit
// parts of 128-bit key: k0 and k1.
func Hash(k0, k1 uint64, b []byte) uint64

//go:noescape

// Hash128 returns the 128-bit SipHash-2-4 of the given byte slice with two
// 64-bit parts of 128-bit key: k0 and k1.
func Hash128(k0, k1 uint64, b []byte) (uint64, uint64)

//go:noescape
func blocks(d *digest, p []uint8)

//go:noescape
func finalize(d *digest) uint64

//go:noescape
func once(d *digest)
