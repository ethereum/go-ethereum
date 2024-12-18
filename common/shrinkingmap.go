package common

// ShrinkingMap is a map that shrinks itself (by allocating a new map) after a certain number of deletions have been performed.
// If shrinkAfterDeletionsCount is set to <=0, the map will never shrink.
// This is useful to prevent memory leaks in long-running processes that delete a lot of keys from a map.
// See here for more details: https://github.com/golang/go/issues/20135
type ShrinkingMap[K comparable, V any] struct {
	m           map[K]V
	deletedKeys int

	shrinkAfterDeletionsCount int
}

func NewShrinkingMap[K comparable, V any](shrinkAfterDeletionsCount int) *ShrinkingMap[K, V] {
	return &ShrinkingMap[K, V]{
		m:                         make(map[K]V),
		shrinkAfterDeletionsCount: shrinkAfterDeletionsCount,
	}
}

func (s *ShrinkingMap[K, V]) Set(key K, value V) {
	s.m[key] = value
}

func (s *ShrinkingMap[K, V]) Get(key K) (value V, exists bool) {
	value, exists = s.m[key]
	return value, exists
}

func (s *ShrinkingMap[K, V]) Has(key K) bool {
	_, exists := s.m[key]
	return exists
}

func (s *ShrinkingMap[K, V]) Delete(key K) (deleted bool) {
	if _, exists := s.m[key]; !exists {
		return false
	}

	delete(s.m, key)
	s.deletedKeys++

	if s.shouldShrink() {
		s.shrink()
	}

	return true
}

func (s *ShrinkingMap[K, V]) Size() (size int) {
	return len(s.m)
}

func (s *ShrinkingMap[K, V]) Clear() {
	s.m = make(map[K]V)
	s.deletedKeys = 0
}

func (s *ShrinkingMap[K, V]) shouldShrink() bool {
	return s.shrinkAfterDeletionsCount > 0 && s.deletedKeys >= s.shrinkAfterDeletionsCount
}

func (s *ShrinkingMap[K, V]) shrink() {
	newMap := make(map[K]V, len(s.m))
	for k, v := range s.m {
		newMap[k] = v
	}

	s.m = newMap
	s.deletedKeys = 0
}
