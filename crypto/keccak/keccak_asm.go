//go:build (amd64 || arm64) && !purego

package keccak

import "unsafe"

// sponge is the core Keccak-256 sponge state used by native (asm) implementations.
type sponge struct {
	state     [200]byte
	buf       [rate]byte
	absorbed  int
	squeezing bool
	readIdx   int // index into state for next Read byte
}

// Reset resets the sponge to its initial state.
func (s *sponge) Reset() {
	s.state = [200]byte{}
	s.absorbed = 0
	s.squeezing = false
	s.readIdx = 0
}

// Write absorbs data into the sponge.
// Panics if called after Read.
func (s *sponge) Write(p []byte) (int, error) {
	if s.squeezing {
		panic("keccak: Write after Read")
	}
	n := len(p)
	if s.absorbed > 0 {
		x := copy(s.buf[s.absorbed:rate], p)
		s.absorbed += x
		p = p[x:]
		if s.absorbed == rate {
			xorAndPermute(&s.state, &s.buf[0])
			s.absorbed = 0
		}
	}

	for len(p) >= rate {
		xorAndPermute(&s.state, &p[0])
		p = p[rate:]
	}

	if len(p) > 0 {
		s.absorbed = copy(s.buf[:], p)
	}
	return n, nil
}

// Sum256 finalizes and returns the 32-byte Keccak-256 digest.
// Does not modify the sponge state.
func (s *sponge) Sum256() [32]byte {
	state := s.state
	xorIn(&state, s.buf[:s.absorbed])
	state[s.absorbed] ^= 0x01
	state[rate-1] ^= 0x80
	keccakF1600(&state)
	return [32]byte(state[:32])
}

// Sum appends the current Keccak-256 digest to b and returns the resulting slice.
// Does not modify the sponge state.
func (s *sponge) Sum(b []byte) []byte {
	d := s.Sum256()
	return append(b, d[:]...)
}

// Size returns the number of bytes Sum will produce (32).
func (s *sponge) Size() int { return 32 }

// BlockSize returns the sponge rate in bytes (136).
func (s *sponge) BlockSize() int { return rate }

// Read squeezes an arbitrary number of bytes from the sponge.
// On the first call, it pads and permutes, transitioning from absorbing to squeezing.
// Subsequent calls to Write will panic. It never returns an error.
func (s *sponge) Read(out []byte) (int, error) {
	if !s.squeezing {
		s.padAndSqueeze()
	}

	n := len(out)
	for len(out) > 0 {
		x := copy(out, s.state[s.readIdx:rate])
		s.readIdx += x
		out = out[x:]
		if s.readIdx == rate {
			keccakF1600(&s.state)
			s.readIdx = 0
		}
	}
	return n, nil
}

func (s *sponge) padAndSqueeze() {
	xorIn(&s.state, s.buf[:s.absorbed])
	s.state[s.absorbed] ^= 0x01
	s.state[rate-1] ^= 0x80
	keccakF1600(&s.state)
	s.squeezing = true
	s.readIdx = 0
}

// sum256Sponge computes Keccak-256 in one shot using the assembly permutation.
func sum256Sponge(data []byte) [32]byte {
	var state [200]byte

	for len(data) >= rate {
		xorAndPermute(&state, &data[0])
		data = data[rate:]
	}

	xorIn(&state, data)
	state[len(data)] ^= 0x01
	state[rate-1] ^= 0x80
	keccakF1600(&state)

	return [32]byte(state[:32])
}

func xorIn(state *[200]byte, data []byte) {
	stateU64 := (*[25]uint64)(unsafe.Pointer(state))
	n := len(data) >> 3
	p := unsafe.Pointer(unsafe.SliceData(data))
	for i := range n {
		stateU64[i] ^= *(*uint64)(unsafe.Add(p, uintptr(i)<<3))
	}
	for i := n << 3; i < len(data); i++ {
		state[i] ^= data[i]
	}
}
