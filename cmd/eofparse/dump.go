// Copyright 2024 The go-ethereum Authors
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
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/urfave/cli/v2"
)

var (
	dumpCommand = &cli.Command{
		Name:   "eofdump",
		Usage:  "Parses hex eof container and prints out human-readable representation of the container.",
		Action: dumpAction,
		Flags: []cli.Flag{
			hexFlag,
		},
	}
)

func dumpAction(ctx *cli.Context) error {
	// If `--hex` is set, parse and validate the hex string argument.
	if ctx.IsSet(hexFlag.Name) {
		return dump(ctx.String(hexFlag.Name))
	}
	// Otherwise read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}
		if err := dump(l); err != nil {
			return err
		}
		fmt.Println("")
	}
	return scanner.Err()
}

func dump(hexdata string) error {
	if len(hexdata) >= 2 && strings.HasPrefix(hexdata, "0x") {
		hexdata = hexdata[2:]
	}
	b, err := hex.DecodeString(hexdata)
	if err != nil {
		return fmt.Errorf("unable to decode data: %w", err)
	}
	var c vm.Container
	if err := c.UnmarshalBinary(b, false); err != nil {
		return err
	}
	fmt.Println(c.String())
	return nil
}
