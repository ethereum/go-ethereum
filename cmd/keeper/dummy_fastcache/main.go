package fastcache

type Cache struct{}

func (*Cache) Get(dst, k []byte) []byte            { return nil }
func (*Cache) HasGet(dst, k []byte) ([]byte, bool) { return nil, false }
func (*Cache) Set(k, v []byte)                     {}
func (*Cache) Reset()                              {}
func (*Cache) Del([]byte)                          {}

func New(int) *Cache {
	return &Cache{}
}
