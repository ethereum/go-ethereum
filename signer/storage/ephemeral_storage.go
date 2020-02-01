package storage

// EphemeralStorageAPI is an in-memory storage api that does
// not persist values to disk. Mainly used for testing
type EphemeralStorageAPI struct {
	data map[string]string
}

// Put stores a value by key. 0-length keys results in noop.
func (s *EphemeralStorageAPI) Put(key, value string) {
	if len(key) == 0 {
		return
	}
	s.data[key] = value
}

// Get returns the previously stored value, or an error if the key is 0-length
// or unknown.
func (s *EphemeralStorageAPI) Get(key string) (string, error) {
	if len(key) == 0 {
		return "", ErrZeroKey
	}
	if v, ok := s.data[key]; ok {
		return v, nil
	}
	return "", ErrNotFound
}

// Del removes a key-value pair. If the key doesn't exist, the method is a noop.
func (s *EphemeralStorageAPI) Del(key string) {
	delete(s.data, key)
}
