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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/fuse"
	"gopkg.in/urfave/cli.v1"
)

var fsCommand = cli.Command{
	Name:               "fs",
	CustomHelpTemplate: helpTemplate,
	Usage:              "perform FUSE operations",
	ArgsUsage:          "fs COMMAND",
	Description:        "Performs FUSE operations by mounting/unmounting/listing mount points. This assumes you already have a Swarm node running locally. For all operation you must reference the correct path to bzzd.ipc in order to communicate with the node",
	Subcommands: []cli.Command{
		{
			Action:             mount,
			CustomHelpTemplate: helpTemplate,
			Name:               "mount",
			Usage:              "mount a swarm hash to a mount point",
			ArgsUsage:          "swarm fs mount <manifest hash> <mount point>",
			Description:        "Mounts a Swarm manifest hash to a given mount point. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
		},
		{
			Action:             unmount,
			CustomHelpTemplate: helpTemplate,
			Name:               "unmount",
			Usage:              "unmount a swarmfs mount",
			ArgsUsage:          "swarm fs unmount <mount point>",
			Description:        "Unmounts a swarmfs mount residing at <mount point>. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
		},
		{
			Action:             listMounts,
			CustomHelpTemplate: helpTemplate,
			Name:               "list",
			Usage:              "list swarmfs mounts",
			ArgsUsage:          "swarm fs list",
			Description:        "Lists all mounted swarmfs volumes. This assumes you already have a Swarm node running locally. You must reference the correct path to your bzzd.ipc file",
		},
	},
}

func mount(cliContext *cli.Context) {
	args := cliContext.Args()
	if len(args) < 2 {
		utils.Fatalf("Usage: swarm fs mount <manifestHash> <file name>")
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
		utils.Fatalf("Usage: swarm fs unmount <mount path>")
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
		utils.Fatalf("encountered an error calling the RPC endpoint while listing mounts: %v", err)
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
	endpoint := getIPCEndpoint(ctx)
	log.Info("IPC endpoint", "path", endpoint)
	return rpc.Dial(endpoint)
}

func getIPCEndpoint(ctx *cli.Context) string {
	cfg := defaultNodeConfig
	utils.SetNodeConfig(ctx, &cfg)

	endpoint := cfg.IPCEndpoint()

	if strings.HasPrefix(endpoint, "rpc:") || strings.HasPrefix(endpoint, "ipc:") {
		// Backwards compatibility with geth < 1.5 which required
		// these prefixes.
		endpoint = endpoint[4:]
	}
	return endpoint
}
