// Copyright 2017 The go-ethereum Authors
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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm"
)

func TestCLISwarmExport(t *testing.T) {
	cluster := newTestCluster(t, 1)
	defer cluster.Cleanup()

	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer tmp.Close()
	defer os.Remove(tmp.Name())

	// write 10mb random data to file
	buf := make([]byte, 10000000)
	_, err = rand.Read(buf)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(tmp.Name(), buf, 0755)

	// upload the file with 'swarm up' and expect a hash
	log.Info("uploading file with 'swarm up'")
	up := runSwarm(t, "--bzzapi", cluster.Nodes[0].URL, "up", tmp.Name())
	_, matches := up.ExpectRegexp(`[a-f\d]{64}`)
	up.ExpectExit()
	hash := matches[0]
	log.Info("file uploaded", "hash", hash)

	var info swarm.Info
	if err := cluster.Nodes[0].Client.Call(&info, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	spew.Dump(info)

	spew.Dump(cluster.TmpDir)
	spew.Dump(cluster.Nodes[0].Name)
	spew.Dump(cluster.Nodes[0].Addr)
	spew.Dump(cluster.Nodes[0].URL)
	spew.Dump(cluster.Nodes[0].Enode)
	spew.Dump(cluster.Nodes[0].Dir)

	cluster.Stop()
	defer cluster.Cleanup()

	file, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	defer os.Remove(file.Name())

	fmt.Println("==================================")

	spew.Dump(info.BzzKey[2:])
	up2 := runSwarm(t, "db", "export", info.Path+"/chunks", file.Name(), info.BzzKey[2:])
	up2.ExpectExit()

	cluster2 := newTestCluster(t, 1)

	var info2 swarm.Info
	if err := cluster2.Nodes[0].Client.Call(&info2, "bzz_info"); err != nil {
		t.Fatal(err)
	}

	cluster2.Stop()
	defer cluster2.Cleanup()

	spew.Dump(info2.BzzKey[2:])
	up3 := runSwarm(t, "db", "import", info2.Path+"/chunks", file.Name(), info2.BzzKey[2:])
	up3.ExpectExit()

	cluster2.StartExistingNodes(t, 1, info2.BzzAccount[2:])

	res, err := http.Get(cluster2.Nodes[0].URL + "/bzz:/" + hash)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected HTTP status %d, got %s", 200, res.Status)
	}
}
