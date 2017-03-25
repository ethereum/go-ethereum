// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package api

import (
	// "encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

func manifest(paths ...string) (manifestReader storage.LazySectionReader) {
	var entries []string
	for _, path := range paths {
		entry := fmt.Sprintf(`{"path":"%s"}`, path)
		entries = append(entries, entry)
	}
	manifest := fmt.Sprintf(`{"entries":[%s]}`, strings.Join(entries, ","))
	return &storage.LazyTestSectionReader{
		SectionReader: io.NewSectionReader(strings.NewReader(manifest), 0, int64(len(manifest))),
	}
}

func testGetEntry(t *testing.T, path, match string, paths ...string) *manifestTrie {
	quitC := make(chan bool)
	trie, err := readManifest(manifest(paths...), nil, nil, quitC)
	if err != nil {
		t.Errorf("unexpected error making manifest: %v", err)
	}
	checkEntry(t, path, match, trie)
	return trie
}

func checkEntry(t *testing.T, path, match string, trie *manifestTrie) {
	entry, fullpath := trie.getEntry(path)
	if match == "-" && entry != nil {
		t.Errorf("expected no match for '%s', got '%s'", path, fullpath)
	} else if entry == nil {
		if match != "-" {
			t.Errorf("expected entry '%s' to match '%s', got no match", match, path)
		}
	} else if fullpath != match {
		t.Errorf("incorrect entry retrieved for '%s'. expected path '%v', got '%s'", path, match, fullpath)
	}
}

func TestGetEntry(t *testing.T) {
	// file system manifest always contains regularized paths
	testGetEntry(t, "a", "a", "a")
	testGetEntry(t, "b", "-", "a")
	testGetEntry(t, "/a//", "a", "a")
	// fallback
	testGetEntry(t, "/a", "", "")
	testGetEntry(t, "/a/b", "a/b", "a/b")
	// longest/deepest math
	testGetEntry(t, "a/b", "-", "a", "a/ba", "a/b/c")
	testGetEntry(t, "a/b", "a/b", "a", "a/b", "a/bb", "a/b/c")
	testGetEntry(t, "//a//b//", "a/b", "a", "a/b", "a/bb", "a/b/c")
}

func TestDeleteEntry(t *testing.T) {

}
