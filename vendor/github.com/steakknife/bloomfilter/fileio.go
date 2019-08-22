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
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
)

// ReadFrom r and overwrite f with new Bloom filter data
func (f *Filter) ReadFrom(r io.Reader) (n int64, err error) {
	f2, n, err := ReadFrom(r)
	if err != nil {
		return -1, err
	}
	f.lock.Lock()
	defer f.lock.Unlock()
	f.m = f2.m
	f.n = f2.n
	f.bits = f2.bits
	f.keys = f2.keys
	return n, nil
}

// ReadFrom Reader r into a lossless-compressed Bloom filter f
func ReadFrom(r io.Reader) (f *Filter, n int64, err error) {
	rawR, err := gzip.NewReader(r)
	if err != nil {
		return nil, -1, err
	}
	defer func() {
		err = rawR.Close()
	}()

	content, err := ioutil.ReadAll(rawR)
	if err != nil {
		return nil, -1, err
	}

	f = new(Filter)
	n = int64(len(content))
	err = f.UnmarshalBinary(content)
	if err != nil {
		return nil, -1, err
	}
	return f, n, nil
}

// ReadFile from filename into a lossless-compressed Bloom Filter f
// Suggested file extension: .bf.gz
func ReadFile(filename string) (f *Filter, n int64, err error) {
	r, err := os.Open(filename)
	if err != nil {
		return nil, -1, err
	}
	defer func() {
		err = r.Close()
	}()

	return ReadFrom(r)
}

// WriteTo a Writer w from lossless-compressed Bloom Filter f
func (f *Filter) WriteTo(w io.Writer) (n int64, err error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	rawW := gzip.NewWriter(w)
	defer func() {
		err = rawW.Close()
	}()

	content, err := f.MarshalBinary()
	if err != nil {
		return -1, err
	}

	intN, err := rawW.Write(content)
	n = int64(intN)
	return n, err
}

// WriteFile filename from a a lossless-compressed Bloom Filter f
// Suggested file extension: .bf.gz
func (f *Filter) WriteFile(filename string) (n int64, err error) {
	w, err := os.Create(filename)
	if err != nil {
		return -1, err
	}
	defer func() {
		err = w.Close()
	}()

	return f.WriteTo(w)
}
