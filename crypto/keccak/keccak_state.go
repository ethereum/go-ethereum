package keccak

// KeccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type KeccakState struct {
	state
}

func NewLegacyKeccak256State() *KeccakState {
	return &KeccakState{state{rate: rateK512, outputLen: 32, dsbyte: dsbyteKeccak}}
}
