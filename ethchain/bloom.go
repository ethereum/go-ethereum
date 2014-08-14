package ethchain

type BloomFilter struct {
	bin []byte
}

func NewBloomFilter(bin []byte) *BloomFilter {
	if bin == nil {
		bin = make([]byte, 256)
	}

	return &BloomFilter{
		bin: bin,
	}
}

func (self *BloomFilter) Set(addr []byte) {
	if len(addr) < 8 {
		chainlogger.Warnf("err: bloom set to small: %x\n", addr)

		return
	}

	for _, i := range addr[len(addr)-8:] {
		self.bin[i] = 1
	}
}

func (self *BloomFilter) Search(addr []byte) bool {
	if len(addr) < 8 {
		chainlogger.Warnf("err: bloom search to small: %x\n", addr)

		return false
	}

	for _, i := range addr[len(addr)-8:] {
		if self.bin[i] == 0 {
			return false
		}
	}

	return true
}

func (self *BloomFilter) Bin() []byte {
	return self.bin
}
