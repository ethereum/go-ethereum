package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

const checksumLen = 4

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(Sha3(data)[12:])
}

// ChecksumAddress creates modifies an address to include a checksum
func ChecksumAddress(addr common.Address) common.ChecksumAddress {
	var checksumaddr common.ChecksumAddress
	checksumaddr.Address = addr
	copy(checksumaddr.Checksum[:], Sha3(addr.Bytes())[:4])

	return checksumaddr
}

// ValidateAddress validates the given address. It must include a checksum
// on the end. It returns the address or an error if the checksum or length of
// the input failed.
func ValidateAddress(addr common.ChecksumAddress) (common.Address, error) {
	if !bytes.Equal(Sha3(addr.Address[:])[:4], addr.Checksum[:]) {
		return common.Address{}, fmt.Errorf("checksum failed for address %x (%x)", addr.Address, addr.Checksum)
	}

	return addr.Address, nil
}

// PubkeyToAddress converts a std ecdsa public key to a common.Address
func PubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := FromECDSAPub(&p)
	return common.BytesToAddress(Sha3(pubBytes[1:])[12:])
}
