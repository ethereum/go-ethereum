package trie

import (
	"bytes"
	"errors"
)

var (
	ErrEmptyLeafKey             = errors.New("the leaf node has empty key")
	ErrDifferentLeafPrefix      = errors.New("the leaf prifix path is different")
	ErrEmptyExtensionPrefix     = errors.New("the extension node has empty prefix")
	ErrDifferentExtensionPrefix = errors.New("the prifix path of extension node is different")
)

func DecodeTrieNode(hash, buf []byte) (node, error) {
	return decodeNodeUnsafe(hash, buf)
}

func TraverseTrieNode(node node, path []byte) ([]byte, []byte, error) {
	switch v := node.(type) {
	case *fullNode:
		first := path[0]
		remainingPath := path[1:]
		return TraverseTrieNode(v.Children[first], remainingPath)
	case *shortNode:
		length := len(v.Key)
		lastKey := v.Key[length-1]
		prePath := v.Key[:length-1]
		if lastKey == 16 {
			if len(prePath) == 0 {
				return nil, nil, ErrEmptyLeafKey
			}
			if !bytes.Equal(prePath, path) {
				return nil, nil, ErrDifferentLeafPrefix
			}
			return (v.Val).(valueNode), path, nil
		} else {
			for index, key := range v.Key {
				if path[index] != key {
					return nil, nil, ErrDifferentExtensionPrefix
				}
			}
			return TraverseTrieNode(v.Val, path[len(v.Key):])
		}
	case hashNode:
		return v, path, nil
	}
	return nil, nil, errors.New("unknown type")
}
