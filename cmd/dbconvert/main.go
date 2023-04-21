// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

var (
	dbDirFlag = &flags.DirectoryFlag{
		Name:  "dbdir",
		Usage: "Directory where the database resides",
		Value: flags.DirectoryString(filepath.Join(node.DefaultDataDir(), "geth")),
	}
	ldbToPebble = &cli.Command{
		Name:  "to-pebble",
		Usage: "Convert (destructively) ldb database to pebble",
		Flags: []cli.Flag{
			dbDirFlag,
		},
		Description: `
	dbconvert --datadir /my/datadir to-pebble

Will open the leveldb database in the given datadir, an create a pebble 
database in the same location. 

OBS! This method _will_, on success, delete the original database. 
If this method is aborted during execution, both databases will be non-functional.
`,
		Action: convertToPebble,
	}
	pebbleToLdb = &cli.Command{
		Name:  "to-ldb",
		Usage: "Convert (destructively) pebble database to leveldb",
		Flags: []cli.Flag{
			dbDirFlag,
		},
		Description: `
	dbconvert --datadir /my/datadir to-ldb

Will open the pebble database in the given datadir, an create a leveldb 
database in the same location. 
OBS! This method _will_, on success, delete the original database. 
If this method is aborted during execution, both databases will be non-functional.
`,
		Action: convertToLdb,
	}
)

var app = flags.NewApp("DB conversion utility")

func init() {
	app.Name = "DB Converter"
	app.Commands = []*cli.Command{
		ldbToPebble,
		pebbleToLdb,
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func convertToPebble(ctx *cli.Context) error {
	return convert(ctx, true)
}

func convertToLdb(ctx *cli.Context) error {
	return convert(ctx, false)
}

func convert(ctx *cli.Context, toPebble bool) error {
	var (
		err      error
		srcDbDir = ctx.String(dbDirFlag.Name) // the database directory to read from.
		dstDbdir string                       // the database destination directory to write to.
	)
	if srcDbDir, err = filepath.Abs(srcDbDir); err != nil {
		return err
	}
	{
		parts := strings.Split(srcDbDir, string(os.PathSeparator))
		parent := parts[:len(parts)-1]
		dstDbdir = string(os.PathSeparator) + filepath.Join(append(parent, ".tempdir")...)
	}
	var (
		cache   = 4 * 1024 // Cache 4GB
		handles = 4 * 1024 // Handles
		src     ethdb.Database
		dst     ethdb.Database
	)

	if toPebble {
		src, err = rawdb.NewLevelDBDatabase(srcDbDir, cache, handles, "", false)
		if err != nil {
			return err
		}
		dst, err = rawdb.NewPebbleDBDatabase(dstDbdir, cache, handles, "", false)
		if err != nil {
			src.Close()
			return err
		}
	} else {
		src, err = rawdb.NewPebbleDBDatabase(srcDbDir, cache, handles, "", false)
		if err != nil {
			return err
		}
		dst, err = rawdb.NewLevelDBDatabase(dstDbdir, cache, handles, "", false)
		if err != nil {
			src.Close()
			return err
		}
	}
	if err := copyDb(src, dst, true); err != nil {
		src.Close()
		dst.Close()
		return err
	}
	src.Close()
	dst.Close()
	// Let's check if we have ancients. If so, move them to the new directory
	if _, err := os.Stat(filepath.Join(srcDbDir, "ancient")); err == nil {
		// Yup, let's move the ancients
		var (
			from = filepath.Join(srcDbDir, "ancient")
			to   = filepath.Join(dstDbdir, "ancient")
		)
		log.Info("Moving ancients", "from", from, "to", to)
		if err := os.Rename(from, to); err != nil {
			return err
		}
	}
	// Now we need to get rid of the old datadir.
	log.Info("Deleting old directory", "dir", srcDbDir)
	if err := os.RemoveAll(srcDbDir); err != nil {
		return err
	}
	// And swap in the new database in it's place
	log.Info("Moving new directory", "from", dstDbdir, "to", srcDbDir)
	if err := os.Rename(dstDbdir, srcDbDir); err != nil {
		return err
	}
	return nil
}

func copyDb(src, dst ethdb.Database, deleteOnCopy bool) error {
	var (
		batch     = dst.NewBatch()
		delbatch  = src.NewBatch()
		it        = src.NewIterator(nil, nil)
		logged    time.Time
		count     = 0
		totalSize uint64
		start     = time.Now()
	)

	flush := func(force bool) error {
		if err := batch.Write(); err != nil {
			return err
		}
		batch.Reset()
		// If we want to delete from the source, we should
		// first close the iterator.
		// Do the write every 100K items or so
		if deleteOnCopy && (count%100_000 == 0 || force) {
			log.Info("Releasing iterator, flushing deletes")
			it.Release()
			if err := delbatch.Write(); err != nil {
				return err
			}
			it = src.NewIterator(nil, nil)
		}
		return nil
	}
	for it.Next() {
		k, v := it.Key(), it.Value()
		count++
		// Add k/v to destination db
		if err := batch.Put(k, v); err != nil {
			return err
		}
		totalSize += uint64(len(v))
		// Delete key from source db
		if deleteOnCopy {
			if err := delbatch.Delete(k); err != nil {
				return err
			}
		}
		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := flush(false); err != nil {
				return err
			}
		}
		if time.Since(logged) > 8*time.Second {
			log.Info("Converting database", "elapsed", time.Since(start),
				"items", count,
				"size", common.StorageSize(totalSize))
			logged = time.Now()
		}
	}
	if err := flush(true); err != nil {
		return err
	}
	if err := it.Error(); err != nil {
		return err
	}
	if err := src.Compact(nil, nil); err != nil {
		return err
	}
	log.Info("Converted database", "elapsed", time.Since(start),
		"items", count,
		"size", common.StorageSize(totalSize))
	return nil
}
