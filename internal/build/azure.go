// Copyright 2016 The go-ethereum Authors
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

package build

import (
	"fmt"
	"os"

	storage "github.com/Azure/azure-storage-go"
)

// AzureBlobstoreConfig is an authentication and configuration struct containing
// the data needed by the Azure SDK to interact with a speicifc container in the
// blobstore.
type AzureBlobstoreConfig struct {
	Account   string // Account name to authorize API requests with
	Token     string // Access token for the above account
	Container string // Blob container to upload files into
}

// AzureBlobstoreUpload uploads a local file to the Azure Blob Storage. Note, this
// method assumes a max file size of 64MB (Azure limitation). Larger files will
// need a multi API call approach implemented.
//
// See: https://msdn.microsoft.com/en-us/library/azure/dd179451.aspx#Anchor_3
func AzureBlobstoreUpload(path string, name string, config AzureBlobstoreConfig) error {
	if *DryRunFlag {
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

// AzureBlobstoreList lists all the files contained within an azure blobstore.
func AzureBlobstoreList(config AzureBlobstoreConfig) ([]storage.Blob, error) {
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

// AzureBlobstoreDelete iterates over a list of files to delete and removes them
// from the blobstore.
func AzureBlobstoreDelete(config AzureBlobstoreConfig, blobs []storage.Blob) error {
	if *DryRunFlag {
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
