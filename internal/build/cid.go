// Copyright 2024 The go-ethereum Authors
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

package build

import (
	"crypto/sha256"
	"encoding/base32"
	"io"
	"math/big"
	"os"
	"strings"
)

// CID represents an IPFS Content Identifier for raw file content.
type CID struct {
	// V1 is the CIDv1 with raw codec: bafkrei... (base32lower, 59 chars)
	// This is the canonical format for raw binary content.
	V1 string

	// Multihash is the raw SHA256 multihash (base58btc encoded): Qm... (46 chars)
	// Note: This is NOT a valid CIDv0 for raw content (CIDv0 requires dag-pb codec).
	// However, it's included for compatibility with tools that expect Qm... format.
	// To get the actual content, use the V1 CID or convert: ipfs cid format -v 1 <multihash>
	Multihash string
}

// ComputeFileCID computes the IPFS CID for a file's raw content.
//
// The CID is computed using SHA256 and the raw multicodec (0x55), which means
// the hash is of the file's exact bytes with no wrapping or chunking.
//
// Returns CIDv1 (bafkrei...) as the primary identifier, plus the base58-encoded
// multihash for compatibility with legacy tooling.
//
// Verify with: ipfs add --only-hash --raw-leaves -Q <file>
func ComputeFileCID(path string) (*CID, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ComputeCID(f)
}

// ComputeCID computes the IPFS CID from a reader's content.
func ComputeCID(r io.Reader) (*CID, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return nil, err
	}
	digest := h.Sum(nil)

	// Build multihash: 0x12 (SHA256) + 0x20 (32 bytes length) + digest
	multihash := make([]byte, 0, 34)
	multihash = append(multihash, 0x12) // SHA256 multicodec
	multihash = append(multihash, 0x20) // 32 bytes
	multihash = append(multihash, digest...)

	// Base58-encoded multihash (Qm... format, for legacy compatibility)
	mhBase58 := base58Encode(multihash)

	// CIDv1 = 'b' + base32lower(0x01 + 0x55 + multihash)
	// 0x01 = CIDv1, 0x55 = raw multicodec
	cidv1Bytes := make([]byte, 0, 36)
	cidv1Bytes = append(cidv1Bytes, 0x01) // CID version 1
	cidv1Bytes = append(cidv1Bytes, 0x55) // raw codec
	cidv1Bytes = append(cidv1Bytes, multihash...)

	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(cidv1Bytes)
	cidv1 := "b" + strings.ToLower(encoded)

	return &CID{V1: cidv1, Multihash: mhBase58}, nil
}

// base58Encode encodes bytes using Bitcoin's base58 alphabet.
// This is used for IPFS CIDv0 encoding.
func base58Encode(data []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Count leading zeros
	var zeros int
	for _, b := range data {
		if b != 0 {
			break
		}
		zeros++
	}

	// Convert to big integer
	num := new(big.Int).SetBytes(data)
	base := big.NewInt(58)
	mod := new(big.Int)

	// Build result in reverse
	var result []byte
	for num.Sign() > 0 {
		num.DivMod(num, base, mod)
		result = append(result, alphabet[mod.Int64()])
	}

	// Add leading '1's for each leading zero byte
	for i := 0; i < zeros; i++ {
		result = append(result, '1')
	}

	// Reverse the result
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}
