package bzz

import (
	// "encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func manifest(paths ...string) (manifestReader SectionReader) {
	var entries []string
	for _, path := range paths {
		entry := fmt.Sprintf(`{"path":"%s"}`, path)
		entries = append(entries, entry)
	}
	manifest := fmt.Sprintf(`{"entries":[%s]}`, strings.Join(entries, ","))
	return io.NewSectionReader(strings.NewReader(manifest), 0, int64(len(manifest)))
}

func testGetEntry(t *testing.T, path, match string, paths ...string) *manifestTrie {
	trie, err := readManifest(manifest(paths...), nil, nil)
	if err != nil {
		t.Errorf("unexpected error making manifest: %v", err)
	}
	checkEntry(t, path, match, trie)
	return trie
}

func checkEntry(t *testing.T, path, match string, trie *manifestTrie) {
	entry, _ := trie.getEntry(path)
	if match == "-" && entry != nil {
		t.Errorf("expected no match for '%s', got '%s'", path, entry.Path)
	} else if entry == nil {
		t.Errorf("expected entry '%s' to match '%s', got no match", match, path)
	} else if entry.Path != match {
		t.Errorf("incorrect entry retrieved for '%s'. expected path '%v', got '%s'", path, match, entry.Path)
	}
}

func TestGetEntry(t *testing.T) {
	testGetEntry(t, "a", "a", "a")
	testGetEntry(t, "b", "-", "a")
	testGetEntry(t, "/a", "/a", "/a")
	testGetEntry(t, "/a", "///a", "///a")
	testGetEntry(t, "/a", "a", "a")
	// fallback
	testGetEntry(t, "/a", "/", "/")
	testGetEntry(t, "a", "/", "/")
	testGetEntry(t, "/a", "", "")
	// longest/deepest math
	testGetEntry(t, "a/b", "a/b", "a///", "a/b")
	// trailing slash
	testGetEntry(t, "", "", "/", "")
	testGetEntry(t, "/", "/", "/", "")
	testGetEntry(t, "a", "a", "a", "a/")
	testGetEntry(t, "a/", "a/", "a/", "a")
	// prefix match
	testGetEntry(t, "a", "a", "a", "ab")
	testGetEntry(t, "ab", "", "a", "")
	testGetEntry(t, "a", "a", "a", "ab")
	testGetEntry(t, "a/b", "a", "a", "ab")
}

func TestDeleteEntry(t *testing.T) {

}
