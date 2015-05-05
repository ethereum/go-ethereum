package pow

type PoW interface {
	Search(block Block, stop <-chan struct{}) (uint64, []byte)
	Verify(block Block) bool
	GetHashrate() int64
	Turbo(bool)
}
