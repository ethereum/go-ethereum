package bigcache

// Hasher is responsible for generating unsigned, 64 bit hash of provided string. Hasher should minimize collisions
// (generating same hash for different strings) and while performance is also important fast functions are preferable (i.e.
// you can use FarmHash family).
type Hasher interface {
	Sum64(string) uint64
}
