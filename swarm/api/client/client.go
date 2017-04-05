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

func (c *Client) UploadFile(file string, fi os.FileInfo, mimetype_hint string) (ManifestEntry, error) {
	var mimetype string
	hash, err := c.uploadFileContent(file, fi)
	if mimetype_hint != "" {
		mimetype = mimetype_hint
		log.Info("Mime type set by override", "mime", mimetype)
	} else {
		ext := filepath.Ext(file)
		log.Info("Ext", "ext", ext, "file", file)
		if ext != "" {
			mimetype = mime.TypeByExtension(filepath.Ext(fi.Name()))
			log.Info("Mime type set by fileextension", "mime", mimetype, "ext", filepath.Ext(file))
		} else {
			f, err := os.Open(file)
			if err == nil {
				first512 := make([]byte, 512)
				fread, _ := f.ReadAt(first512, 0)
				if fread > 0 {
					mimetype = http.DetectContentType(first512[:fread])
					log.Info("Mime type set by autodetection", "mime", mimetype)
				}
			}
			f.Close()
		}

	}
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

// ManifestFileList downloads the manifest with the given hash and generates a
// list of files and directory prefixes which have the specified prefix.
//
// For example, if the manifest represents the following directory structure:
//
// file1.txt
// file2.txt
// dir1/file3.txt
// dir1/dir2/file4.txt
//
// Then:
//
// - a prefix of ""      would return [dir1/, file1.txt, file2.txt]
// - a prefix of "file"  would return [file1.txt, file2.txt]
// - a prefix of "dir1/" would return [dir1/dir2/, dir1/file3.txt]
func (c *Client) ManifestFileList(hash, prefix string) (entries []ManifestEntry, err error) {
	manifest, err := c.DownloadManifest(hash)
	if err != nil {
		return nil, err
	}

	// handleFile handles a manifest entry which is a direct reference to a
	// file (i.e. it is not a swarm manifest)
	handleFile := func(entry ManifestEntry) {
		// ignore the file if it doesn't have the specified prefix
		if !strings.HasPrefix(entry.Path, prefix) {
			return
		}
		// if the path after the prefix contains a directory separator,
		// add a directory prefix to the entries, otherwise add the
		// file
		suffix := strings.TrimPrefix(entry.Path, prefix)
		if sepIndex := strings.Index(suffix, "/"); sepIndex > -1 {
			entries = append(entries, ManifestEntry{
				Path:        prefix + suffix[:sepIndex+1],
				ContentType: "DIR",
			})
		} else {
			if entry.Path == "" {
				entry.Path = "/"
			}
			entries = append(entries, entry)
		}
	}

	// handleManifest handles a manifest entry which is a reference to
	// another swarm manifest.
	handleManifest := func(entry ManifestEntry) error {
		// if the manifest's path is a prefix of the specified prefix
		// then just recurse into the manifest by stripping its path
		// from the prefix
		if strings.HasPrefix(prefix, entry.Path) {
			subPrefix := strings.TrimPrefix(prefix, entry.Path)
			subEntries, err := c.ManifestFileList(entry.Hash, subPrefix)
			if err != nil {
				return err
			}
			// prefix the manifest's path to the sub entries and
			// add them to the returned entries
			for i, subEntry := range subEntries {
				subEntry.Path = entry.Path + subEntry.Path
				subEntries[i] = subEntry
			}
			entries = append(entries, subEntries...)
			return nil
		}

		// if the manifest's path has the specified prefix, then if the
		// path after the prefix contains a directory separator, add a
		// directory prefix to the entries, otherwise recurse into the
		// manifest
		if strings.HasPrefix(entry.Path, prefix) {
			suffix := strings.TrimPrefix(entry.Path, prefix)
			sepIndex := strings.Index(suffix, "/")
			if sepIndex > -1 {
				entries = append(entries, ManifestEntry{
					Path:        prefix + suffix[:sepIndex+1],
					ContentType: "DIR",
				})
				return nil
			}
			subEntries, err := c.ManifestFileList(entry.Hash, "")
			if err != nil {
				return err
			}
			// prefix the manifest's path to the sub entries and
			// add them to the returned entries
			for i, subEntry := range subEntries {
				subEntry.Path = entry.Path + subEntry.Path
				subEntries[i] = subEntry
			}
			entries = append(entries, subEntries...)
			return nil
		}
		return nil
	}

	for _, entry := range manifest.Entries {
		if entry.ContentType == "application/bzz-manifest+json" {
			if err := handleManifest(entry); err != nil {
				return nil, err
			}
		} else {
			handleFile(entry)
		}
	}

	return
}
