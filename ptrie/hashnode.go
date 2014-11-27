package ptrie

type HashNode struct {
	key []byte
}

func NewHash(key []byte) *HashNode {
	return &HashNode{key}
}

func (self *HashNode) RlpData() interface{} {
	return self.key
}

func (self *HashNode) Hash() interface{} {
	return self.key
}

// These methods will never be called but we have to satisfy Node interface
func (self *HashNode) Value() Node { return nil }
func (self *HashNode) Dirty() bool { return true }
func (self *HashNode) Copy() Node  { return self }
