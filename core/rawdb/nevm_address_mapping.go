package rawdb

import (
	"github.com/ethereum/go-ethereum/common"
	"encoding/binary"
)

type NEVMAddressMapping struct {
	AddressMappings map[common.Address]uint32 // Map from address to collateral height
}

// NewNEVMAddressMapping creates a new NEVMAddressMapping instance
func NewNEVMAddressMapping() *NEVMAddressMapping {
	return &NEVMAddressMapping{
		AddressMappings: make(map[common.Address]uint32),
	}
}

// AddNEVMAddress adds a new NEVM address to the mapping
func (m *NEVMAddressMapping) AddNEVMAddress(address common.Address, collateralHeight uint32) {
	m.AddressMappings[address] = collateralHeight
}

// UpdateNEVMAddress updates an existing NEVM address with a new address
func (m *NEVMAddressMapping) UpdateNEVMAddress(oldAddress common.Address, newAddress common.Address) {
	if height, exists := m.AddressMappings[oldAddress]; exists {
		m.AddressMappings[newAddress] = height
		delete(m.AddressMappings, oldAddress)
	}
}

// RemoveNEVMAddress removes an NEVM address from the mapping
func (m *NEVMAddressMapping) RemoveNEVMAddress(address common.Address) {
	delete(m.AddressMappings, address)
}

// GetNEVMAddress returns the collateral height for a given NEVM address
func (m *NEVMAddressMapping) GetNEVMAddress(address common.Address) []byte {
	collateralHeight, exists := m.AddressMappings[address]
	if exists {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, collateralHeight)
		return buf
	} else {
		return []byte{}
	}
}
