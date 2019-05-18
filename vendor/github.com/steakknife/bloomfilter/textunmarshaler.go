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
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"fmt"
	"io"
)

const (
	keyFormat  = "%016x"
	bitsFormat = "%016x"
)

func nl() string {
	return fmt.Sprintln()
}

func unmarshalTextHeader(r io.Reader) (k, n, m uint64, err error) {
	format := "k" + nl() + "%d" + nl()
	format += "n" + nl() + "%d" + nl()
	format += "m" + nl() + "%d" + nl()
	format += "keys" + nl()

	_, err = fmt.Fscanf(r, format, k, n, m)
	return k, n, m, err
}

func unmarshalTextKeys(r io.Reader, keys []uint64) (err error) {
	for i := range keys {
		_, err = fmt.Fscanf(r, keyFormat, keys[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func unmarshalTextBits(r io.Reader, bits []uint64) (err error) {
	_, err = fmt.Fscanf(r, "bits")
	if err != nil {
		return err
	}

	for i := range bits {
		_, err = fmt.Fscanf(r, bitsFormat, bits[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func unmarshalAndCheckTextHash(r io.Reader, f *Filter) (err error) {
	_, err = fmt.Fscanf(r, "sha384")
	if err != nil {
		return err
	}

	actualHash := [sha512.Size384]byte{}

	for i := range actualHash {
		_, err = fmt.Fscanf(r, "%02x", actualHash[i])
		if err != nil {
			return err
		}
	}

	_, expectedHash, err := f.marshal()
	if err != nil {
		return err
	}

	if !hmac.Equal(expectedHash[:], actualHash[:]) {
		return errHash()
	}

	return nil
}

// UnmarshalText conforms to TextUnmarshaler
func UnmarshalText(text []byte) (f *Filter, err error) {
	r := bytes.NewBuffer(text)
	k, n, m, err := unmarshalTextHeader(r)
	if err != nil {
		return nil, err
	}

	keys, err := newKeysBlank(k)
	if err != nil {
		return nil, err
	}

	err = unmarshalTextKeys(r, keys)
	if err != nil {
		return nil, err
	}

	bits, err := newBits(m)
	if err != nil {
		return nil, err
	}

	err = unmarshalTextBits(r, bits)
	if err != nil {
		return nil, err
	}

	f, err = newWithKeysAndBits(m, keys, bits, n)
	if err != nil {
		return nil, err
	}

	err = unmarshalAndCheckTextHash(r, f)
	if err != nil {
		return nil, err
	}

	return f, nil
}

// UnmarshalText method overwrites f with data decoded from text
func (f *Filter) UnmarshalText(text []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	f2, err := UnmarshalText(text)
	if err != nil {
		return err
	}

	f.m = f2.m
	f.n = f2.n
	copy(f.bits, f2.bits)
	copy(f.keys, f2.keys)

	return nil
}
