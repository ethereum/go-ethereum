// Copyright 2017 The go-ethereum Authors
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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	swarmapi "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	"github.com/mattn/go-colorable"
)

func init() {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

func TestSwarmUp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	initCluster(t)

	cases := []struct {
		name string
		f    func(t *testing.T)
	}{
		{"NoEncryption", testNoEncryption},
		{"Encrypted", testEncrypted},
		{"RecursiveNoEncryption", testRecursiveNoEncryption},
		{"RecursiveEncrypted", testRecursiveEncrypted},
		{"DefaultPathAll", testDefaultPathAll},
	}

	for _, tc := range cases {
		t.Run(tc.name, tc.f)
	}
}

// testNoEncryption tests that running 'swarm up' makes the resulting file
// available from all nodes via the HTTP API
func testNoEncryption(t *testing.T) {
	testDefault(false, t)
}

// testEncrypted tests that running 'swarm up --encrypted' makes the resulting file
// available from all nodes via the HTTP API
func testEncrypted(t *testing.T) {
	testDefault(true, t)
}

func testRecursiveNoEncryption(t *testing.T) {
	testRecursive(false, t)
}

func testRecursiveEncrypted(t *testing.T) {
	testRecursive(true, t)
}

func testDefault(toEncrypt bool, t *testing.T) {
	tmpFileName := testutil.TempFileWithContent(t, data)
	defer os.Remove(tmpFileName)

	// write data to file
	hashRegexp := `[a-f\d]{64}`
	flags := []string{
		"--bzzapi", cluster.Nodes[0].URL,
		"up",
		tmpFileName}
	if toEncrypt {
		hashRegexp = `[a-f\d]{128}`
		flags = []string{
			"--bzzapi", cluster.Nodes[0].URL,
			"up",
			"--encrypt",
			tmpFileName}
	}
	// upload the file with 'swarm up' and expect a hash
	log.Info(fmt.Sprintf("uploading file with 'swarm up'"))
	up := runSwarm(t, flags...)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()
	hash := matches[0]
	log.Info("file uploaded", "hash", hash)

	// get the file from the HTTP API of each node
	for _, node := range cluster.Nodes {
		log.Info("getting file from node", "node", node.Name)

		res, err := http.Get(node.URL + "/bzz:/" + hash)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		reply, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("expected HTTP status 200, got %s", res.Status)
		}
		if string(reply) != data {
			t.Fatalf("expected HTTP body %q, got %q", data, reply)
		}
		log.Debug("verifying uploaded file using `swarm down`")
		//try to get the content with `swarm down`
		tmpDownload, err := ioutil.TempDir("", "swarm-test")
		tmpDownload = path.Join(tmpDownload, "tmpfile.tmp")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDownload)

		bzzLocator := "bzz:/" + hash
		flags = []string{
			"--bzzapi", cluster.Nodes[0].URL,
			"down",
			bzzLocator,
			tmpDownload,
		}

		down := runSwarm(t, flags...)
		down.ExpectExit()

		fi, err := os.Stat(tmpDownload)
		if err != nil {
			t.Fatalf("could not stat path: %v", err)
		}

		switch mode := fi.Mode(); {
		case mode.IsRegular():
			downloadedBytes, err := ioutil.ReadFile(tmpDownload)
			if err != nil {
				t.Fatalf("had an error reading the downloaded file: %v", err)
			}
			if !bytes.Equal(downloadedBytes, bytes.NewBufferString(data).Bytes()) {
				t.Fatalf("retrieved data and posted data not equal!")
			}

		default:
			t.Fatalf("expected to download regular file, got %s", fi.Mode())
		}
	}

	timeout := time.Duration(2 * time.Second)
	httpClient := http.Client{
		Timeout: timeout,
	}

	// try to squeeze a timeout by getting an non-existent hash from each node
	for _, node := range cluster.Nodes {
		_, err := httpClient.Get(node.URL + "/bzz:/1023e8bae0f70be7d7b5f74343088ba408a218254391490c85ae16278e230340")
		// we're speeding up the timeout here since netstore has a 60 seconds timeout on a request
		if err != nil && !strings.Contains(err.Error(), "Client.Timeout exceeded while awaiting headers") {
			t.Fatal(err)
		}
		// this is disabled since it takes 60s due to netstore timeout
		// if res.StatusCode != 404 {
		// 	t.Fatalf("expected HTTP status 404, got %s", res.Status)
		// }
	}
}

