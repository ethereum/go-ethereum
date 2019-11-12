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
	"encoding/binary"
	"io"
)

func unmarshalBinaryHeader(r io.Reader) (k, n, m uint64, err error) {
	err = binary.Read(r, binary.LittleEndian, &k)
	if err != nil {
		return k, n, m, err
	}

	if k < KMin {
		return k, n, m, errK()
	}

	err = binary.Read(r, binary.LittleEndian, &n)
	if err != nil {
		return k, n, m, err
	}

	err = binary.Read(r, binary.LittleEndian, &m)
	if err != nil {
		return k, n, m, err
	}

	if m < MMin {
		return k, n, m, errM()
	}

	debug("read bf k=%d n=%d m=%d\n", k, n, m)

	return k, n, m, err
}

func unmarshalBinaryBits(r io.Reader, m uint64) (bits []uint64, err error) {
	bits, err = newBits(m)
	if err != nil {
		return bits, err
	}
	err = binary.Read(r, binary.LittleEndian, bits)
	return bits, err

}

func unmarshalBinaryKeys(r io.Reader, k uint64) (keys []uint64, err error) {
	keys = make([]uint64, k)
	err = binary.Read(r, binary.LittleEndian, keys)
	return keys, err
}

func checkBinaryHash(r io.Reader, data []byte) (err error) {
	expectedHash := make([]byte, sha512.Size384)
	err = binary.Read(r, binary.LittleEndian, expectedHash)
	if err != nil {
		return err
	}

	actualHash := sha512.Sum384(data[:len(data)-sha512.Size384])

	if !hmac.Equal(expectedHash, actualHash[:]) {
		debug("bloomfilter.UnmarshalBinary() sha384 hash failed:"+
			" actual %v  expected %v", actualHash, expectedHash)
		return errHash()
	}

	debug("bloomfilter.UnmarshalBinary() successfully read"+
		" %d byte(s), sha384 %v", len(data), actualHash)
	return nil
}

// UnmarshalBinary converts []bytes into a Filter
// conforms to encoding.BinaryUnmarshaler
func (f *Filter) UnmarshalBinary(data []byte) (err error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	buf := bytes.NewBuffer(data)

	var k uint64
	k, f.n, f.m, err = unmarshalBinaryHeader(buf)
	if err != nil {
		return err
	}

	f.keys, err = unmarshalBinaryKeys(buf, k)
	if err != nil {
		return err
	}

	f.bits, err = unmarshalBinaryBits(buf, f.m)
	if err != nil {
		return err
	}

	return checkBinaryHash(buf, data)
}
