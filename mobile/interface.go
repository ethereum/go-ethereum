// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Contains perverted wrappers to allow crossing over empty interfaces.

package geth

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Interface represents a wrapped version of Go's interface{}, with the capacity
// to store arbitrary data types.
//
// Since it's impossible to get the arbitrary-ness converted between Go and mobile
// platforms, we're using explicit getters and setters for the conversions. There
// is of course no point in enumerating everything, just enough to support the
// contract bindins requiring client side generated code.
type Interface struct {
	object interface{}
}

// NewInterface creates a new empty interface that can be used to pass around
// generic types.
func NewInterface() *Interface {
	return new(Interface)
}

func (i *Interface) SetBool(b bool)                { i.object = &b }
func (i *Interface) SetBools(bs []bool)            { i.object = &bs }
func (i *Interface) SetString(str string)          { i.object = &str }
func (i *Interface) SetStrings(strs *Strings)      { i.object = &strs.strs }
func (i *Interface) SetBinary(binary []byte)       { b := common.CopyBytes(binary); i.object = &b }
func (i *Interface) SetBinaries(binaries [][]byte) { i.object = &binaries }
func (i *Interface) SetAddress(address *Address)   { i.object = &address.address }
func (i *Interface) SetAddresses(addrs *Addresses) { i.object = &addrs.addresses }
func (i *Interface) SetHash(hash *Hash)            { i.object = &hash.hash }
func (i *Interface) SetHashes(hashes *Hashes)      { i.object = &hashes.hashes }
func (i *Interface) SetInt8(n int8)                { i.object = &n }
func (i *Interface) SetInt16(n int16)              { i.object = &n }
func (i *Interface) SetInt32(n int32)              { i.object = &n }
func (i *Interface) SetInt64(n int64)              { i.object = &n }
func (i *Interface) SetUint8(bigint *BigInt)       { n := uint8(bigint.bigint.Uint64()); i.object = &n }
func (i *Interface) SetUint16(bigint *BigInt)      { n := uint16(bigint.bigint.Uint64()); i.object = &n }
func (i *Interface) SetUint32(bigint *BigInt)      { n := uint32(bigint.bigint.Uint64()); i.object = &n }
func (i *Interface) SetUint64(bigint *BigInt)      { n := uint64(bigint.bigint.Uint64()); i.object = &n }
func (i *Interface) SetBigInt(bigint *BigInt)      { i.object = &bigint.bigint }
func (i *Interface) SetBigInts(bigints *BigInts)   { i.object = &bigints.bigints }

func (i *Interface) SetDefaultBool()      { i.object = new(bool) }
func (i *Interface) SetDefaultBools()     { i.object = new([]bool) }
func (i *Interface) SetDefaultString()    { i.object = new(string) }
func (i *Interface) SetDefaultStrings()   { i.object = new([]string) }
func (i *Interface) SetDefaultBinary()    { i.object = new([]byte) }
func (i *Interface) SetDefaultBinaries()  { i.object = new([][]byte) }
func (i *Interface) SetDefaultAddress()   { i.object = new(common.Address) }
func (i *Interface) SetDefaultAddresses() { i.object = new([]common.Address) }
func (i *Interface) SetDefaultHash()      { i.object = new(common.Hash) }
func (i *Interface) SetDefaultHashes()    { i.object = new([]common.Hash) }
func (i *Interface) SetDefaultInt8()      { i.object = new(int8) }
func (i *Interface) SetDefaultInt16()     { i.object = new(int16) }
func (i *Interface) SetDefaultInt32()     { i.object = new(int32) }
func (i *Interface) SetDefaultInt64()     { i.object = new(int64) }
func (i *Interface) SetDefaultUint8()     { i.object = new(uint8) }
func (i *Interface) SetDefaultUint16()    { i.object = new(uint16) }
func (i *Interface) SetDefaultUint32()    { i.object = new(uint32) }
func (i *Interface) SetDefaultUint64()    { i.object = new(uint64) }
func (i *Interface) SetDefaultBigInt()    { i.object = new(*big.Int) }
func (i *Interface) SetDefaultBigInts()   { i.object = new([]*big.Int) }

func (i *Interface) GetBool() bool            { return *i.object.(*bool) }
func (i *Interface) GetBools() []bool         { return *i.object.(*[]bool) }
func (i *Interface) GetString() string        { return *i.object.(*string) }
func (i *Interface) GetStrings() *Strings     { return &Strings{*i.object.(*[]string)} }
func (i *Interface) GetBinary() []byte        { return *i.object.(*[]byte) }
func (i *Interface) GetBinaries() [][]byte    { return *i.object.(*[][]byte) }
func (i *Interface) GetAddress() *Address     { return &Address{*i.object.(*common.Address)} }
func (i *Interface) GetAddresses() *Addresses { return &Addresses{*i.object.(*[]common.Address)} }
func (i *Interface) GetHash() *Hash           { return &Hash{*i.object.(*common.Hash)} }
func (i *Interface) GetHashes() *Hashes       { return &Hashes{*i.object.(*[]common.Hash)} }
func (i *Interface) GetInt8() int8            { return *i.object.(*int8) }
func (i *Interface) GetInt16() int16          { return *i.object.(*int16) }
func (i *Interface) GetInt32() int32          { return *i.object.(*int32) }
func (i *Interface) GetInt64() int64          { return *i.object.(*int64) }
func (i *Interface) GetUint8() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*i.object.(*uint8)))}
}
func (i *Interface) GetUint16() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*i.object.(*uint16)))}
}
func (i *Interface) GetUint32() *BigInt {
	return &BigInt{new(big.Int).SetUint64(uint64(*i.object.(*uint32)))}
}
func (i *Interface) GetUint64() *BigInt {
	return &BigInt{new(big.Int).SetUint64(*i.object.(*uint64))}
}
func (i *Interface) GetBigInt() *BigInt   { return &BigInt{*i.object.(**big.Int)} }
func (i *Interface) GetBigInts() *BigInts { return &BigInts{*i.object.(*[]*big.Int)} }

// Interfaces is a slices of wrapped generic objects.
type Interfaces struct {
	objects []interface{}
}

// NewInterfaces creates a slice of uninitialized interfaces.
func NewInterfaces(size int) *Interfaces {
	return &Interfaces{
		objects: make([]interface{}, size),
	}
}

// Size returns the number of interfaces in the slice.
func (i *Interfaces) Size() int {
	return len(i.objects)
}

// Get returns the bigint at the given index from the slice.
func (i *Interfaces) Get(index int) (iface *Interface, _ error) {
	if index < 0 || index >= len(i.objects) {
		return nil, errors.New("index out of bounds")
	}
	return &Interface{i.objects[index]}, nil
}

// Set sets the big int at the given index in the slice.
func (i *Interfaces) Set(index int, object *Interface) error {
	if index < 0 || index >= len(i.objects) {
		return errors.New("index out of bounds")
	}
	i.objects[index] = object.object
	return nil
}
