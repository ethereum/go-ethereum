// Copyright 2016 The go-ethereum Authors
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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/log"
)

var (
	DefaultGateway = "http://localhost:8500"
	DefaultClient  = NewClient(DefaultGateway)
)

// Manifest represents a swarm manifest.
type Manifest struct {
	Entries []ManifestEntry `json:"entries,omitempty"`
}

// ManifestEntry represents an entry in a swarm manifest.
type ManifestEntry struct {
	Hash        string `json:"hash,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Path        string `json:"path,omitempty"`
}

func NewClient(gateway string) *Client {
	return &Client{
		Gateway: gateway,
	}
}

// Client wraps interaction with a swarm HTTP gateway.
type Client struct {
	Gateway string
}

func (c *Client) UploadDirectory(dir string, defaultPath string) (string, error) {
	mhash, err := c.postRaw("application/json", 2, ioutil.NopCloser(bytes.NewReader([]byte("{}"))))
	if err != nil {
		return "", fmt.Errorf("failed to upload empty manifest")
	}
	if len(defaultPath) > 0 {
		fi, err := os.Stat(defaultPath)
		if err != nil {
			return "", err
		}
		mhash, err = c.uploadToManifest(mhash, "", defaultPath, fi)
		if err != nil {
			return "", err
		}
	}
	prefix := filepath.ToSlash(filepath.Clean(dir)) + "/"
	err = filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if !strings.HasPrefix(path, dir) {
			return fmt.Errorf("path %s outside directory %s", path, dir)
		}
		uripath := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), prefix)
		mhash, err = c.uploadToManifest(mhash, uripath, path, fi)
		return err
	})
	return mhash, err
}

func (c *Client) UploadFile(file string, fi os.FileInfo) (ManifestEntry, error) {
	hash, err := c.uploadFileContent(file, fi)
	m := ManifestEntry{
		Hash:        hash,
		ContentType: mime.TypeByExtension(filepath.Ext(fi.Name())),
	}
	return m, err
}

func (c *Client) uploadFileContent(file string, fi os.FileInfo) (string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	log.Info("Uploading swarm content", "file", file, "bytes", fi.Size())
	return c.postRaw("application/octet-stream", fi.Size(), fd)
}

func (c *Client) UploadManifest(m Manifest) (string, error) {
	jsm, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	log.Info("Uploading swarm manifest")
	return c.postRaw("application/json", int64(len(jsm)), ioutil.NopCloser(bytes.NewReader(jsm)))
}

func (c *Client) uploadToManifest(mhash string, path string, fpath string, fi os.FileInfo) (string, error) {
	fd, err := os.Open(fpath)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	log.Info("Uploading swarm content and path", "file", fpath, "bytes", fi.Size(), "path", path)
	req, err := http.NewRequest("PUT", c.Gateway+"/bzz:/"+mhash+"/"+path, fd)
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", mime.TypeByExtension(filepath.Ext(fi.Name())))
	req.ContentLength = fi.Size()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	content, err := ioutil.ReadAll(resp.Body)
	return string(content), err
}

func (c *Client) postRaw(mimetype string, size int64, body io.ReadCloser) (string, error) {
	req, err := http.NewRequest("POST", c.Gateway+"/bzzr:/", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", mimetype)
	req.ContentLength = size
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	content, err := ioutil.ReadAll(resp.Body)
	return string(content), err
}

func (c *Client) DownloadManifest(mhash string) (Manifest, error) {

	mroot := Manifest{}
	req, err := http.NewRequest("GET", c.Gateway+"/bzzr:/"+mhash, nil)
	if err != nil {
		return mroot, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return mroot, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return mroot, fmt.Errorf("bad status: %s", resp.Status)

	}
	content, err := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(content, &mroot)
	if err != nil {
		return mroot, fmt.Errorf("Manifest %v is malformed: %v", mhash, err)
	}
	return mroot, err
}
