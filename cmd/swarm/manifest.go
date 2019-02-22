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

// Command  MANIFEST update
package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"

	"github.com/ubiq/go-ubiq/cmd/utils"
	"github.com/ubiq/go-ubiq/swarm/api"
	swarm "github.com/ubiq/go-ubiq/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

func add(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		utils.Fatalf("Need atleast three arguments <MHASH> <path> <HASH> [<content-type>]")
	}

	var (
		mhash = args[0]
		path  = args[1]
		hash  = args[2]

		ctype        string
		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		mroot        api.Manifest
	)

	if len(args) > 3 {
		ctype = args[3]
	} else {
		ctype = mime.TypeByExtension(filepath.Ext(path))
	}

	newManifest := addEntryToManifest(ctx, mhash, path, hash, ctype)
	fmt.Println(newManifest)

	if !wantManifest {
		// Print the manifest. This is the only output to stdout.
		mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
		fmt.Println(string(mrootJSON))
		return
	}
}

func update(ctx *cli.Context) {

	args := ctx.Args()
	if len(args) < 3 {
		utils.Fatalf("Need atleast three arguments <MHASH> <path> <HASH>")
	}

	var (
		mhash = args[0]
		path  = args[1]
		hash  = args[2]

		ctype        string
		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		mroot        api.Manifest
	)
	if len(args) > 3 {
		ctype = args[3]
	} else {
		ctype = mime.TypeByExtension(filepath.Ext(path))
	}

	newManifest := updateEntryInManifest(ctx, mhash, path, hash, ctype)
	fmt.Println(newManifest)

	if !wantManifest {
		// Print the manifest. This is the only output to stdout.
		mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
		fmt.Println(string(mrootJSON))
		return
	}
}

func remove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		utils.Fatalf("Need atleast two arguments <MHASH> <path>")
	}

	var (
		mhash = args[0]
		path  = args[1]

		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		mroot        api.Manifest
	)

	newManifest := removeEntryFromManifest(ctx, mhash, path)
	fmt.Println(newManifest)

	if !wantManifest {
		// Print the manifest. This is the only output to stdout.
		mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
		fmt.Println(string(mrootJSON))
		return
	}
}

func addEntryToManifest(ctx *cli.Context, mhash, path, hash, ctype string) string {

	var (
		bzzapi           = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client           = swarm.NewClient(bzzapi)
		longestPathEntry = api.ManifestEntry{}
	)

	mroot, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	//TODO: check if the "hash" to add is valid and present in swarm
	_, err = client.DownloadManifest(hash)
	if err != nil {
		utils.Fatalf("Hash to add is not present: %v", err)
	}

	// See if we path is in this Manifest or do we have to dig deeper
	for _, entry := range mroot.Entries {
		if path == entry.Path {
			utils.Fatalf("Path %s already present, not adding anything", path)
		} else {
			if entry.ContentType == "application/bzz-manifest+json" {
				prfxlen := strings.HasPrefix(path, entry.Path)
				if prfxlen && len(path) > len(longestPathEntry.Path) {
					longestPathEntry = entry
				}
			}
		}
	}

	if longestPathEntry.Path != "" {
		// Load the child Manifest add the entry there
		newPath := path[len(longestPathEntry.Path):]
		newHash := addEntryToManifest(ctx, longestPathEntry.Hash, newPath, hash, ctype)

		// Replace the hash for parent Manifests
		newMRoot := &api.Manifest{}
		for _, entry := range mroot.Entries {
			if longestPathEntry.Path == entry.Path {
				entry.Hash = newHash
			}
			newMRoot.Entries = append(newMRoot.Entries, entry)
		}
		mroot = newMRoot
	} else {
		// Add the entry in the leaf Manifest
		newEntry := api.ManifestEntry{
			Hash:        hash,
			Path:        path,
			ContentType: ctype,
		}
		mroot.Entries = append(mroot.Entries, newEntry)
	}

	newManifestHash, err := client.UploadManifest(mroot)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash

}

