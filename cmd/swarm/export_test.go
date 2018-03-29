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
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/swarm"
)

// TestCLISwarmExportImport perform the following test:
// 1. runs swarm node
// 2. uploads a random file
// 3. runs an export of the local datastore
// 4. runs a second swarm node
// 5. imports the exported datastore
// 6. fetches the uploaded random file from the second node
func TestCLISwarmExportImport(t *testing.T) {
	cluster := newTestCluster(t, 1)

	// generate random 10mb file
	f, cleanup := generateRandomFile(t, 10000000)
	defer cleanup()

	// upload the file with 'swarm up' and expect a hash
	up := runSwarm(t, "--bzzapi", cluster.Nodes[0].URL, "up", f.Name())
	_, matches := up.ExpectRegexp(`[a-f\d]{64}`)
	up.ExpectExit()
	hash := matches[0]

	var info swarm.Info
	if err := cluster.Nodes[0].Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	cluster.Stop()
	defer cluster.Cleanup()

	// generate an export.tar
	exportCmd := runSwarm(t, "db", "export", info.Path+"/chunks", info.Path+"/export.tar", info.BzzKey[2:])
	exportCmd.ExpectExit()

	// start second cluster
	cluster2 := newTestCluster(t, 1)

	var info2 swarm.Info
	if err := cluster2.Nodes[0].Client.Call(&info2, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	// stop second cluster, so that we close LevelDB
	cluster2.Stop()
	defer cluster2.Cleanup()

	// import the export.tar
	importCmd := runSwarm(t, "db", "import", info2.Path+"/chunks", info.Path+"/export.tar", info2.BzzKey[2:])
	importCmd.ExpectExit()

	// spin second cluster back up
	cluster2.StartExistingNodes(t, 1, info2.BzzAccount[2:])

	// try to fetch imported file
	res, err := http.Get(cluster2.Nodes[0].URL + "/bzz:/" + hash)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected HTTP status %d, got %s", 200, res.Status)
	}

	//TODO: compare res with generated random file
}

func generateRandomFile(t *testing.T, size int) (f *os.File, teardown func()) {
	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}

	// callback for tmp file cleanup
	teardown = func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}

	// write 10mb random data to file
	buf := make([]byte, 10000000)
	_, err = rand.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	ioutil.WriteFile(tmp.Name(), buf, 0755)

	return tmp, teardown
}
