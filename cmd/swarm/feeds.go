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

// Command feed allows the user to create and update signed Swarm feeds
package main

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/cmd/utils"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/storage/feed"
	"gopkg.in/urfave/cli.v1"
)

func NewGenericSigner(ctx *cli.Context) feed.Signer {
	return feed.NewGenericSigner(getPrivKey(ctx))
}

func getTopic(ctx *cli.Context) (topic feed.Topic) {
	var name = ctx.String(SwarmFeedNameFlag.Name)
	var relatedTopic = ctx.String(SwarmFeedTopicFlag.Name)
	var relatedTopicBytes []byte
	var err error

	if relatedTopic != "" {
		relatedTopicBytes, err = hexutil.Decode(relatedTopic)
		if err != nil {
			utils.Fatalf("Error parsing topic: %s", err)
		}
	}

	topic, err = feed.NewTopic(name, relatedTopicBytes)
	if err != nil {
		utils.Fatalf("Error parsing topic: %s", err)
	}
	return topic
}

// swarm feed create <frequency> [--name <name>] [--data <0x Hexdata> [--multihash=false]]
// swarm feed update <Manifest Address or ENS domain> <0x Hexdata> [--multihash=false]
// swarm feed info <Manifest Address or ENS domain>

func feedCreateManifest(ctx *cli.Context) {
	var (
		bzzapi = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client = swarm.NewClient(bzzapi)
	)

	newFeedUpdateRequest := feed.NewFirstRequest(getTopic(ctx))
	newFeedUpdateRequest.Feed.User = feedGetUser(ctx)

	manifestAddress, err := client.CreateFeedWithManifest(newFeedUpdateRequest)
	if err != nil {
		utils.Fatalf("Error creating feed manifest: %s", err.Error())
		return
	}
	fmt.Println(manifestAddress) // output manifest address to the user in a single line (useful for other commands to pick up)

}

func feedUpdate(ctx *cli.Context) {
	args := ctx.Args()

	var (
		bzzapi                  = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client                  = swarm.NewClient(bzzapi)
		manifestAddressOrDomain = ctx.String(SwarmFeedManifestFlag.Name)
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

	var updateRequest *feed.Request
	var query *feed.Query

	if manifestAddressOrDomain == "" {
		query = new(feed.Query)
		query.User = signer.Address()
		query.Topic = getTopic(ctx)

	}

	// Retrieve a feed update request
	updateRequest, err = client.GetFeedRequest(query, manifestAddressOrDomain)
	if err != nil {
		utils.Fatalf("Error retrieving feed status: %s", err.Error())
	}

	// set the new data
	updateRequest.SetData(data)

	// sign update
	if err = updateRequest.Sign(signer); err != nil {
		utils.Fatalf("Error signing feed update: %s", err.Error())
	}

	// post update
	err = client.UpdateFeed(updateRequest)
	if err != nil {
		utils.Fatalf("Error updating feed: %s", err.Error())
		return
	}
}

func feedInfo(ctx *cli.Context) {
	var (
		bzzapi                  = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client                  = swarm.NewClient(bzzapi)
		manifestAddressOrDomain = ctx.String(SwarmFeedManifestFlag.Name)
	)

	var query *feed.Query
	if manifestAddressOrDomain == "" {
		query = new(feed.Query)
		query.Topic = getTopic(ctx)
		query.User = feedGetUser(ctx)
	}

	metadata, err := client.GetFeedRequest(query, manifestAddressOrDomain)
	if err != nil {
		utils.Fatalf("Error retrieving feed metadata: %s", err.Error())
		return
	}
	encodedMetadata, err := metadata.MarshalJSON()
	if err != nil {
		utils.Fatalf("Error encoding metadata to JSON for display:%s", err)
	}
	fmt.Println(string(encodedMetadata))
}

func feedGetUser(ctx *cli.Context) common.Address {
	var user = ctx.String(SwarmFeedUserFlag.Name)
	if user != "" {
		return common.HexToAddress(user)
	}
	pk := getPrivKey(ctx)
	if pk == nil {
		utils.Fatalf("Cannot read private key. Must specify --user or --bzzaccount")
	}
	return crypto.PubkeyToAddress(pk.PublicKey)

}
