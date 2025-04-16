package vm

import (
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// StoreBytes stores an arbitrary-length []byte into StateDB using multiple storage slots.
func StoreBytes(statedb StateDB, addr common.Address, key common.Hash, value []byte) {
	// Compute the storage root key (keccak256(key)) as the base
	lengthKey := crypto.Keccak256Hash(key.Bytes())
	statedb.SetState(addr, lengthKey, common.BigToHash(big.NewInt(int64(len(value)))))

	// Split value into 32-byte chunks and store them
	for i := 0; i < len(value); i += 32 {
		end := i + 32
		if end > len(value) {
			end = len(value)
		}
		chunk := make([]byte, 32)
		copy(chunk, value[i:end]) // Copy to ensure 32 bytes, padding with zeros

		// Compute storage key: keccak256(lengthKey || index)
		storageKey := crypto.Keccak256Hash(append(lengthKey.Bytes(), common.Uint64ToBytes(uint64(i/32))...))
		statedb.SetState(addr, storageKey, common.BytesToHash(chunk))
	}
}

// LoadBytes retrieves an arbitrary-length []byte from StateDB.
func LoadBytes(statedb StateDB, addr common.Address, key common.Hash) []byte {
	// Read length
	lengthKey := crypto.Keccak256Hash(key.Bytes())
	length := statedb.GetState(addr, lengthKey).Big().Uint64()

	// Read stored chunks
	var buffer bytes.Buffer
	for i := uint64(0); i < length; i += 32 {
		storageKey := crypto.Keccak256Hash(append(lengthKey.Bytes(), common.Uint64ToBytes(uint64(i/32))...))
		chunk := statedb.GetState(addr, storageKey).Bytes()
		buffer.Write(chunk)
	}
	return buffer.Bytes()[:length] // Trim padding
}
