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
	"crypto/sha512"
	"encoding/binary"
)

// conforms to encoding.BinaryMarshaler

// marshalled binary layout (Little Endian):
//
//	 k	1 uint64
//	 n	1 uint64
//	 m	1 uint64
//	 keys	[k]uint64
//	 bits	[(m+63)/64]uint64
//	 hash	sha384 (384 bits == 48 bytes)
//
//	 size = (3 + k + (m+63)/64) * 8 bytes
//

func (f *Filter) marshal() (buf *bytes.Buffer,
	hash [sha512.Size384]byte,
	err error,
) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	debug("write bf k=%d n=%d m=%d\n", f.K(), f.n, f.m)

	buf = new(bytes.Buffer)

	err = binary.Write(buf, binary.LittleEndian, f.K())
	if err != nil {
		return nil, hash, err
	}

	err = binary.Write(buf, binary.LittleEndian, f.n)
	if err != nil {
		return nil, hash, err
	}

	err = binary.Write(buf, binary.LittleEndian, f.m)
	if err != nil {
		return nil, hash, err
	}

	err = binary.Write(buf, binary.LittleEndian, f.keys)
	if err != nil {
		return nil, hash, err
	}

	err = binary.Write(buf, binary.LittleEndian, f.bits)
	if err != nil {
		return nil, hash, err
	}

	hash = sha512.Sum384(buf.Bytes())
	err = binary.Write(buf, binary.LittleEndian, hash)
	return buf, hash, err
}

// MarshalBinary converts a Filter into []bytes
func (f *Filter) MarshalBinary() (data []byte, err error) {
	buf, hash, err := f.marshal()
	if err != nil {
		return nil, err
	}

	debug(
		"bloomfilter.MarshalBinary: Successfully wrote %d byte(s), sha384 %v",
		buf.Len(), hash,
	)
	data = buf.Bytes()
	return data, nil
}
