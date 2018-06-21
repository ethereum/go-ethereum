// Copyright 2018 The go-ethereum Authors
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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/fuse"
	"gopkg.in/urfave/cli.v1"
)

func mount(cliContext *cli.Context) {
	args := cliContext.Args()
	if len(args) < 2 {
		utils.Fatalf("Usage: swarm fs mount --ipcpath <path to bzzd.ipc> <manifestHash> <file name>")
	}

	client, err := dialRPC(cliContext)
	if err != nil {
		utils.Fatalf("had an error dailing to RPC endpoint: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mf := &fuse.MountInfo{}
	mountPoint, err := filepath.Abs(filepath.Clean(args[1]))
	if err != nil {
		utils.Fatalf("error expanding path for mount point: %v", err)
	}
	err = client.CallContext(ctx, mf, "swarmfs_mount", args[0], mountPoint)
	if err != nil {
		utils.Fatalf("had an error calling the RPC endpoint while mounting: %v", err)
	}
}

func unmount(cliContext *cli.Context) {
	args := cliContext.Args()

	if len(args) < 1 {
		utils.Fatalf("Usage: swarm fs unmount --ipcpath <path to bzzd.ipc> <mount path>")
	}
	client, err := dialRPC(cliContext)
	if err != nil {
		utils.Fatalf("had an error dailing to RPC endpoint: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mf := fuse.MountInfo{}
	err = client.CallContext(ctx, &mf, "swarmfs_unmount", args[0])
	if err != nil {
		utils.Fatalf("encountered an error calling the RPC endpoint while unmounting: %v", err)
	}
	fmt.Printf("%s\n", mf.LatestManifest) //print the latest manifest hash for user reference
}

func listMounts(cliContext *cli.Context) {
	client, err := dialRPC(cliContext)
	if err != nil {
		utils.Fatalf("had an error dailing to RPC endpoint: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mf := []fuse.MountInfo{}
	err = client.CallContext(ctx, &mf, "swarmfs_listmounts")
	if err != nil {
		utils.Fatalf("encountered an error calling the RPC endpoint while unmounting: %v", err)
	}
	if len(mf) == 0 {
		fmt.Print("Could not found any swarmfs mounts. Please make sure you've specified the correct RPC endpoint\n")
	} else {
		fmt.Printf("Found %d swarmfs mount(s):\n", len(mf))
		for i, mountInfo := range mf {
			fmt.Printf("%d:\n", i)
			fmt.Printf("\tMount point: %s\n", mountInfo.MountPoint)
			fmt.Printf("\tLatest Manifest: %s\n", mountInfo.LatestManifest)
			fmt.Printf("\tStart Manifest: %s\n", mountInfo.StartManifest)
		}
	}
}

func dialRPC(ctx *cli.Context) (*rpc.Client, error) {
	var endpoint string

	if ctx.IsSet(utils.IPCPathFlag.Name) {
		endpoint = ctx.String(utils.IPCPathFlag.Name)
	} else {
		utils.Fatalf("swarm ipc endpoint not specified")
	}

	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	} else if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		// Backwards compatibility with geth < 1.5 which required
		// these prefixes.
		endpoint = endpoint[4:]
	}
	return rpc.Dial(endpoint)
}
