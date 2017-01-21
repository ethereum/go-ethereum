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

// Command  MANIFEST update
package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"mime"
	"path/filepath"
	"strings"
)

func add(ctx *cli.Context) {

	args := ctx.Args()
	if len(args) < 3 {
		log.Fatal("need atleast three arguments")
	}

	var (
		mhash  = args[0]
		path   = args[1]
		hash   = args[2]

	)

	updateManifest (ctx, mhash, "add", path, hash)
}

func update(ctx *cli.Context) {

	args := ctx.Args()
	if len(args) < 3 {
		log.Fatal("need atleast three arguments")
	}

	var (
		mhash  = args[0]
		path   = args[1]
		hash   = args[2]

	)

	updateManifest (ctx, mhash, "update", path, hash)
}

func remove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		log.Fatal("need atleast two arguments")
	}

	var (
		mhash  = args[0]
		path   = args[1]

	)

	updateManifest (ctx, mhash, "remove", path, "")
}


func updateManifest(ctx *cli.Context, mhash , subcmd, path, hash string) {

	var (
		bzzapi       = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		client = &client{api: bzzapi}
		mroot  manifest
	)

	/* TODO: check for proper hash
	if !common.IsHexAddress(mhash) {
		log.Fatal(mhash, " is not a valid hash")
	}

	if !common.IsHexAddress(hash) {
		log.Fatal(hash, " is not a valid hash")
	}
	*/

	mroot, err := client.downloadManifest(mhash)
	if err != nil {
		log.Fatalln("manifest download failed:", err)
	}

	switch subcmd {

	case "add":
		for _, entry := range mroot.Entries {
			if path == entry.Path {
				log.Fatal(path, "Already present, not adding anything")
			}
		}

		newEntry := manifestEntry{
			Path:        path,
			Hash:        hash,
			ContentType: mime.TypeByExtension(filepath.Ext(path)),
		}
		mroot.Entries = append(mroot.Entries, newEntry)
		break

	case "update":
		foundEntry := bool(false)
		newMRoot := manifest{}
		for _, entry := range mroot.Entries {
			if path == entry.Path {
				newEntry := manifestEntry{
					Path:        entry.Path,
					Hash:        hash,
					ContentType: entry.ContentType,
				}
				foundEntry = true
				newMRoot.Entries = append(newMRoot.Entries, newEntry)
			} else {
				newMRoot.Entries = append(newMRoot.Entries, entry)
			}
		}

		if !foundEntry {
			log.Fatal(path, " Path not present in the Manifest, not setting anything")
		}

		mroot = newMRoot
		break

	case "remove":

		foundEntry := bool(false)
		newMRoot := manifest{}
		for _, entry := range mroot.Entries {
			if path != entry.Path {
				newEntry := manifestEntry{
					Path:        entry.Path,
					Hash:        entry.Hash,
					ContentType: entry.ContentType,
				}
				newMRoot.Entries = append(newMRoot.Entries, newEntry)
			} else {
				foundEntry = true
			}
		}

		if !foundEntry {
			log.Fatal(path, "Path not present in the Manifest, not removing anything")
		}

		mroot = newMRoot
		break

	}

	if !wantManifest {
		// Print the manifest. This is the only output to stdout.
		mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
		fmt.Println(string(mrootJSON))
		return
	}

	newManifestHash, err := client.uploadManifest(mroot)
	if err != nil {
		log.Fatalln("manifest upload failed:", err)
	}
	fmt.Println(newManifestHash)

}
