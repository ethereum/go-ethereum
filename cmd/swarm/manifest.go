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
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

var manifestCommand = cli.Command{
	Name:               "manifest",
	CustomHelpTemplate: helpTemplate,
	Usage:              "perform operations on swarm manifests",
	ArgsUsage:          "COMMAND",
	Description:        "Updates a MANIFEST by adding/removing/updating the hash of a path.\nCOMMAND could be: add, update, remove",
	Subcommands: []cli.Command{
		{
			Action:             manifestAdd,
			CustomHelpTemplate: helpTemplate,
			Name:               "add",
			Usage:              "add a new path to the manifest",
			ArgsUsage:          "<MANIFEST> <path> <hash>",
			Description:        "Adds a new path to the manifest",
		},
		{
			Action:             manifestUpdate,
			CustomHelpTemplate: helpTemplate,
			Name:               "update",
			Usage:              "update the hash for an already existing path in the manifest",
			ArgsUsage:          "<MANIFEST> <path> <newhash>",
			Description:        "Update the hash for an already existing path in the manifest",
		},
		{
			Action:             manifestRemove,
			CustomHelpTemplate: helpTemplate,
			Name:               "remove",
			Usage:              "removes a path from the manifest",
			ArgsUsage:          "<MANIFEST> <path>",
			Description:        "Removes a path from the manifest",
		},
	},
}

// manifestAdd adds a new entry to the manifest at the given path.
// New entry hash, the last argument, must be the hash of a manifest
// with only one entry, which meta-data will be added to the original manifest.
// On success, this function will print new (updated) manifest's hash.
func manifestAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 {
		utils.Fatalf("Need exactly three arguments <MHASH> <path> <HASH>")
	}

	var (
		mhash = args[0]
		path  = args[1]
		hash  = args[2]
	)

	bzzapi := strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
	client := swarm.NewClient(bzzapi)

	m, _, err := client.DownloadManifest(hash)
	if err != nil {
		utils.Fatalf("Error downloading manifest to add: %v", err)
	}
	l := len(m.Entries)
	if l == 0 {
		utils.Fatalf("No entries in manifest %s", hash)
	} else if l > 1 {
		utils.Fatalf("Too many entries in manifest %s", hash)
	}

	newManifest := addEntryToManifest(client, mhash, path, m.Entries[0])
	fmt.Println(newManifest)
}

// manifestUpdate replaces an existing entry of the manifest at the given path.
// New entry hash, the last argument, must be the hash of a manifest
// with only one entry, which meta-data will be added to the original manifest.
// On success, this function will print hash of the updated manifest.
func manifestUpdate(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 {
		utils.Fatalf("Need exactly three arguments <MHASH> <path> <HASH>")
	}

	var (
		mhash = args[0]
		path  = args[1]
		hash  = args[2]
	)

	bzzapi := strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
	client := swarm.NewClient(bzzapi)

	m, _, err := client.DownloadManifest(hash)
	if err != nil {
		utils.Fatalf("Error downloading manifest to update: %v", err)
	}
	l := len(m.Entries)
	if l == 0 {
		utils.Fatalf("No entries in manifest %s", hash)
	} else if l > 1 {
		utils.Fatalf("Too many entries in manifest %s", hash)
	}

	newManifest, _, defaultEntryUpdated := updateEntryInManifest(client, mhash, path, m.Entries[0], true)
	if defaultEntryUpdated {
		// Print informational message to stderr
		// allowing the user to get the new manifest hash from stdout
		// without the need to parse the complete output.
		fmt.Fprintln(os.Stderr, "Manifest default entry is updated, too")
	}
	fmt.Println(newManifest)
}

// manifestRemove removes an existing entry of the manifest at the given path.
// On success, this function will print hash of the manifest which does not
// contain the path.
func manifestRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 2 {
		utils.Fatalf("Need exactly two arguments <MHASH> <path>")
	}

	var (
		mhash = args[0]
		path  = args[1]
	)

	bzzapi := strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
	client := swarm.NewClient(bzzapi)

	newManifest := removeEntryFromManifest(client, mhash, path)
	fmt.Println(newManifest)
}

