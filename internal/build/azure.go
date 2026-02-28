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
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// AzureBlobstoreConfig is an authentication and configuration struct containing
// the data needed by the Azure SDK to interact with a specific container in the
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
	credential, err := azblob.NewSharedKeyCredential(config.Account, config.Token)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.Account, config.Container)
	container, err := azblob.NewContainerClientWithSharedKey(u, credential, nil)
	if err != nil {
		return err
	}
	// Stream the file to upload into the designated blobstore container
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	blockblob := container.NewBlockBlobClient(name)
	_, err = blockblob.Upload(context.Background(), in, nil)
	return err
}

// AzureBlobstoreList lists all the files contained within an azure blobstore.
func AzureBlobstoreList(config AzureBlobstoreConfig) ([]*azblob.BlobItemInternal, error) {
	// Create an authenticated client against the Azure cloud
	credential, err := azblob.NewSharedKeyCredential(config.Account, config.Token)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.Account, config.Container)
	container, err := azblob.NewContainerClientWithSharedKey(u, credential, nil)
	if err != nil {
		return nil, err
	}
	var maxResults int32 = 5000
	pager := container.ListBlobsFlat(&azblob.ContainerListBlobFlatSegmentOptions{
		Maxresults: &maxResults,
	})
	var allBlobs []*azblob.BlobItemInternal
	for pager.NextPage(context.Background()) {
		res := pager.PageResponse()
		allBlobs = append(allBlobs, res.ContainerListBlobFlatSegmentResult.Segment.BlobItems...)
	}
	return allBlobs, pager.Err()
}

// AzureBlobstoreDelete iterates over a list of files to delete and removes them
// from the blobstore.
func AzureBlobstoreDelete(config AzureBlobstoreConfig, blobs []*azblob.BlobItemInternal) error {
	if *DryRunFlag {
		for _, blob := range blobs {
			fmt.Printf("would delete %s (%s) from %s/%s\n", *blob.Name, blob.Properties.LastModified, config.Account, config.Container)
		}
		return nil
	}
	// Create an authenticated client against the Azure cloud
	credential, err := azblob.NewSharedKeyCredential(config.Account, config.Token)
	if err != nil {
		return err
	}
	u := fmt.Sprintf("https://%s.blob.core.windows.net/%s", config.Account, config.Container)
	container, err := azblob.NewContainerClientWithSharedKey(u, credential, nil)
	if err != nil {
		return err
	}
	// Iterate over the blobs and delete them
	for _, blob := range blobs {
		blockblob := container.NewBlockBlobClient(*blob.Name)
		if _, err := blockblob.Delete(context.Background(), &azblob.DeleteBlobOptions{}); err != nil {
			return err
		}
		fmt.Printf("deleted  %s (%s)\n", *blob.Name, blob.Properties.LastModified)
	}
	return nil
}
