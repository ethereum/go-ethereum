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
	"archive/tar"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage/localstore"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"gopkg.in/urfave/cli.v1"
)

var legacyKeyIndex = byte(0)
var keyData = byte(6)

type dpaDBIndex struct {
	Idx    uint64
	Access uint64
}

var dbCommand = cli.Command{
	Name:               "db",
	CustomHelpTemplate: helpTemplate,
	Usage:              "manage the local chunk database",
	ArgsUsage:          "db COMMAND",
	Description:        "Manage the local chunk database",
	Subcommands: []cli.Command{
		{
			Action:             dbExport,
			CustomHelpTemplate: helpTemplate,
			Name:               "export",
			Usage:              "export a local chunk database as a tar archive (use - to send to stdout)",
			ArgsUsage:          "<chunkdb> <file>",
			Description: `
Export a local chunk database as a tar archive (use - to send to stdout).

    swarm db export ~/.ethereum/swarm/bzz-KEY/chunks chunks.tar

The export may be quite large, consider piping the output through the Unix
pv(1) tool to get a progress bar:

    swarm db export ~/.ethereum/swarm/bzz-KEY/chunks - | pv > chunks.tar
`,
		},
		{
			Action:             dbImport,
			CustomHelpTemplate: helpTemplate,
			Name:               "import",
			Usage:              "import chunks from a tar archive into a local chunk database (use - to read from stdin)",
			ArgsUsage:          "<chunkdb> <file>",
			Description: `Import chunks from a tar archive into a local chunk database (use - to read from stdin).

    swarm db import ~/.ethereum/swarm/bzz-KEY/chunks chunks.tar

The import may be quite large, consider piping the input through the Unix
pv(1) tool to get a progress bar:

    pv chunks.tar | swarm db import ~/.ethereum/swarm/bzz-KEY/chunks -`,
			Flags: []cli.Flag{
				SwarmLegacyFlag,
			},
		},
	},
}

func dbExport(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 3 {
		utils.Fatalf("invalid arguments, please specify both <chunkdb> (path to a local chunk database), <file> (path to write the tar archive to, - for stdout) and the base key")
	}

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

	isLegacy := localstore.IsLegacyDatabase(args[0])
	if isLegacy {
		count, err := exportLegacy(args[0], common.Hex2Bytes(args[2]), out)
		if err != nil {
			utils.Fatalf("error exporting legacy local chunk database: %s", err)
		}

		log.Info(fmt.Sprintf("successfully exported %d chunks from legacy db", count))
		return
	}

	store, err := openLDBStore(args[0], common.Hex2Bytes(args[2]))
	if err != nil {
		utils.Fatalf("error opening local chunk database: %s", err)
	}
	defer store.Close()

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

	legacy := ctx.IsSet(SwarmLegacyFlag.Name)

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

	count, err := store.Import(in, legacy)
	if err != nil {
		utils.Fatalf("error importing local chunk database: %s", err)
	}

	log.Info(fmt.Sprintf("successfully imported %d chunks", count))
}

func openLDBStore(path string, basekey []byte) (*localstore.DB, error) {
	if _, err := os.Stat(filepath.Join(path, "CURRENT")); err != nil {
		return nil, fmt.Errorf("invalid chunkdb path: %s", err)
	}

	return localstore.New(path, basekey, nil)
}

func decodeIndex(data []byte, index *dpaDBIndex) error {
	dec := rlp.NewStream(bytes.NewReader(data), 0)
	return dec.Decode(index)
}

func getDataKey(idx uint64, po uint8) []byte {
	key := make([]byte, 10)
	key[0] = keyData
	key[1] = po
	binary.BigEndian.PutUint64(key[2:], idx)

	return key
}

func exportLegacy(path string, basekey []byte, out io.Writer) (int64, error) {
	tw := tar.NewWriter(out)
	defer tw.Close()
	db, err := leveldb.OpenFile(path, &opt.Options{OpenFilesCacheCapacity: 128})
	if err != nil {
		return 0, err
	}
	defer db.Close()

	it := db.NewIterator(nil, nil)
	defer it.Release()
	var count int64
	for ok := it.Seek([]byte{legacyKeyIndex}); ok; ok = it.Next() {
		key := it.Key()
		if (key == nil) || (key[0] != legacyKeyIndex) {
			break
		}

		var index dpaDBIndex

		hash := key[1:]
		decodeIndex(it.Value(), &index)

		po := uint8(chunk.Proximity(basekey, hash))

		datakey := getDataKey(index.Idx, po)
		data, err := db.Get(datakey, nil)
		if err != nil {
			log.Crit(fmt.Sprintf("Chunk %x found but could not be accessed: %v, %x", key, err, datakey))
			continue
		}

		hdr := &tar.Header{
			Name: hex.EncodeToString(hash),
			Mode: 0644,
			Size: int64(len(data)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return count, err
		}
		if _, err := tw.Write(data); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}
