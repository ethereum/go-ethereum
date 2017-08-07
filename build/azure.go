// Copyright 2017 The go-ethereum Authors
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

// +build none

/*
The azure command signs and uploads release binaries to a Microsoft Azure bucket and is called from
Continuous Integration scripts.

Usage: go run build/azure.go <command> <command flags/arguments>

Available commands are:

   upload     [ -store s ] [ -signer key-var ] [ file... ] -- uploads build artefacts
   purge      [ -store s ] [ -days threshold ]             -- purges old archives from the blobstore
   list       [ -store s ]                                 -- lists contents of the blobstore

For all commands, -n prevents the actual upload/download (dry run mode).
*/
package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	storage "github.com/Azure/azure-storage-go"
	"github.com/ethereum/go-ethereum/internal/build"
	"golang.org/x/crypto/openpgp"
)

const defaultStore = "gethstore/builds"

func main() {
	log.SetFlags(log.Lshortfile)

	if _, err := os.Stat(filepath.Join("build", "azure.go")); os.IsNotExist(err) {
		log.Fatal("this script must be run from the root of the repository")
	}
	if len(os.Args) < 2 {
		log.Fatal("need subcommand as first argument")
	}
	switch os.Args[1] {
	case "upload":
		doUpload(os.Args[2:])
	case "purge":
		doPurge(os.Args[2:])
	case "list":
		doList(os.Args[2:])
	default:
		log.Fatal("unknown command ", os.Args[1])
	}
}

func doUpload(cmdline []string) {
	var (
		signer = flag.String("signer", "", `Environment variable holding the signing key (e.g. LINUX_SIGNING_KEY)`)
		store  = flag.String("store", defaultStore, `Destination to upload the archives to`)
	)
	flag.CommandLine.Parse(cmdline)
	build.MaybeSkipArchive(build.Env())

	for _, archive := range flag.Args() {
		uploadArchive(archive, *store, *signer)
	}
}

