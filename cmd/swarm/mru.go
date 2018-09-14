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

// Command resource allows the user to create and update signed mutable resource updates
package main

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/cmd/utils"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"
	"gopkg.in/urfave/cli.v1"
)

func NewGenericSigner(ctx *cli.Context) mru.Signer {
	return mru.NewGenericSigner(getPrivKey(ctx))
}

func getTopic(ctx *cli.Context) (topic mru.Topic) {
	var name = ctx.String(SwarmResourceNameFlag.Name)
	var relatedTopic = ctx.String(SwarmResourceTopicFlag.Name)
	var relatedTopicBytes []byte
	var err error

	if relatedTopic != "" {
		relatedTopicBytes, err = hexutil.Decode(relatedTopic)
		if err != nil {
			utils.Fatalf("Error parsing topic: %s", err)
		}
	}

	topic, err = mru.NewTopic(name, relatedTopicBytes)
	if err != nil {
		utils.Fatalf("Error parsing topic: %s", err)
	}
	return topic
}

// swarm resource create <frequency> [--name <name>] [--data <0x Hexdata> [--multihash=false]]
// swarm resource update <Manifest Address or ENS domain> <0x Hexdata> [--multihash=false]
// swarm resource info <Manifest Address or ENS domain>

func resourceCreate(ctx *cli.Context) {
	var (
		bzzapi = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client = swarm.NewClient(bzzapi)
	)

	newResourceRequest := mru.NewFirstRequest(getTopic(ctx))
	newResourceRequest.View.User = resourceGetUser(ctx)

	manifestAddress, err := client.CreateResource(newResourceRequest)
	if err != nil {
		utils.Fatalf("Error creating resource: %s", err.Error())
		return
	}
	fmt.Println(manifestAddress) // output manifest address to the user in a single line (useful for other commands to pick up)

}

func resourceUpdate(ctx *cli.Context) {
	args := ctx.Args()

	var (
		bzzapi                  = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client                  = swarm.NewClient(bzzapi)
		manifestAddressOrDomain = ctx.String(SwarmResourceManifestFlag.Name)
	)

	if len(args) < 1 {
		fmt.Println("Incorrect number of arguments")
		cli.ShowCommandHelpAndExit(ctx, "update", 1)
		return
	}

	signer := NewGenericSigner(ctx)

	data, err := hexutil.Decode(args[0])
	if err != nil {
		utils.Fatalf("Error parsing data: %s", err.Error())
		return
	}

	var updateRequest *mru.Request
	var query *mru.Query

	if manifestAddressOrDomain == "" {
		query = new(mru.Query)
		query.User = signer.Address()
		query.Topic = getTopic(ctx)

	}

	// Retrieve resource status and metadata out of the manifest
	updateRequest, err = client.GetResourceMetadata(query, manifestAddressOrDomain)
	if err != nil {
		utils.Fatalf("Error retrieving resource status: %s", err.Error())
	}

	// set the new data
	updateRequest.SetData(data)

	// sign update
	if err = updateRequest.Sign(signer); err != nil {
		utils.Fatalf("Error signing resource update: %s", err.Error())
	}

	// post update
	err = client.UpdateResource(updateRequest)
	if err != nil {
		utils.Fatalf("Error updating resource: %s", err.Error())
		return
	}
}

func resourceInfo(ctx *cli.Context) {
	var (
		bzzapi                  = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client                  = swarm.NewClient(bzzapi)
		manifestAddressOrDomain = ctx.String(SwarmResourceManifestFlag.Name)
	)

	var query *mru.Query
	if manifestAddressOrDomain == "" {
		query = new(mru.Query)
		query.Topic = getTopic(ctx)
		query.User = resourceGetUser(ctx)
	}

	metadata, err := client.GetResourceMetadata(query, manifestAddressOrDomain)
	if err != nil {
		utils.Fatalf("Error retrieving resource metadata: %s", err.Error())
		return
	}
	encodedMetadata, err := metadata.MarshalJSON()
	if err != nil {
		utils.Fatalf("Error encoding metadata to JSON for display:%s", err)
	}
	fmt.Println(string(encodedMetadata))
}

func resourceGetUser(ctx *cli.Context) common.Address {
	var user = ctx.String(SwarmResourceUserFlag.Name)
	if user != "" {
		return common.HexToAddress(user)
	}
	pk := getPrivKey(ctx)
	if pk == nil {
		utils.Fatalf("Cannot read private key. Must specify --user or --bzzaccount")
	}
	return crypto.PubkeyToAddress(pk.PublicKey)

}
