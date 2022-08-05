// Copyright 2022 The go-ethereum Authors
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
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/ethtest"
	"github.com/urfave/cli/v2"
)

var (
	ethCommand = &cli.Command{
		Name:  "eth",
		Usage: "Eth Commands",
		Subcommands: []*cli.Command{
			&ethNewBlockCommand,
		},
		Flags: []cli.Flag{
			chainFile,
			genesisFile,
		},
	}
	// Flags
	chainFile = &cli.StringFlag{
		Name:  "chain",
		Usage: "Path to chain.rlp file.",
		Value: "chain.rlp",
	}
	genesisFile = &cli.StringFlag{
		Name:  "genesis",
		Usage: "Path to genesis file.",
		Value: "genesis.json",
	}
	// Subcommands
	ethNewBlockCommand = cli.Command{
		Name:   "new-block",
		Usage:  "<node> <msg>",
		Action: ethNewBlock,
	}
)

// ethNewBlock peers with node, sends the provided new block announcement, then disconnects from the peer.
func ethNewBlock(ctx *cli.Context) error {
	// Decode message body.
	msg, err := hex.DecodeString(ctx.Args().Get(1))
	if err != nil {
		return fmt.Errorf("unable to decode msg: %s", err)
	}
	chain, err := ethtest.LoadChain(ctx.String(chainFile.Name), ctx.String(genesisFile.Name))
	if err != nil {
		return fmt.Errorf("error loading chain: %s", err)
	}
	conn, err := ethtest.Dial(getNodeArg(ctx))
	if err != nil {
		return fmt.Errorf("error dialing peer: %s", err)
	}
	// Peer with node.
	status := &ethtest.Status{
		ProtocolVersion: 66,
		NetworkID:       chain.Config().ChainID.Uint64(),
		TD:              chain.TD(),
		Head:            chain.Head().Hash(),
		Genesis:         chain.GetHeaderByNumber(0).Hash(),
		ForkID:          chain.ForkID(),
	}
	if err := conn.Peer(chain, status); err != nil {
		return fmt.Errorf("unable to peer with node: %s", err)
	}
	// Send new block announcement.
	code := uint64((ethtest.NewBlock{}).Code())
	if _, err = conn.Conn.Write(code, msg); err != nil {
		exit(fmt.Errorf("failed to write to connection: %w", err))
	}
	return conn.Close()
}