func doPurge(cmdline []string) {
	var (
		store = flag.String("store", defaultStore, `Destination from where to purge archives`)
		limit = flag.Int("days", 30, `Age threshold above which to delete unstalbe archives`)
	)
	flag.CommandLine.Parse(cmdline)

	if env := build.Env(); !env.IsCronJob {
		log.Printf("skipping because not a cron job")
		os.Exit(0)
	}
	// Create the azure authentication and list the current archives
	auth := newConfig(*store)
	blobs, err := blobstoreList(auth)
	if err != nil {
		log.Fatal(err)
	}
	// Iterate over the blobs, collect and sort all unstable builds
	for i := 0; i < len(blobs); i++ {
		if !strings.Contains(blobs[i].Name, "unstable") {
			blobs = append(blobs[:i], blobs[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(blobs); i++ {
		for j := i + 1; j < len(blobs); j++ {
			iTime, err := time.Parse(time.RFC1123, blobs[i].Properties.LastModified)
			if err != nil {
				log.Fatal(err)
			}
			jTime, err := time.Parse(time.RFC1123, blobs[j].Properties.LastModified)
			if err != nil {
				log.Fatal(err)
			}
			if iTime.After(jTime) {
				blobs[i], blobs[j] = blobs[j], blobs[i]
			}
		}
	}
	// Filter out all archives more recent that the given threshold
	for i, blob := range blobs {
		timestamp, _ := time.Parse(time.RFC1123, blob.Properties.LastModified)
		if time.Since(timestamp) < time.Duration(*limit)*24*time.Hour {
			blobs = blobs[:i]
			break
		}
	}
	// Delete all marked as such and return
	if err := blobstoreDelete(auth, blobs); err != nil {
		log.Fatal(err)
	}
}

func doList(cmdline []string) {
	var store = flag.String("store", defaultStore, `Blobstore to list`)
	flag.CommandLine.Parse(cmdline)
	auth := newConfig(*store)

	blobs, err := blobstoreList(auth)
	if err != nil {
		log.Fatal(err)
	}
	namelen := 0
	for _, b := range blobs {
		if len(b.Name) > namelen {
			namelen = len(b.Name)
		}
	}
	for _, b := range blobs {
		name := b.Name + strings.Repeat(" ", namelen-len(b.Name))
		fmt.Printf("%s  size: %v, last-modified: %v\n", name, b.Properties.ContentLength, b.Properties.LastModified)
	}
}

func newConfig(store string) Config {
	return Config{
		Account:   strings.Split(store, "/")[0],
		Token:     os.Getenv("AZURE_BLOBSTORE_TOKEN"),
		Container: strings.SplitN(store, "/", 2)[1],
	}
}

// uploadArchive uploads the given file to the blobstore.
func uploadArchive(archive, blobstore, signer string) {
	// If signing was requested, generate the signature files
	if signer != "" {
		pgpkey, err := base64.StdEncoding.DecodeString(os.Getenv(signer))
		if err != nil {
			log.Fatalf("invalid base64 in variable %s", signer)
		}
		if err := signFile(archive, archive+".asc", string(pgpkey)); err != nil {
			log.Fatalf("can't sign: %v", err)
		}
	}
	// If uploading to Azure was requested, push the archive possibly with its signature
	auth := newConfig(blobstore)
	if err := blobstoreUpload(archive, filepath.Base(archive), auth); err != nil {
		log.Fatal(err)
	}
	if signer != "" {
		if err := blobstoreUpload(archive+".asc", filepath.Base(archive+".asc"), auth); err != nil {
			log.Fatal(err)
		}
	}
}

// signFile parses a PGP private key from the specified string and creates a signature file
// into the output parameter of the input file.
// pgpkey should be a single key in armored format.
func signFile(input string, output string, pgpkey string) error {
	// Parse the keyring and make sure we only have a single private key in it
	keys, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(pgpkey))
	if err != nil {
		return err
	}
	if len(keys) != 1 {
		return fmt.Errorf("key count mismatch: have %d, want %d", len(keys), 1)
	}
	// Create the input and output streams for signing
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	// Generate the signature and return
	return openpgp.ArmoredDetachSign(out, keys[0], in, nil)
}

// Config is an authentication and configuration struct containing the data needed by the
// Azure SDK to interact with a speicifc container in the blobstore.
type Config struct {
	Account   string // Account name to authorize API requests with
	Token     string // Access token for the above account
	Container string // Blob container to upload files into
}

// blobstoreUpload uploads a local file to the Azure Blob Storage. Note, this method
// assumes a max file size of 64MB (Azure limitation). Larger files will need a multi API
// call approach implemented.
//
// See: https://msdn.microsoft.com/en-us/library/azure/dd179451.aspx#Anchor_3
func blobstoreUpload(path string, name string, config Config) error {
	if *build.DryRunFlag {
		fmt.Printf("would upload %q to %s/%s/%s\n", path, config.Account, config.Container, name)
		return nil
	}
	// Create an authenticated client against the Azure cloud
	rawClient, err := storage.NewBasicClient(config.Account, config.Token)
	if err != nil {
		return err
	}
	client := rawClient.GetBlobService()

	// Stream the file to upload into the designated blobstore container
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	return client.CreateBlockBlobFromReader(config.Container, name, uint64(info.Size()), in, nil)
}

// blobstoreList lists all the files contained within an azure blobstore.
func blobstoreList(config Config) ([]storage.Blob, error) {
	// Create an authenticated client against the Azure cloud
	rawClient, err := storage.NewBasicClient(config.Account, config.Token)
	if err != nil {
		return nil, err
	}
	client := rawClient.GetBlobService()

	// List all the blobs from the container and return them
	container := client.GetContainerReference(config.Container)

	blobs, err := container.ListBlobs(storage.ListBlobsParameters{
		MaxResults: 1024 * 1024 * 1024, // Yes, fetch all of them
		Timeout:    3600,               // Yes, wait for all of them
	})
	if err != nil {
		return nil, err
	}
	return blobs.Blobs, nil
}

// blobstoreDelete iterates over a list of files to delete and removes them.
func blobstoreDelete(config Config, blobs []storage.Blob) error {
	if *build.DryRunFlag {
		for _, blob := range blobs {
			fmt.Printf("would delete %s (%s) from %s/%s\n", blob.Name, blob.Properties.LastModified, config.Account, config.Container)
		}
		return nil
	}
	// Create an authenticated client against the Azure cloud
	rawClient, err := storage.NewBasicClient(config.Account, config.Token)
	if err != nil {
		return err
	}
	client := rawClient.GetBlobService()

	// Iterate over the blobs and delete them
	for _, blob := range blobs {
		if err := client.DeleteBlob(config.Container, blob.Name, nil); err != nil {
			return err
		}
	}
	return nil
}