func addEntryToManifest(client *swarm.Client, mhash, path string, entry api.ManifestEntry) string {
	var longestPathEntry = api.ManifestEntry{}

	mroot, isEncrypted, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	// See if we path is in this Manifest or do we have to dig deeper
	for _, e := range mroot.Entries {
		if path == e.Path {
			utils.Fatalf("Path %s already present, not adding anything", path)
		} else {
			if e.ContentType == api.ManifestType {
				prfxlen := strings.HasPrefix(path, e.Path)
				if prfxlen && len(path) > len(longestPathEntry.Path) {
					longestPathEntry = e
				}
			}
		}
	}

	if longestPathEntry.Path != "" {
		// Load the child Manifest add the entry there
		newPath := path[len(longestPathEntry.Path):]
		newHash := addEntryToManifest(client, longestPathEntry.Hash, newPath, entry)

		// Replace the hash for parent Manifests
		newMRoot := &api.Manifest{}
		for _, e := range mroot.Entries {
			if longestPathEntry.Path == e.Path {
				e.Hash = newHash
			}
			newMRoot.Entries = append(newMRoot.Entries, e)
		}
		mroot = newMRoot
	} else {
		// Add the entry in the leaf Manifest
		entry.Path = path
		mroot.Entries = append(mroot.Entries, entry)
	}

	newManifestHash, err := client.UploadManifest(mroot, isEncrypted)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash
}

// updateEntryInManifest updates an existing entry o path with a new one in the manifest with provided mhash
// finding the path recursively through all nested manifests. Argument isRoot is used for default
// entry update detection. If the updated entry has the same hash as the default entry, then the
// default entry in root manifest will be updated too.
// Returned values are the new manifest hash, hash of the entry that was replaced by the new entry and
// a a bool that is true if default entry is updated.
func updateEntryInManifest(client *swarm.Client, mhash, path string, entry api.ManifestEntry, isRoot bool) (newManifestHash, oldHash string, defaultEntryUpdated bool) {
	var (
		newEntry         = api.ManifestEntry{}
		longestPathEntry = api.ManifestEntry{}
	)

	mroot, isEncrypted, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	// See if we path is in this Manifest or do we have to dig deeper
	for _, e := range mroot.Entries {
		if path == e.Path {
			newEntry = e
			// keep the reference of the hash of the entry that should be replaced
			// for default entry detection
			oldHash = e.Hash
		} else {
			if e.ContentType == api.ManifestType {
				prfxlen := strings.HasPrefix(path, e.Path)
				if prfxlen && len(path) > len(longestPathEntry.Path) {
					longestPathEntry = e
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
		var newHash string
		newHash, oldHash, _ = updateEntryInManifest(client, longestPathEntry.Hash, newPath, entry, false)

		// Replace the hash for parent Manifests
		newMRoot := &api.Manifest{}
		for _, e := range mroot.Entries {
			if longestPathEntry.Path == e.Path {
				e.Hash = newHash
			}
			newMRoot.Entries = append(newMRoot.Entries, e)

		}
		mroot = newMRoot
	}

	// update the manifest if the new entry is found and
	// check if default entry should be updated
	if newEntry.Path != "" || isRoot {
		// Replace the hash for leaf Manifest
		newMRoot := &api.Manifest{}
		for _, e := range mroot.Entries {
			if newEntry.Path == e.Path {
				entry.Path = e.Path
				newMRoot.Entries = append(newMRoot.Entries, entry)
			} else if isRoot && e.Path == "" && e.Hash == oldHash {
				entry.Path = e.Path
				newMRoot.Entries = append(newMRoot.Entries, entry)
				defaultEntryUpdated = true
			} else {
				newMRoot.Entries = append(newMRoot.Entries, e)
			}
		}
		mroot = newMRoot
	}

	newManifestHash, err = client.UploadManifest(mroot, isEncrypted)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash, oldHash, defaultEntryUpdated
}

func removeEntryFromManifest(client *swarm.Client, mhash, path string) string {
	var (
		entryToRemove    = api.ManifestEntry{}
		longestPathEntry = api.ManifestEntry{}
	)

	mroot, isEncrypted, err := client.DownloadManifest(mhash)
	if err != nil {
		utils.Fatalf("Manifest download failed: %v", err)
	}

	// See if we path is in this Manifest or do we have to dig deeper
	for _, entry := range mroot.Entries {
		if path == entry.Path {
			entryToRemove = entry
		} else {
			if entry.ContentType == api.ManifestType {
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
		newHash := removeEntryFromManifest(client, longestPathEntry.Hash, newPath)

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

	newManifestHash, err := client.UploadManifest(mroot, isEncrypted)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	return newManifestHash
}
