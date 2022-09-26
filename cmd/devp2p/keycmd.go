// Copyright 2020 The go-ethereum Authors
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
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/urfave/cli/v2"
)

var (
	keyCommand = &cli.Command{
		Name:  "key",
		Usage: "Operations on node keys",
		Subcommands: []*cli.Command{
			keyGenerateCommand,
			keyToNodeCommand,
		},
	}
	keyGenerateCommand = &cli.Command{
		Name:      "generate",
		Usage:     "Generates node key files",
		ArgsUsage: "keyfile",
		Action:    genkey,
	}
	keyToNodeCommand = &cli.Command{
		Name:      "to-enode",
		Usage:     "Creates an enode URL from a node key file",
		ArgsUsage: "keyfile",
		Action:    keyToURL,
		Flags:     []cli.Flag{hostFlag, tcpPortFlag, udpPortFlag},
	}
)

var (
	hostFlag = &cli.StringFlag{
		Name:  "ip",
		Usage: "IP address of the node",
		Value: "127.0.0.1",
	}
	tcpPortFlag = &cli.IntFlag{
		Name:  "tcp",
		Usage: "TCP port of the node",
		Value: 30303,
	}
	udpPortFlag = &cli.IntFlag{
		Name:  "udp",
		Usage: "UDP port of the node",
		Value: 30303,
	}
)

func genkey(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("need key file as argument")
	}
	file := ctx.Args().Get(0)

	key, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("could not generate key: %v", err)
	}
	return crypto.SaveECDSA(file, key)
}

func keyToURL(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("need key file as argument")
	}

	var (
		file = ctx.Args().Get(0)
		host = ctx.String(hostFlag.Name)
		tcp  = ctx.Int(tcpPortFlag.Name)
		udp  = ctx.Int(udpPortFlag.Name)
	)
	key, err := crypto.LoadECDSA(file)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("invalid IP address %q", host)
	}
	node := enode.NewV4(&key.PublicKey, ip, tcp, udp)
	fmt.Println(node.URLv4())
	return nil
}
