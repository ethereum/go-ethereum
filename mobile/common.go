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

// Contains all the wrappers from the common package.

package geth

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/hexutil"
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash struct {
	hash common.Hash
}

// NewHashFromBytes converts a slice of bytes to a hash value.
func NewHashFromBytes(binary []byte) (hash *Hash, _ error) {
	h := new(Hash)
	if err := h.SetBytes(common.CopyBytes(binary)); err != nil {
		return nil, err
	}
	return h, nil
}

// NewHashFromHex converts a hex string to a hash value.
func NewHashFromHex(hex string) (hash *Hash, _ error) {
	h := new(Hash)
	if err := h.SetHex(hex); err != nil {
		return nil, err
	}
	return h, nil
}

// SetBytes sets the specified slice of bytes as the hash value.
func (h *Hash) SetBytes(hash []byte) error {
	if length := len(hash); length != common.HashLength {
		return fmt.Errorf("invalid hash length: %v != %v", length, common.HashLength)
	}
	copy(h.hash[:], hash)
	return nil
}

// GetBytes retrieves the byte representation of the hash.
func (h *Hash) GetBytes() []byte {
	return h.hash[:]
}

// SetHex sets the specified hex string as the hash value.
func (h *Hash) SetHex(hash string) error {
	hash = strings.ToLower(hash)
	if len(hash) >= 2 && hash[:2] == "0x" {
		hash = hash[2:]
	}
	if length := len(hash); length != 2*common.HashLength {
		return fmt.Errorf("invalid hash hex length: %v != %v", length, 2*common.HashLength)
	}
	bin, err := hex.DecodeString(hash)
	if err != nil {
		return err
	}
	copy(h.hash[:], bin)
	return nil
}

// GetHex retrieves the hex string representation of the hash.
func (h *Hash) GetHex() string {
	return h.hash.Hex()
}

// Hashes represents a slice of hashes.
type Hashes struct{ hashes []common.Hash }

// NewHashes creates a slice of uninitialized Hashes.
func NewHashes(size int) *Hashes {
	return &Hashes{
		hashes: make([]common.Hash, size),
	}
}

// NewHashesEmpty creates an empty slice of Hashes values.
func NewHashesEmpty() *Hashes {
	return NewHashes(0)
}

// Size returns the number of hashes in the slice.
func (h *Hashes) Size() int {
	return len(h.hashes)
}

// Get returns the hash at the given index from the slice.
func (h *Hashes) Get(index int) (hash *Hash, _ error) {
	if index < 0 || index >= len(h.hashes) {
		return nil, errors.New("index out of bounds")
	}
	return &Hash{h.hashes[index]}, nil
}

// Set sets the Hash at the given index in the slice.
func (h *Hashes) Set(index int, hash *Hash) error {
	if index < 0 || index >= len(h.hashes) {
		return errors.New("index out of bounds")
	}
	h.hashes[index] = hash.hash
	return nil
}

// Append adds a new Hash element to the end of the slice.
func (h *Hashes) Append(hash *Hash) {
	h.hashes = append(h.hashes, hash.hash)
}

// Address represents the 20 byte address of an Ethereum account.
type Address struct {
	address common.Address
}

// NewAddressFromBytes converts a slice of bytes to a hash value.
func NewAddressFromBytes(binary []byte) (address *Address, _ error) {
	a := new(Address)
	if err := a.SetBytes(common.CopyBytes(binary)); err != nil {
		return nil, err
	}
	return a, nil
}

// NewAddressFromHex converts a hex string to a address value.
func NewAddressFromHex(hex string) (address *Address, _ error) {
	a := new(Address)
	if err := a.SetHex(hex); err != nil {
		return nil, err
	}
	return a, nil
}

// SetBytes sets the specified slice of bytes as the address value.
func (a *Address) SetBytes(address []byte) error {
	if length := len(address); length != common.AddressLength {
		return fmt.Errorf("invalid address length: %v != %v", length, common.AddressLength)
	}
	copy(a.address[:], address)
	return nil
}

// GetBytes retrieves the byte representation of the address.
func (a *Address) GetBytes() []byte {
	return a.address[:]
}

// SetHex sets the specified hex string as the address value.
func (a *Address) SetHex(address string) error {
	address = strings.ToLower(address)
	if len(address) >= 2 && address[:2] == "0x" {
		address = address[2:]
	}
	if length := len(address); length != 2*common.AddressLength {
		return fmt.Errorf("invalid address hex length: %v != %v", length, 2*common.AddressLength)
	}
	bin, err := hex.DecodeString(address)
	if err != nil {
		return err
	}
	copy(a.address[:], bin)
	return nil
}

// GetHex retrieves the hex string representation of the address.
func (a *Address) GetHex() string {
	return a.address.Hex()
}

// Addresses represents a slice of addresses.
type Addresses struct{ addresses []common.Address }

// NewAddresses creates a slice of uninitialized addresses.
func NewAddresses(size int) *Addresses {
	return &Addresses{
		addresses: make([]common.Address, size),
	}
}

// NewAddressesEmpty creates an empty slice of Addresses values.
func NewAddressesEmpty() *Addresses {
	return NewAddresses(0)
}

// Size returns the number of addresses in the slice.
func (a *Addresses) Size() int {
	return len(a.addresses)
}

// Get returns the address at the given index from the slice.
func (a *Addresses) Get(index int) (address *Address, _ error) {
	if index < 0 || index >= len(a.addresses) {
		return nil, errors.New("index out of bounds")
	}
	return &Address{a.addresses[index]}, nil
}

// Set sets the address at the given index in the slice.
func (a *Addresses) Set(index int, address *Address) error {
	if index < 0 || index >= len(a.addresses) {
		return errors.New("index out of bounds")
	}
	a.addresses[index] = address.address
	return nil
}

// Append adds a new address element to the end of the slice.
func (a *Addresses) Append(address *Address) {
	a.addresses = append(a.addresses, address.address)
}

// EncodeToHex encodes b as a hex string with 0x prefix.
func EncodeToHex(b []byte) string {
	return hexutil.Encode(b)
}

// DecodeFromHex decodes a hex string with 0x prefix.
func DecodeFromHex(s string) ([]byte, error) {
	return hexutil.Decode(s)
}
