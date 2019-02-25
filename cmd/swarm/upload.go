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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/log"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var upCommand = cli.Command{
	Action:             upload,
	CustomHelpTemplate: helpTemplate,
	Name:               "up",
	Usage:              "uploads a file or directory to swarm using the HTTP API",
	ArgsUsage:          "<file>",
	Flags:              []cli.Flag{SwarmEncryptedFlag},
	Description:        "uploads a file or directory to swarm using the HTTP API and prints the root hash",
}

func upload(ctx *cli.Context) {
	args := ctx.Args()
	var (
		bzzapi          = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		recursive       = ctx.GlobalBool(SwarmRecursiveFlag.Name)
		wantManifest    = ctx.GlobalBoolT(SwarmWantManifestFlag.Name)
		defaultPath     = ctx.GlobalString(SwarmUploadDefaultPath.Name)
		fromStdin       = ctx.GlobalBool(SwarmUpFromStdinFlag.Name)
		mimeType        = ctx.GlobalString(SwarmUploadMimeType.Name)
		client          = swarm.NewClient(bzzapi)
		toEncrypt       = ctx.Bool(SwarmEncryptedFlag.Name)
		autoDefaultPath = false
		file            string
	)
	if autoDefaultPathString := os.Getenv(SwarmAutoDefaultPath); autoDefaultPathString != "" {
		b, err := strconv.ParseBool(autoDefaultPathString)
		if err != nil {
			utils.Fatalf("invalid environment variable %s: %v", SwarmAutoDefaultPath, err)
		}
		autoDefaultPath = b
	}
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
		file = expandPath(args[0])
	}

	if !wantManifest {
		f, err := swarm.Open(file)
		if err != nil {
			utils.Fatalf("Error opening file: %s", err)
		}
		defer f.Close()
		hash, err := client.UploadRaw(f, f.Size, toEncrypt)
		if err != nil {
			utils.Fatalf("Upload failed: %s", err)
		}
		fmt.Println(hash)
		return
	}

	stat, err := os.Stat(file)
	if err != nil {
		utils.Fatalf("Error opening file: %s", err)
	}

	// define a function which either uploads a directory or single file
	// based on the type of the file being uploaded
	var doUpload func() (hash string, err error)
	if stat.IsDir() {
		doUpload = func() (string, error) {
			if !recursive {
				return "", errors.New("Argument is a directory and recursive upload is disabled")
			}
			if autoDefaultPath && defaultPath == "" {
				defaultEntryCandidate := path.Join(file, "index.html")
				log.Debug("trying to find default path", "path", defaultEntryCandidate)
				defaultEntryStat, err := os.Stat(defaultEntryCandidate)
				if err == nil && !defaultEntryStat.IsDir() {
					log.Debug("setting auto detected default path", "path", defaultEntryCandidate)
					defaultPath = defaultEntryCandidate
				}
			}
			if defaultPath != "" {
				// construct absolute default path
				absDefaultPath, _ := filepath.Abs(defaultPath)
				absFile, _ := filepath.Abs(file)
				// make sure absolute directory ends with only one "/"
				// to trim it from absolute default path and get relative default path
				absFile = strings.TrimRight(absFile, "/") + "/"
				if absDefaultPath != "" && absFile != "" && strings.HasPrefix(absDefaultPath, absFile) {
					defaultPath = strings.TrimPrefix(absDefaultPath, absFile)
				}
			}
			return client.UploadDirectory(file, defaultPath, "", toEncrypt)
		}
	} else {
		doUpload = func() (string, error) {
			f, err := swarm.Open(file)
			if err != nil {
				return "", fmt.Errorf("error opening file: %s", err)
			}
			defer f.Close()
			if mimeType != "" {
				f.ContentType = mimeType
			}
			return client.Upload(f, "", toEncrypt)
		}
	}
	hash, err := doUpload()
	if err != nil {
		utils.Fatalf("Upload failed: %s", err)
	}
	fmt.Println(hash)
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if i := strings.Index(p, ":"); i > 0 {
		return p
	}
	if i := strings.Index(p, "@"); i > 0 {
		return p
	}
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
