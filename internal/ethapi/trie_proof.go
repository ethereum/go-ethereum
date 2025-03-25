package ethapi

import (
	"bytes"

	"github.com/XinFinOrg/XDPoSChain/common"
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
	buf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := range list.Len() {
		buf.Reset()
		rlp.Encode(buf, uint(i))
		key := common.CopyBytes(buf.Bytes())
		buf.Reset()
		list.EncodeIndex(i, buf)
		value := common.CopyBytes(buf.Bytes())
		trie.Update(key, value)
	}
	return trie
}
