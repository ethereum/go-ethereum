package trie

type FullNode struct {
	trie  *Trie
	nodes [17]Node
	dirty bool
}

func NewFullNode(t *Trie) *FullNode {
	return &FullNode{trie: t}
}

func (self *FullNode) Dirty() bool { return self.dirty }
func (self *FullNode) Value() Node {
	self.nodes[16] = self.trie.trans(self.nodes[16])
	return self.nodes[16]
}
func (self *FullNode) Branches() []Node {
	return self.nodes[:16]
}

func (self *FullNode) Copy(t *Trie) Node {
	nnode := NewFullNode(t)
	for i, node := range self.nodes {
		if node != nil {
			nnode.nodes[i] = node.Copy(t)
		}
	}
	nnode.dirty = true

	return nnode
}

// Returns the length of non-nil nodes
func (self *FullNode) Len() (amount int) {
	for _, node := range self.nodes {
		if node != nil {
			amount++
		}
	}

	return
}

func (self *FullNode) Hash() interface{} {
	return self.trie.store(self)
}

func (self *FullNode) RlpData() interface{} {
	t := make([]interface{}, 17)
	for i, node := range self.nodes {
		if node != nil {
			t[i] = node.Hash()
		} else {
			t[i] = ""
		}
	}

	return t
}

func (self *FullNode) set(k byte, value Node) {
	self.nodes[int(k)] = value
	self.dirty = true
}

func (self *FullNode) branch(i byte) Node {
	if self.nodes[int(i)] != nil {
		self.nodes[int(i)] = self.trie.trans(self.nodes[int(i)])

		return self.nodes[int(i)]
	}
	return nil
}

func (self *FullNode) setDirty(dirty bool) {
	self.dirty = dirty
}
