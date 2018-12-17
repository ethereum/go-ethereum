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
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

func init() {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
}

//TestSnapshotCreate is a high level e2e test that tests for snapshot generation
func TestSnapshotCreate(t *testing.T) {

	for _, v := range []struct {
		name string
		args []string
	}{
		{
			name: "no topology - discovery enabled",
			args: []string{
				"c",
			},
		},
		{
			name: "yes topology - discovery disabled",
			args: []string{
				"--topology",
				"ring",
				"c",
			},
		},
	} {
		t.Run(v.name, func(t *testing.T) {
			file, err := ioutil.TempFile("", "swarm-snapshot")
			defer os.Remove(file.Name())
			if err != nil {
				t.Fatal(err)
			}

			file.Close()
			snap := runSnapshot(t, append(v.args, file.Name())...)

			snap.ExpectExit()
			if snap.ExitStatus() != 0 {
				t.Fatal("expected exit code 0")
			}

			_, err = os.Stat(file.Name())
			if err != nil {
				t.Fatal("could not stat snapshot json")
			}
		})
	}
}
