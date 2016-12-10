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

// Command bzzup uploads files to the swarm HTTP API.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

func upload(ctx *cli.Context) {
	args := ctx.Args()
	var (
		bzzapi       = ctx.GlobalString(SwarmApiFlag.Name)
		recursive    = ctx.GlobalBool(SwarmRecursiveUploadFlag.Name)
		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
	)
	if len(args) != 1 {
		log.Fatal("need filename as the first and only argument")
	}

	var (
		file   = args[0]
		client = &client{api: bzzapi}
		mroot  manifest
	)
	fi, err := os.Stat(file)
	if err != nil {
		log.Fatal(err)
	}
	if fi.IsDir() {
		if !recursive {
			log.Fatal("argument is a directory and recursive upload is disabled")
		}
		mroot, err = client.uploadDirectory(file)
	} else {
		mroot, err = client.uploadFile(file, fi)
		if wantManifest {
			// Wrap the raw file entry in a proper manifest so both hashes get printed.
			mroot = manifest{Entries: []manifest{mroot}}
		}
	}
	if err != nil {
		log.Fatalln("upload failed:", err)
	}
	if wantManifest {
		hash, err := client.uploadManifest(mroot)
		if err != nil {
			log.Fatalln("manifest upload failed:", err)
		}
		mroot.Hash = hash
	}

	// Print the manifest. This is the only output to stdout.
	mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
	fmt.Println(string(mrootJSON))
}

// client wraps interaction with the swarm HTTP gateway.
type client struct {
	api string
}

// manifest is the JSON representation of a swarm manifest.
type manifest struct {
	Hash        string     `json:"hash,omitempty"`
	ContentType string     `json:"contentType,omitempty"`
	Path        string     `json:"path,omitempty"`
	Entries     []manifest `json:"entries,omitempty"`
}

func (c *client) uploadFile(file string, fi os.FileInfo) (manifest, error) {
	hash, err := c.uploadFileContent(file, fi)
	m := manifest{
		Hash:        hash,
		ContentType: mime.TypeByExtension(filepath.Ext(fi.Name())),
	}
	return m, err
}

func (c *client) uploadDirectory(dir string) (manifest, error) {
	dirm := manifest{}
	prefix := filepath.ToSlash(filepath.Clean(dir)) + "/"
	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if !strings.HasPrefix(path, dir) {
			return fmt.Errorf("path %s outside directory %s", path, dir)
		}
		entry, err := c.uploadFile(path, fi)
		entry.Path = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), prefix)
		dirm.Entries = append(dirm.Entries, entry)
		return err
	})
	return dirm, err
}

func (c *client) uploadFileContent(file string, fi os.FileInfo) (string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	log.Printf("uploading file %s (%d bytes)", file, fi.Size())
	return c.postRaw("application/octet-stream", fi.Size(), fd)
}

func (c *client) uploadManifest(m manifest) (string, error) {
	jsm, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	log.Println("uploading manifest")
	return c.postRaw("application/json", int64(len(jsm)), ioutil.NopCloser(bytes.NewReader(jsm)))
}

func (c *client) postRaw(mimetype string, size int64, body io.ReadCloser) (string, error) {
	req, err := http.NewRequest("POST", c.api+"/bzzr:/", body)
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
