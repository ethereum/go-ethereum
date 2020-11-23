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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"gopkg.in/urfave/cli.v1"
)

func dbExport(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 {
		utils.Fatalf("invalid arguments, please specify both <chunkdb> (path to a local chunk database), <file> (path to write the tar archive to, - for stdout) and the base key")
	}

	store, err := openLDBStore(args[0], common.Hex2Bytes(args[2]))
	if err != nil {
		utils.Fatalf("error opening local chunk database: %s", err)
	}
	defer store.Close()

	var out io.Writer
	if args[1] == "-" {
		out = os.Stdout
	} else {
		f, err := os.Create(args[1])
		if err != nil {
			utils.Fatalf("error opening output file: %s", err)
		}
		defer f.Close()
		out = f
	}

	count, err := store.Export(out)
	if err != nil {
		utils.Fatalf("error exporting local chunk database: %s", err)
	}

	log.Info(fmt.Sprintf("successfully exported %d chunks", count))
}

func dbImport(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 {
		utils.Fatalf("invalid arguments, please specify both <chunkdb> (path to a local chunk database), <file> (path to read the tar archive from, - for stdin) and the base key")
	}

	store, err := openLDBStore(args[0], common.Hex2Bytes(args[2]))
	if err != nil {
		utils.Fatalf("error opening local chunk database: %s", err)
	}
	defer store.Close()

	var in io.Reader
	if args[1] == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(args[1])
		if err != nil {
			utils.Fatalf("error opening input file: %s", err)
		}
		defer f.Close()
		in = f
	}

	count, err := store.Import(in)
	if err != nil {
		utils.Fatalf("error importing local chunk database: %s", err)
	}

	log.Info(fmt.Sprintf("successfully imported %d chunks", count))
}

func dbClean(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 2 {
		utils.Fatalf("invalid arguments, please specify <chunkdb> (path to a local chunk database) and the base key")
	}

	store, err := openLDBStore(args[0], common.Hex2Bytes(args[1]))
	if err != nil {
		utils.Fatalf("error opening local chunk database: %s", err)
	}
	defer store.Close()

	store.Cleanup()
}

func openLDBStore(path string, basekey []byte) (*storage.LDBStore, error) {
	if _, err := os.Stat(filepath.Join(path, "CURRENT")); err != nil {
		return nil, fmt.Errorf("invalid chunkdb path: %s", err)
	}

	storeparams := storage.NewDefaultStoreParams()
	ldbparams := storage.NewLDBStoreParams(storeparams, path)
	ldbparams.BaseKey = basekey
	return storage.NewLDBStore(ldbparams)
}
