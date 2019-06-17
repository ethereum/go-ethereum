// Package bloomfilter is face-meltingly fast, thread-safe,
// marshalable, unionable, probability- and
// optimal-size-calculating Bloom filter in go
//
// https://github.com/steakknife/bloomfilter
//
// Copyright © 2014, 2015, 2018 Barry Allard
//
// MIT license
//
package bloomfilter

import (
	"encoding"
	"encoding/gob"
	"io"
)

// compile-time conformance tests
var (
	_ encoding.BinaryMarshaler   = (*Filter)(nil)
	_ encoding.BinaryUnmarshaler = (*Filter)(nil)
	_ encoding.TextMarshaler     = (*Filter)(nil)
	_ encoding.TextUnmarshaler   = (*Filter)(nil)
	_ io.ReaderFrom              = (*Filter)(nil)
	_ io.WriterTo                = (*Filter)(nil)
	_ gob.GobDecoder             = (*Filter)(nil)
	_ gob.GobEncoder             = (*Filter)(nil)
)
