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

package client

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/testutil"
)

func TestClientManifestFileList(t *testing.T) {
	srv := testutil.NewTestSwarmServer(t)
	defer srv.Close()

	dir, err := ioutil.TempDir("", "swarm-client-test")
	if err != nil {
		t.Fatal(err)
	}
	files := []string{
		"file1.txt",
		"file2.txt",
		"dir1/file3.txt",
		"dir1/file4.txt",
		"dir2/file5.txt",
		"dir2/dir3/file6.txt",
		"dir2/dir4/file7.txt",
		"dir2/dir4/file8.txt",
	}
	for _, file := range files {
		path := filepath.Join(dir, file)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("error creating dir for %s: %s", path, err)
		}
		if err := ioutil.WriteFile(path, []byte("data"), 0644); err != nil {
			t.Fatalf("error writing file %s: %s", path, err)
		}
	}

	client := NewClient(srv.URL)

	hash, err := client.UploadDirectory(dir, "")
	if err != nil {
		t.Fatalf("error uploading directory: %s", err)
	}

	ls := func(prefix string) []string {
		entries, err := client.ManifestFileList(hash, prefix)
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, len(entries))
		for i, entry := range entries {
			paths[i] = entry.Path
		}
		sort.Strings(paths)
		return paths
	}

	tests := map[string][]string{
		"":                    []string{"dir1/", "dir2/", "file1.txt", "file2.txt"},
		"file":                []string{"file1.txt", "file2.txt"},
		"file1":               []string{"file1.txt"},
		"file2.txt":           []string{"file2.txt"},
		"file12":              []string{},
		"dir":                 []string{"dir1/", "dir2/"},
		"dir1":                []string{"dir1/"},
		"dir1/":               []string{"dir1/file3.txt", "dir1/file4.txt"},
		"dir1/file":           []string{"dir1/file3.txt", "dir1/file4.txt"},
		"dir1/file3.txt":      []string{"dir1/file3.txt"},
		"dir1/file34":         []string{},
		"dir2/":               []string{"dir2/dir3/", "dir2/dir4/", "dir2/file5.txt"},
		"dir2/file":           []string{"dir2/file5.txt"},
		"dir2/dir":            []string{"dir2/dir3/", "dir2/dir4/"},
		"dir2/dir3/":          []string{"dir2/dir3/file6.txt"},
		"dir2/dir4/":          []string{"dir2/dir4/file7.txt", "dir2/dir4/file8.txt"},
		"dir2/dir4/file":      []string{"dir2/dir4/file7.txt", "dir2/dir4/file8.txt"},
		"dir2/dir4/file7.txt": []string{"dir2/dir4/file7.txt"},
		"dir2/dir4/file78":    []string{},
	}
	for prefix, expected := range tests {
		actual := ls(prefix)
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected prefix %q to return paths %v, got %v", prefix, expected, actual)
		}
	}
}
