// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
)

// TestManifestChange tests manifest add, update and remove
// cli commands without encryption.
func TestManifestChange(t *testing.T) {
	testManifestChange(t, false)
}

// TestManifestChange tests manifest add, update and remove
// cli commands with encryption enabled.
func TestManifestChangeEncrypted(t *testing.T) {
	testManifestChange(t, true)
}

// testManifestChange performs cli commands:
// - manifest add
// - manifest update
// - manifest remove
// on a manifest, testing the functionality of this
// comands on paths that are in root manifest or a nested one.
// Argument encrypt controls whether to use encryption or not.
func testManifestChange(t *testing.T, encrypt bool) {
	t.Parallel()
	cluster := newTestCluster(t, 1)
	defer cluster.Shutdown()

	tmp, err := ioutil.TempDir("", "swarm-manifest-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	origDir := filepath.Join(tmp, "orig")
	if err := os.Mkdir(origDir, 0777); err != nil {
		t.Fatal(err)
	}

	indexDataFilename := filepath.Join(origDir, "index.html")
	err = ioutil.WriteFile(indexDataFilename, []byte("<h1>Test</h1>"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	// Files paths robots.txt and robots.html share the same prefix "robots."
	// which will result a manifest with a nested manifest under path "robots.".
	// This will allow testing manifest changes on both root and nested manifest.
	err = ioutil.WriteFile(filepath.Join(origDir, "robots.txt"), []byte("Disallow: /"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(origDir, "robots.html"), []byte("<strong>No Robots Allowed</strong>"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(origDir, "mutants.txt"), []byte("Frank\nMarcus"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--bzzapi",
		cluster.Nodes[0].URL,
		"--recursive",
		"--defaultpath",
		indexDataFilename,
		"up",
		origDir,
	}
	if encrypt {
		args = append(args, "--encrypt")
	}

	origManifestHash := runSwarmExpectHash(t, args...)

	checkHashLength(t, origManifestHash, encrypt)

	client := swarm.NewClient(cluster.Nodes[0].URL)

	// upload a new file and use its manifest to add it the original manifest.
	t.Run("add", func(t *testing.T) {
		humansData := []byte("Ann\nBob")
		humansDataFilename := filepath.Join(tmp, "humans.txt")
		err = ioutil.WriteFile(humansDataFilename, humansData, 0666)
		if err != nil {
			t.Fatal(err)
		}

		humansManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"up",
			humansDataFilename,
		)

		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"add",
			origManifestHash,
			"humans.txt",
			humansManifestHash,
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
		for _, e := range newManifest.Entries {
			if e.Path == "humans.txt" {
				found = true
				if e.Size != int64(len(humansData)) {
					t.Errorf("expected humans.txt size %v, got %v", len(humansData), e.Size)
				}
				if e.ModTime.IsZero() {
					t.Errorf("got zero mod time for humans.txt")
				}
				ct := "text/plain; charset=utf-8"
				if e.ContentType != ct {
					t.Errorf("expected content type %q, got %q", ct, e.ContentType)
				}
				break
			}
		}
		if !found {
			t.Fatal("no humans.txt in new manifest")
		}

		checkFile(t, client, newManifestHash, "humans.txt", humansData)
	})

	// upload a new file and use its manifest to add it the original manifest,
	// but ensure that the file will be in the nested manifest of the original one.
	t.Run("add nested", func(t *testing.T) {
		robotsData := []byte(`{"disallow": "/"}`)
		robotsDataFilename := filepath.Join(tmp, "robots.json")
		err = ioutil.WriteFile(robotsDataFilename, robotsData, 0666)
		if err != nil {
			t.Fatal(err)
		}

		robotsManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"up",
			robotsDataFilename,
		)

		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"add",
			origManifestHash,
			"robots.json",
			robotsManifestHash,
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
	loop:
		for _, e := range newManifest.Entries {
			if e.Path == "robots." {
				nestedManifest := downloadManifest(t, client, e.Hash, encrypt)
				for _, e := range nestedManifest.Entries {
					if e.Path == "json" {
						found = true
						if e.Size != int64(len(robotsData)) {
							t.Errorf("expected robots.json size %v, got %v", len(robotsData), e.Size)
						}
						if e.ModTime.IsZero() {
							t.Errorf("got zero mod time for robots.json")
						}
						ct := "application/json"
						if e.ContentType != ct {
							t.Errorf("expected content type %q, got %q", ct, e.ContentType)
						}
						break loop
					}
				}
			}
		}
		if !found {
			t.Fatal("no robots.json in new manifest")
		}

		checkFile(t, client, newManifestHash, "robots.json", robotsData)
	})

	// upload a new file and use its manifest to change the file it the original manifest.
	t.Run("update", func(t *testing.T) {
		indexData := []byte("<h1>Ethereum Swarm</h1>")
		indexDataFilename := filepath.Join(tmp, "index.html")
		err = ioutil.WriteFile(indexDataFilename, indexData, 0666)
		if err != nil {
			t.Fatal(err)
		}

		indexManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"up",
			indexDataFilename,
		)

		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"update",
			origManifestHash,
			"index.html",
			indexManifestHash,
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
		for _, e := range newManifest.Entries {
			if e.Path == "index.html" {
				found = true
				if e.Size != int64(len(indexData)) {
					t.Errorf("expected index.html size %v, got %v", len(indexData), e.Size)
				}
				if e.ModTime.IsZero() {
					t.Errorf("got zero mod time for index.html")
				}
				ct := "text/html; charset=utf-8"
				if e.ContentType != ct {
					t.Errorf("expected content type %q, got %q", ct, e.ContentType)
				}
				break
			}
		}
		if !found {
			t.Fatal("no index.html in new manifest")
		}

		checkFile(t, client, newManifestHash, "index.html", indexData)

		// check default entry change
		checkFile(t, client, newManifestHash, "", indexData)
	})

	// upload a new file and use its manifest to change the file it the original manifest,
	// but ensure that the file is in the nested manifest of the original one.
	t.Run("update nested", func(t *testing.T) {
		robotsData := []byte(`<string>Only humans allowed!!!</strong>`)
		robotsDataFilename := filepath.Join(tmp, "robots.html")
		err = ioutil.WriteFile(robotsDataFilename, robotsData, 0666)
		if err != nil {
			t.Fatal(err)
		}

		humansManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"up",
			robotsDataFilename,
		)

		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"update",
			origManifestHash,
			"robots.html",
			humansManifestHash,
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
	loop:
		for _, e := range newManifest.Entries {
			if e.Path == "robots." {
				nestedManifest := downloadManifest(t, client, e.Hash, encrypt)
				for _, e := range nestedManifest.Entries {
					if e.Path == "html" {
						found = true
						if e.Size != int64(len(robotsData)) {
							t.Errorf("expected robots.html size %v, got %v", len(robotsData), e.Size)
						}
						if e.ModTime.IsZero() {
							t.Errorf("got zero mod time for robots.html")
						}
						ct := "text/html; charset=utf-8"
						if e.ContentType != ct {
							t.Errorf("expected content type %q, got %q", ct, e.ContentType)
						}
						break loop
					}
				}
			}
		}
		if !found {
			t.Fatal("no robots.html in new manifest")
		}

		checkFile(t, client, newManifestHash, "robots.html", robotsData)
	})

	// remove a file from the manifest.
	t.Run("remove", func(t *testing.T) {
		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"remove",
			origManifestHash,
			"mutants.txt",
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
		for _, e := range newManifest.Entries {
			if e.Path == "mutants.txt" {
				found = true
				break
			}
		}
		if found {
			t.Fatal("mutants.txt is not removed")
		}
	})

	// remove a file from the manifest, but ensure that the file is in
	// the nested manifest of the original one.
	t.Run("remove nested", func(t *testing.T) {
		newManifestHash := runSwarmExpectHash(t,
			"--bzzapi",
			cluster.Nodes[0].URL,
			"manifest",
			"remove",
			origManifestHash,
			"robots.html",
		)

		checkHashLength(t, newManifestHash, encrypt)

		newManifest := downloadManifest(t, client, newManifestHash, encrypt)

		var found bool
	loop:
		for _, e := range newManifest.Entries {
			if e.Path == "robots." {
				nestedManifest := downloadManifest(t, client, e.Hash, encrypt)
				for _, e := range nestedManifest.Entries {
					if e.Path == "html" {
						found = true
						break loop
					}
				}
			}
		}
		if found {
			t.Fatal("robots.html in not removed")
		}
	})
}

// TestNestedDefaultEntryUpdate tests if the default entry is updated
// if the file in nested manifest used for it is also updated.
func TestNestedDefaultEntryUpdate(t *testing.T) {
	testNestedDefaultEntryUpdate(t, false)
}

// TestNestedDefaultEntryUpdateEncrypted tests if the default entry
// of encrypted upload is updated if the file in nested manifest
// used for it is also updated.
func TestNestedDefaultEntryUpdateEncrypted(t *testing.T) {
	testNestedDefaultEntryUpdate(t, true)
}

func testNestedDefaultEntryUpdate(t *testing.T, encrypt bool) {
	t.Parallel()
	cluster := newTestCluster(t, 1)
	defer cluster.Shutdown()

	tmp, err := ioutil.TempDir("", "swarm-manifest-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	origDir := filepath.Join(tmp, "orig")
	if err := os.Mkdir(origDir, 0777); err != nil {
		t.Fatal(err)
	}

	indexData := []byte("<h1>Test</h1>")
	indexDataFilename := filepath.Join(origDir, "index.html")
	err = ioutil.WriteFile(indexDataFilename, indexData, 0666)
	if err != nil {
		t.Fatal(err)
	}
	// Add another file with common prefix as the default entry to test updates of
	// default entry with nested manifests.
	err = ioutil.WriteFile(filepath.Join(origDir, "index.txt"), []byte("Test"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	args := []string{
		"--bzzapi",
		cluster.Nodes[0].URL,
		"--recursive",
		"--defaultpath",
		indexDataFilename,
		"up",
		origDir,
	}
	if encrypt {
		args = append(args, "--encrypt")
	}

	origManifestHash := runSwarmExpectHash(t, args...)

	checkHashLength(t, origManifestHash, encrypt)

	client := swarm.NewClient(cluster.Nodes[0].URL)

	newIndexData := []byte("<h1>Ethereum Swarm</h1>")
	newIndexDataFilename := filepath.Join(tmp, "index.html")
	err = ioutil.WriteFile(newIndexDataFilename, newIndexData, 0666)
	if err != nil {
		t.Fatal(err)
	}

	newIndexManifestHash := runSwarmExpectHash(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"up",
		newIndexDataFilename,
	)

	newManifestHash := runSwarmExpectHash(t,
		"--bzzapi",
		cluster.Nodes[0].URL,
		"manifest",
		"update",
		origManifestHash,
		"index.html",
		newIndexManifestHash,
	)

	checkHashLength(t, newManifestHash, encrypt)

	newManifest := downloadManifest(t, client, newManifestHash, encrypt)

	var found bool
	for _, e := range newManifest.Entries {
		if e.Path == "index." {
			found = true
			newManifest = downloadManifest(t, client, e.Hash, encrypt)
			break
		}
	}
	if !found {
		t.Fatal("no index. path in new manifest")
	}

	found = false
	for _, e := range newManifest.Entries {
		if e.Path == "html" {
			found = true
			if e.Size != int64(len(newIndexData)) {
				t.Errorf("expected index.html size %v, got %v", len(newIndexData), e.Size)
			}
			if e.ModTime.IsZero() {
				t.Errorf("got zero mod time for index.html")
			}
			ct := "text/html; charset=utf-8"
			if e.ContentType != ct {
				t.Errorf("expected content type %q, got %q", ct, e.ContentType)
			}
			break
		}
	}
	if !found {
		t.Fatal("no html in new manifest")
	}

	checkFile(t, client, newManifestHash, "index.html", newIndexData)

	// check default entry change
	checkFile(t, client, newManifestHash, "", newIndexData)
}

func runSwarmExpectHash(t *testing.T, args ...string) (hash string) {
	t.Helper()
	hashRegexp := `[a-f\d]{64,128}`
	up := runSwarm(t, args...)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()

	if len(matches) < 1 {
		t.Fatal("no matches found")
	}
	return matches[0]
}

func checkHashLength(t *testing.T, hash string, encrypted bool) {
	t.Helper()
	l := len(hash)
	if encrypted && l != 128 {
		t.Errorf("expected hash length 128, got %v", l)
	}
	if !encrypted && l != 64 {
		t.Errorf("expected hash length 64, got %v", l)
	}
}

func downloadManifest(t *testing.T, client *swarm.Client, hash string, encrypted bool) (manifest *api.Manifest) {
	t.Helper()
	m, isEncrypted, err := client.DownloadManifest(hash)
	if err != nil {
		t.Fatal(err)
	}

	if encrypted != isEncrypted {
		t.Error("new manifest encryption flag is not correct")
	}
	return m
}

func checkFile(t *testing.T, client *swarm.Client, hash, path string, expected []byte) {
	t.Helper()
	f, err := client.Download(hash, path)
	if err != nil {
		t.Fatal(err)
	}

	got, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, expected) {
		t.Errorf("expected file content %q, got %q", expected, got)
	}
}
