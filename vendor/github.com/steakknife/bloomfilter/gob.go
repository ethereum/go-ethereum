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

import _ "encoding/gob" // make sure gob is available

// GobDecode conforms to interface gob.GobDecoder
func (f *Filter) GobDecode(data []byte) error {
	return f.UnmarshalBinary(data)
}

// GobEncode conforms to interface gob.GobEncoder
func (f *Filter) GobEncode() ([]byte, error) {
	return f.MarshalBinary()
}
