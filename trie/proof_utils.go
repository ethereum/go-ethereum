package trie

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

// StoreBytesProofs stores proofs in a key value database.
func StoreBytesProofs(proofs [][]byte, db ethdb.KeyValueWriter) error {
	for _, node := range proofs {
		hash := crypto.Keccak256(node)
		if err := db.Put(hash, node); err != nil {
			return fmt.Errorf("failed to store proof node %q: %w", node, err)
		}
	}
	return nil
}

// StoreHexProofs stores proofs in a key value database.
func StoreHexProofs(proofs []string, db ethdb.KeyValueWriter) error {
	byteProofs := make([][]byte, len(proofs))
	for i, node := range proofs {
		byteProof, err := hexutil.Decode(node)
		if err != nil {
			return fmt.Errorf("failed to decode proof node %q: %w", node, err)
		}
		byteProofs[i] = byteProof
	}
	return StoreBytesProofs(byteProofs, db)
}
