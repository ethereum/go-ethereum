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
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

var testDownloadDir, _ = ioutil.TempDir(os.TempDir(), "bzz-test")

func testFileSystem(t *testing.T, f func(*FileSystem)) {
	testApi(t, func(api *Api) {
		f(NewFileSystem(api))
	})
}

func readPath(t *testing.T, parts ...string) string {
	file := filepath.Join(parts...)
	content, err := ioutil.ReadFile(file)

	if err != nil {
		t.Fatalf("unexpected error reading '%v': %v", file, err)
	}
	return string(content)
}

func TestApiDirUpload0(t *testing.T) {
	testFileSystem(t, func(fs *FileSystem) {
		api := fs.api
		bzzhash, err := fs.Upload(filepath.Join("testdata", "test0"), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		content := readPath(t, "testdata", "test0", "index.html")
		resp := testGet(t, api, bzzhash, "index.html")
		exp := expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)

		content = readPath(t, "testdata", "test0", "index.css")
		resp = testGet(t, api, bzzhash, "index.css")
		exp = expResponse(content, "text/css", 0)
		checkResponse(t, resp, exp)

		key := storage.Key(common.Hex2Bytes(bzzhash))
		_, _, _, err = api.Get(key, "")
		if err == nil {
			t.Fatalf("expected error: %v", err)
		}

		downloadDir := filepath.Join(testDownloadDir, "test0")
		defer os.RemoveAll(downloadDir)
		err = fs.Download(bzzhash, downloadDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		newbzzhash, err := fs.Upload(downloadDir, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if bzzhash != newbzzhash {
			t.Fatalf("download %v reuploaded has incorrect hash, expected %v, got %v", downloadDir, bzzhash, newbzzhash)
		}
	})
}

func TestApiDirUploadModify(t *testing.T) {
	testFileSystem(t, func(fs *FileSystem) {
		api := fs.api
		bzzhash, err := fs.Upload(filepath.Join("testdata", "test0"), "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		key := storage.Key(common.Hex2Bytes(bzzhash))
		key, err = api.Modify(key, "index.html", "", "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		index, err := ioutil.ReadFile(filepath.Join("testdata", "test0", "index.html"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		wg := &sync.WaitGroup{}
		hash, err := api.Store(bytes.NewReader(index), int64(len(index)), wg)
		wg.Wait()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		key, err = api.Modify(key, "index2.html", hash.Hex(), "text/html; charset=utf-8")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		key, err = api.Modify(key, "img/logo.png", hash.Hex(), "text/html; charset=utf-8")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		bzzhash = key.String()

		content := readPath(t, "testdata", "test0", "index.html")
		resp := testGet(t, api, bzzhash, "index2.html")
		exp := expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)

		resp = testGet(t, api, bzzhash, "img/logo.png")
		exp = expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)

		content = readPath(t, "testdata", "test0", "index.css")
		resp = testGet(t, api, bzzhash, "index.css")
		exp = expResponse(content, "text/css", 0)
		checkResponse(t, resp, exp)

		_, _, _, err = api.Get(key, "")
		if err == nil {
			t.Errorf("expected error: %v", err)
		}
	})
}

func TestApiDirUploadWithRootFile(t *testing.T) {
	testFileSystem(t, func(fs *FileSystem) {
		api := fs.api
		bzzhash, err := fs.Upload(filepath.Join("testdata", "test0"), "index.html")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		content := readPath(t, "testdata", "test0", "index.html")
		resp := testGet(t, api, bzzhash, "")
		exp := expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)
	})
}

func TestApiFileUpload(t *testing.T) {
	testFileSystem(t, func(fs *FileSystem) {
		api := fs.api
		bzzhash, err := fs.Upload(filepath.Join("testdata", "test0", "index.html"), "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		content := readPath(t, "testdata", "test0", "index.html")
		resp := testGet(t, api, bzzhash, "index.html")
		exp := expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)
	})
}

func TestApiFileUploadWithRootFile(t *testing.T) {
	testFileSystem(t, func(fs *FileSystem) {
		api := fs.api
		bzzhash, err := fs.Upload(filepath.Join("testdata", "test0", "index.html"), "index.html")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}

		content := readPath(t, "testdata", "test0", "index.html")
		resp := testGet(t, api, bzzhash, "")
		exp := expResponse(content, "text/html; charset=utf-8", 0)
		checkResponse(t, resp, exp)
	})
}
