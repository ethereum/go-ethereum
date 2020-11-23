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
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/cmd/utils"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/storage/mru"
	"gopkg.in/urfave/cli.v1"
)

func NewGenericSigner(ctx *cli.Context) mru.Signer {
	return mru.NewGenericSigner(getPrivKey(ctx))
}

// swarm resource create <frequency> [--name <name>] [--data <0x Hexdata> [--multihash=false]]
// swarm resource update <Manifest Address or ENS domain> <0x Hexdata> [--multihash=false]
// swarm resource info <Manifest Address or ENS domain>

func resourceCreate(ctx *cli.Context) {
	args := ctx.Args()

	var (
		bzzapi      = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client      = swarm.NewClient(bzzapi)
		multihash   = ctx.Bool(SwarmResourceMultihashFlag.Name)
		initialData = ctx.String(SwarmResourceDataOnCreateFlag.Name)
		name        = ctx.String(SwarmResourceNameFlag.Name)
	)

	if len(args) < 1 {
		fmt.Println("Incorrect number of arguments")
		cli.ShowCommandHelpAndExit(ctx, "create", 1)
		return
	}
	signer := NewGenericSigner(ctx)
	frequency, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		fmt.Printf("Frequency formatting error: %s\n", err.Error())
		cli.ShowCommandHelpAndExit(ctx, "create", 1)
		return
	}

	metadata := mru.ResourceMetadata{
		Name:      name,
		Frequency: frequency,
		Owner:     signer.Address(),
	}

	var newResourceRequest *mru.Request
	if initialData != "" {
		initialDataBytes, err := hexutil.Decode(initialData)
		if err != nil {
			fmt.Printf("Error parsing data: %s\n", err.Error())
			cli.ShowCommandHelpAndExit(ctx, "create", 1)
			return
		}
		newResourceRequest, err = mru.NewCreateUpdateRequest(&metadata)
		if err != nil {
			utils.Fatalf("Error creating new resource request: %s", err)
		}
		newResourceRequest.SetData(initialDataBytes, multihash)
		if err = newResourceRequest.Sign(signer); err != nil {
			utils.Fatalf("Error signing resource update: %s", err.Error())
		}
	} else {
		newResourceRequest, err = mru.NewCreateRequest(&metadata)
		if err != nil {
			utils.Fatalf("Error creating new resource request: %s", err)
		}
	}

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
		bzzapi    = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client    = swarm.NewClient(bzzapi)
		multihash = ctx.Bool(SwarmResourceMultihashFlag.Name)
	)

	if len(args) < 2 {
		fmt.Println("Incorrect number of arguments")
		cli.ShowCommandHelpAndExit(ctx, "update", 1)
		return
	}
	signer := NewGenericSigner(ctx)
	manifestAddressOrDomain := args[0]
	data, err := hexutil.Decode(args[1])
	if err != nil {
		utils.Fatalf("Error parsing data: %s", err.Error())
		return
	}

	// Retrieve resource status and metadata out of the manifest
	updateRequest, err := client.GetResourceMetadata(manifestAddressOrDomain)
	if err != nil {
		utils.Fatalf("Error retrieving resource status: %s", err.Error())
	}

	// set the new data
	updateRequest.SetData(data, multihash)

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
		bzzapi = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		client = swarm.NewClient(bzzapi)
	)
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Println("Incorrect number of arguments.")
		cli.ShowCommandHelpAndExit(ctx, "info", 1)
		return
	}
	manifestAddressOrDomain := args[0]
	metadata, err := client.GetResourceMetadata(manifestAddressOrDomain)
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
