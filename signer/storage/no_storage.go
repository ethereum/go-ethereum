package storage

import "errors"

// NoStorageAPI is a dummy construct which doesn't remember anything you tell it
type NoStorageAPI struct{}

// Put is a dummy function that do nothing
func (s *NoStorageAPI) Put(key, value string) {}

// Del is a dummy function that do nothing
func (s *NoStorageAPI) Del(key string) {}

// Get is a dummy function that do nothing
func (s *NoStorageAPI) Get(key string) (string, error) {
	return "", errors.New("missing key, I probably forgot")
}
