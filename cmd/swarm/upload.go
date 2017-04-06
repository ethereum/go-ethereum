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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

func upload(ctx *cli.Context) {

	args := ctx.Args()
	var (
		bzzapi       = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		recursive    = ctx.GlobalBool(SwarmRecursiveUploadFlag.Name)
		wantManifest = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		defaultPath  = ctx.GlobalString(SwarmUploadDefaultPath.Name)
		fromStdin    = ctx.GlobalBool(SwarmUpFromStdinFlag.Name)
		mimeType     = ctx.GlobalString(SwarmUploadMimeType.Name)
	)

	var client = swarm.NewClient(bzzapi)
	var entry swarm.ManifestEntry
	var file string

	if len(args) != 1 {
		if fromStdin {
			tmp, err := ioutil.TempFile("", "swarm-stdin")
			if err != nil {
				utils.Fatalf("error create tempfile: %s", err)
			}
			defer os.Remove(tmp.Name())
			n, err := io.Copy(tmp, os.Stdin)
			if err != nil {
				utils.Fatalf("error copying stdin to tempfile: %s", err)
			} else if n == 0 {
				utils.Fatalf("error reading from stdin: zero length")
			}
			file = tmp.Name()
		} else {
			utils.Fatalf("Need filename as the first and only argument")
		}
	} else {
		file = args[0]
	}

	fi, err := os.Stat(expandPath(file))
	if err != nil {
		utils.Fatalf("Failed to stat file: %v", err)
	}
	if fi.IsDir() {
		if !recursive {
			utils.Fatalf("Argument is a directory and recursive upload is disabled")
		}
		if !wantManifest {
			utils.Fatalf("Manifest is required for directory uploads")
		}
		mhash, err := client.UploadDirectory(file, defaultPath)
		if err != nil {
			utils.Fatalf("Failed to upload directory: %v", err)
		}
		fmt.Println(mhash)
		return
	}
	entry, err = client.UploadFile(file, fi, mimeType)
	if err != nil {
		utils.Fatalf("Upload failed: %v", err)
	}
	mroot := swarm.Manifest{Entries: []swarm.ManifestEntry{entry}}
	if !wantManifest {
		// Print the manifest. This is the only output to stdout.
		mrootJSON, _ := json.MarshalIndent(mroot, "", "  ")
		fmt.Println(string(mrootJSON))
		return
	}
	hash, err := client.UploadManifest(mroot)
	if err != nil {
		utils.Fatalf("Manifest upload failed: %v", err)
	}
	fmt.Println(hash)
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
