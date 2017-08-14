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
	"bytes"
	"encoding/json"
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
	testGetEntry(t, "read", "read", "readme.md", "readit.md")
	testGetEntry(t, "rf", "-", "readme.md", "readit.md")
	testGetEntry(t, "readme", "readme", "readme.md")
	testGetEntry(t, "readme", "-", "readit.md")
	testGetEntry(t, "readme.md", "readme.md", "readme.md")
	testGetEntry(t, "readme.md", "-", "readit.md")
	testGetEntry(t, "readmeAmd", "-", "readit.md")
	testGetEntry(t, "readme.mdffff", "-", "readme.md")
	testGetEntry(t, "ab", "ab", "ab/cefg", "ab/cedh", "ab/kkkkkk")
	testGetEntry(t, "ab/ce", "ab/ce", "ab/cefg", "ab/cedh", "ab/ceuuuuuuuuuu")
	testGetEntry(t, "abc", "abc", "abcd", "abczzzzef", "abc/def", "abc/e/g")
	testGetEntry(t, "a/b", "a/b", "a", "a/bc", "a/ba", "a/b/c")
	testGetEntry(t, "a/b", "a/b", "a", "a/b", "a/bb", "a/b/c")
	testGetEntry(t, "//a//b//", "a/b", "a", "a/b", "a/bb", "a/b/c")
}
func TestDeleteEntry(t *testing.T) {

}

// TestAddFileWithManifestPath tests that adding an entry at a path which
// already exists as a manifest just adds the entry to the manifest rather
// than replacing the manifest with the entry
func TestAddFileWithManifestPath(t *testing.T) {
	// create a manifest containing "ab" and "ac"
	manifest, _ := json.Marshal(&Manifest{
		Entries: []ManifestEntry{
			{Path: "ab", Hash: "ab"},
			{Path: "ac", Hash: "ac"},
		},
	})
	reader := &storage.LazyTestSectionReader{
		SectionReader: io.NewSectionReader(bytes.NewReader(manifest), 0, int64(len(manifest))),
	}
	trie, err := readManifest(reader, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkEntry(t, "ab", "ab", trie)
	checkEntry(t, "ac", "ac", trie)

	// now add path "a" and check we can still get "ab" and "ac"
	entry := &manifestTrieEntry{}
	entry.Path = "a"
	entry.Hash = "a"
	trie.addEntry(entry, nil)
	checkEntry(t, "ab", "ab", trie)
	checkEntry(t, "ac", "ac", trie)
	checkEntry(t, "a", "a", trie)
}
