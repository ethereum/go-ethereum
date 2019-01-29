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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/simulations"
)

// TestSnapshotCreate is a high level e2e test that tests for snapshot generation.
// It runs a few "create" commands with different flag values and loads generated
// snapshot files to validate their content.
func TestSnapshotCreate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}

	for _, v := range []struct {
		name     string
		nodes    int
		services string
	}{
		{
			name: "defaults",
		},
		{
			name:  "more nodes",
			nodes: defaultNodes + 5,
		},
		{
			name:     "services",
			services: "stream,pss,zorglub",
		},
		{
			name:     "services with bzz",
			services: "bzz,pss",
		},
	} {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			file, err := ioutil.TempFile("", "swarm-snapshot")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(file.Name())

			if err = file.Close(); err != nil {
				t.Error(err)
			}

			args := []string{"create"}
			if v.nodes > 0 {
				args = append(args, "--nodes", strconv.Itoa(v.nodes))
			}
			if v.services != "" {
				args = append(args, "--services", v.services)
			}
			testCmd := runSnapshot(t, append(args, file.Name())...)

			testCmd.ExpectExit()
			if code := testCmd.ExitStatus(); code != 0 {
				t.Fatalf("command exit code %v, expected 0", code)
			}

			f, err := os.Open(file.Name())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				err := f.Close()
				if err != nil {
					t.Error("closing snapshot file", "err", err)
				}
			}()

			b, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}
			var snap simulations.Snapshot
			err = json.Unmarshal(b, &snap)
			if err != nil {
				t.Fatal(err)
			}

			wantNodes := v.nodes
			if wantNodes == 0 {
				wantNodes = defaultNodes
			}
			gotNodes := len(snap.Nodes)
			if gotNodes != wantNodes {
				t.Errorf("got %v nodes, want %v", gotNodes, wantNodes)
			}

			if len(snap.Conns) == 0 {
				t.Error("no connections in a snapshot")
			}

			var wantServices []string
			if v.services != "" {
				wantServices = strings.Split(v.services, ",")
			} else {
				wantServices = []string{"bzz"}
			}
			// sort service names so they can be comparable
			// as strings to every node sorted services
			sort.Strings(wantServices)

			for i, n := range snap.Nodes {
				gotServices := n.Node.Config.Services
				sort.Strings(gotServices)
				if fmt.Sprint(gotServices) != fmt.Sprint(wantServices) {
					t.Errorf("got services %v for node %v, want %v", gotServices, i, wantServices)
				}
			}

		})
	}
}
