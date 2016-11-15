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

	"github.com/ethereum/go-ethereum/common"
)

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash struct {
	hash common.Hash
}

// NewHashFromBytes converts a slice of bytes to a hash value.
func NewHashFromBytes(hash []byte) (*Hash, error) {
	h := new(Hash)
	if err := h.SetBytes(hash); err != nil {
		return nil, err
	}
	return h, nil
}

// NewHashFromHex converts a hex string to a hash value.
func NewHashFromHex(hash string) (*Hash, error) {
	h := new(Hash)
	if err := h.SetHex(hash); err != nil {
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

// Size returns the number of hashes in the slice.
func (h *Hashes) Size() int {
	return len(h.hashes)
}

// Get returns the hash at the given index from the slice.
func (h *Hashes) Get(index int) (*Hash, error) {
	if index < 0 || index >= len(h.hashes) {
		return nil, errors.New("index out of bounds")
	}
	return &Hash{h.hashes[index]}, nil
}

// Address represents the 20 byte address of an Ethereum account.
type Address struct {
	address common.Address
}

// NewAddressFromBytes converts a slice of bytes to a hash value.
func NewAddressFromBytes(address []byte) (*Address, error) {
	a := new(Address)
	if err := a.SetBytes(address); err != nil {
		return nil, err
	}
	return a, nil
}

// NewAddressFromHex converts a hex string to a address value.
func NewAddressFromHex(address string) (*Address, error) {
	a := new(Address)
	if err := a.SetHex(address); err != nil {
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

// Size returns the number of addresses in the slice.
func (a *Addresses) Size() int {
	return len(a.addresses)
}

// Get returns the address at the given index from the slice.
func (a *Addresses) Get(index int) (*Address, error) {
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
