package trie

import "github.com/ethereum/go-ethereum/common"

type ValueNode struct {
	trie *Trie
	data []byte
}

func (self *ValueNode) Value() Node          { return self } // Best not to call :-)
func (self *ValueNode) Val() []byte          { return self.data }
func (self *ValueNode) Dirty() bool          { return true }
func (self *ValueNode) Copy(t *Trie) Node    { return &ValueNode{t, common.CopyBytes(self.data)} }
func (self *ValueNode) RlpData() interface{} { return self.data }
func (self *ValueNode) Hash() interface{}    { return self.data }
