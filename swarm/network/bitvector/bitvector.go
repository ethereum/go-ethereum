package bitvector

type BitVector struct {
	len int
	b   []byte
}

func New(l int) *BitVector {
	return NewFromBytes(make([]byte, l/8+1), l)
}

func NewFromBytes(b []byte, l int) *BitVector {
	return &BitVector{
		len: l,
		b:   b,
	}
}

func (bv *BitVector) Get(i int) bool {
	bi := i / 8
	return uint8(bv.b[bi])&0x1>>uint(i%8) != 0
}

func (bv *BitVector) Set(i int, v bool) {
	bi := i / 8
	cv := bv.Get(i)
	if cv != v {
		bv.b[bi] ^= 0x1 >> uint8(i%8)
	}
}

func (bv *BitVector) Bytes() []byte {
	return bv.b
}