func testRecursive(toEncrypt bool, t *testing.T) {
	tmpUploadDir, err := ioutil.TempDir("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpUploadDir)
	// create tmp files
	for _, path := range []string{"tmp1", "tmp2"} {
		if err := ioutil.WriteFile(filepath.Join(tmpUploadDir, path), bytes.NewBufferString(data).Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	hashRegexp := `[a-f\d]{64}`
	flags := []string{
		"--bzzapi", cluster.Nodes[0].URL,
		"--recursive",
		"up",
		tmpUploadDir}
	if toEncrypt {
		hashRegexp = `[a-f\d]{128}`
		flags = []string{
			"--bzzapi", cluster.Nodes[0].URL,
			"--recursive",
			"up",
			"--encrypt",
			tmpUploadDir}
	}
	// upload the file with 'swarm up' and expect a hash
	log.Info(fmt.Sprintf("uploading file with 'swarm up'"))
	up := runSwarm(t, flags...)
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()
	hash := matches[0]
	log.Info("dir uploaded", "hash", hash)

	// get the file from the HTTP API of each node
	for _, node := range cluster.Nodes {
		log.Info("getting file from node", "node", node.Name)
		//try to get the content with `swarm down`
		tmpDownload, err := ioutil.TempDir("", "swarm-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDownload)
		bzzLocator := "bzz:/" + hash
		flagss := []string{
			"--bzzapi", cluster.Nodes[0].URL,
			"down",
			"--recursive",
			bzzLocator,
			tmpDownload,
		}

		fmt.Println("downloading from swarm with recursive")
		down := runSwarm(t, flagss...)
		down.ExpectExit()

		files, err := ioutil.ReadDir(tmpDownload)
		for _, v := range files {
			fi, err := os.Stat(path.Join(tmpDownload, v.Name()))
			if err != nil {
				t.Fatalf("got an error: %v", err)
			}

			switch mode := fi.Mode(); {
			case mode.IsRegular():
				if file, err := swarmapi.Open(path.Join(tmpDownload, v.Name())); err != nil {
					t.Fatalf("encountered an error opening the file returned from the CLI: %v", err)
				} else {
					ff := make([]byte, len(data))
					io.ReadFull(file, ff)
					buf := bytes.NewBufferString(data)

					if !bytes.Equal(ff, buf.Bytes()) {
						t.Fatalf("retrieved data and posted data not equal!")
					}
				}
			default:
				t.Fatalf("this shouldnt happen")
			}
		}
		if err != nil {
			t.Fatalf("could not list files at: %v", files)
		}
	}
}

// testDefaultPathAll tests swarm recursive upload with relative and absolute
// default paths and with encryption.
func testDefaultPathAll(t *testing.T) {
	testDefaultPath(false, false, t)
	testDefaultPath(false, true, t)
	testDefaultPath(true, false, t)
	testDefaultPath(true, true, t)
}

func testDefaultPath(toEncrypt bool, absDefaultPath bool, t *testing.T) {
	tmp, err := ioutil.TempDir("", "swarm-defaultpath-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	err = ioutil.WriteFile(filepath.Join(tmp, "index.html"), []byte("<h1>Test</h1>"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(tmp, "robots.txt"), []byte("Disallow: /"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	defaultPath := "index.html"
	if absDefaultPath {
		defaultPath = filepath.Join(tmp, defaultPath)
	}

	args := []string{
		"--bzzapi",
		cluster.Nodes[0].URL,
		"--recursive",
		"--defaultpath",
		defaultPath,
		"up",
		tmp,
	}
	if toEncrypt {
		args = append(args, "--encrypt")
	}

	up := runSwarm(t, args...)
	hashRegexp := `[a-f\d]{64,128}`
	_, matches := up.ExpectRegexp(hashRegexp)
	up.ExpectExit()
	hash := matches[0]

	client := swarmapi.NewClient(cluster.Nodes[0].URL)

	m, isEncrypted, err := client.DownloadManifest(hash)
	if err != nil {
		t.Fatal(err)
	}

	if toEncrypt != isEncrypted {
		t.Error("downloaded manifest is not encrypted")
	}

	var found bool
	var entriesCount int
	for _, e := range m.Entries {
		entriesCount++
		if e.Path == "" {
			found = true
		}
	}

	if !found {
		t.Error("manifest default entry was not found")
	}

	if entriesCount != 3 {
		t.Errorf("manifest contains %v entries, expected %v", entriesCount, 3)
	}
}
