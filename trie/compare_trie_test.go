// --- Start fork code ---
package trie

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type witness struct {
	PreRoot  common.Hash     `json:"preRoot"`
	PostRoot common.Hash     `json:"postRoot"`
	State    []hexutil.Bytes `json:"state"`
}

func TestCompareTrie(t *testing.T) {
	// Load the witness data from witness.json
	w := new(witness)
	f, err := os.Open("witness.json")
	require.NoError(t, err)
	defer f.Close()

	err = json.NewDecoder(f).Decode(w)
	require.NoError(t, err)

	proofDB := memorydb.New()
	for _, n := range w.State {
		proofDB.Put(crypto.Keccak256(n), n)
	}

	diffs, err := CompareTrie(w.PreRoot, w.PostRoot, proofDB)
	require.NoError(t, err)
	assert.Equal(t, len(diffs), 0)
}

// --- End fork code ---
