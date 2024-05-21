package ethapi

import (
	"bytes"

	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

// proofPairList implements ethdb.KeyValueWriter and collects the proofs as
// hex-strings of key and value for delivery to rpc-caller.
type proofPairList struct {
	keys   []string
	values []string
}

func (n *proofPairList) Put(key []byte, value []byte) error {
	n.keys = append(n.keys, hexutil.Encode(key))
	n.values = append(n.values, hexutil.Encode(value))
	return nil
}

func (n *proofPairList) Delete(key []byte) error {
	panic("not supported")
}

// modified from core/types/derive_sha.go
func deriveTrie(list types.DerivableList) *trie.Trie {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie
}
