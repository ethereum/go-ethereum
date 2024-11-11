package trie

type bytepool struct {
	c chan []byte
	w int
	h int
}

func newByteslicepool(sliceCap, nitems int) *bytepool {
	b := &bytepool{
		c: make(chan []byte, nitems),
		w: sliceCap,
	}
	return b
}

func (bp *bytepool) Get() []byte {
	select {
	case b := <-bp.c:
		return b
	default:
		return make([]byte, 0, bp.w)
	}
}

func (bp *bytepool) Put(b []byte) {
	// Ignore too small slices
	if cap(b) < bp.w {
		return
	}
	// Don't retain too large slices either
	if cap(b) > 3*bp.w {
		return
	}
	select {
	case bp.c <- b:
	default:
	}
}
