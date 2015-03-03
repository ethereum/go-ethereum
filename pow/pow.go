package pow

type PoW interface {
	Search(block Block, stop <-chan struct{}) ([]byte, []byte, []byte)
	Verify(block Block) bool
	GetHashrate() int64
	Turbo(bool)
}
