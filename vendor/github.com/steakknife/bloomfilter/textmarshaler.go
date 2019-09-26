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

import "fmt"

// MarshalText conforms to encoding.TextMarshaler
func (f *Filter) MarshalText() (text []byte, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	s := fmt.Sprintln("k")
	s += fmt.Sprintln(f.K())
	s += fmt.Sprintln("n")
	s += fmt.Sprintln(f.n)
	s += fmt.Sprintln("m")
	s += fmt.Sprintln(f.m)

	s += fmt.Sprintln("keys")
	for key := range f.keys {
		s += fmt.Sprintf(keyFormat, key) + nl()
	}

	s += fmt.Sprintln("bits")
	for w := range f.bits {
		s += fmt.Sprintf(bitsFormat, w) + nl()
	}

	_, hash, err := f.marshal()
	if err != nil {
		return nil, err
	}
	s += fmt.Sprintln("sha384")
	for b := range hash {
		s += fmt.Sprintf("%02x", b)
	}
	s += nl()

	text = []byte(s)
	return text, nil
}