func updateEntryInManifest(ctx *cli.Context, mhash, path, hash, ctype string) string {

	var (
		bzzapi           = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client           = swarm.NewClient(bzzapi)
		newEntry         = api.ManifestEntry{}
		longestPathEntry = api.ManifestEntry{}
	)

	mroot, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	//TODO: check if the "hash" with which to update is valid and present in swarm

	// See if we path is in this Manifest or do we have to dig deeper
	for _, entry := range mroot.Entries {
		if path == entry.Path {
			newEntry = entry
		} else {
			if entry.ContentType == "application/bzz-manifest+json" {
				prfxlen := strings.HasPrefix(path, entry.Path)
				if prfxlen && len(path) > len(longestPathEntry.Path) {
					longestPathEntry = entry
				}
			}
		}
	}

	if longestPathEntry.Path == "" && newEntry.Path == "" {
		utils.Fatalf("Path %s not present in the Manifest, not setting anything", path)
	}

	if longestPathEntry.Path != "" {
		// Load the child Manifest add the entry there
		newPath := path[len(longestPathEntry.Path):]
		newHash := updateEntryInManifest(ctx, longestPathEntry.Hash, newPath, hash, ctype)

		// Replace the hash for parent Manifests
		newMRoot := &api.Manifest{}
		for _, entry := range mroot.Entries {
			if longestPathEntry.Path == entry.Path {
				entry.Hash = newHash
			}
			newMRoot.Entries = append(newMRoot.Entries, entry)

		}
		mroot = newMRoot
	}

	if newEntry.Path != "" {
		// Replace the hash for leaf Manifest
		newMRoot := &api.Manifest{}
		for _, entry := range mroot.Entries {
			if newEntry.Path == entry.Path {
				myEntry := api.ManifestEntry{
					Hash:        hash,
					Path:        entry.Path,
					ContentType: ctype,
				}
				newMRoot.Entries = append(newMRoot.Entries, myEntry)
			} else {
				newMRoot.Entries = append(newMRoot.Entries, entry)
			}
		}
		mroot = newMRoot
	}

	newManifestHash, err := client.UploadManifest(mroot)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash
}

func removeEntryFromManifest(ctx *cli.Context, mhash, path string) string {

	var (
		bzzapi           = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client           = swarm.NewClient(bzzapi)
		entryToRemove    = api.ManifestEntry{}
		longestPathEntry = api.ManifestEntry{}
	)

	mroot, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	// See if we path is in this Manifest or do we have to dig deeper
	for _, entry := range mroot.Entries {
		if path == entry.Path {
			entryToRemove = entry
		} else {
			if entry.ContentType == "application/bzz-manifest+json" {
				prfxlen := strings.HasPrefix(path, entry.Path)
				if prfxlen && len(path) > len(longestPathEntry.Path) {
					longestPathEntry = entry
				}
			}
		}
	}

	if longestPathEntry.Path == "" && entryToRemove.Path == "" {
		utils.Fatalf("Path %s not present in the Manifest, not removing anything", path)
	}

	if longestPathEntry.Path != "" {
		// Load the child Manifest remove the entry there
		newPath := path[len(longestPathEntry.Path):]
		newHash := removeEntryFromManifest(ctx, longestPathEntry.Hash, newPath)

		// Replace the hash for parent Manifests
		newMRoot := &api.Manifest{}
		for _, entry := range mroot.Entries {
			if longestPathEntry.Path == entry.Path {
				entry.Hash = newHash
			}
			newMRoot.Entries = append(newMRoot.Entries, entry)
		}
		mroot = newMRoot
	}

	if entryToRemove.Path != "" {
		// remove the entry in this Manifest
		newMRoot := &api.Manifest{}
		for _, entry := range mroot.Entries {
			if entryToRemove.Path != entry.Path {
				newMRoot.Entries = append(newMRoot.Entries, entry)
			}
		}
		mroot = newMRoot
	}

	newManifestHash, err := client.UploadManifest(mroot)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash
}
