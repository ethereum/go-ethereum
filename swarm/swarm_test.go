// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package swarm

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
)

// TestNewSwarm validates Swarm fields in repsect to the provided configuration.
func TestNewSwarm(t *testing.T) {
	dir, err := ioutil.TempDir("", "swarm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// a simple rpc endpoint for testing dialing
	ipcEndpoint := path.Join(dir, "TestSwarm.ipc")

	// windows namedpipes are not on filesystem but on NPFS
	if runtime.GOOS == "windows" {
		b := make([]byte, 8)
		rand.Read(b)
		ipcEndpoint = `\\.\pipe\TestSwarm-` + hex.EncodeToString(b)
	}

	_, server, err := rpc.StartIPCEndpoint(ipcEndpoint, nil)
	if err != nil {
		t.Error(err)
	}
	defer server.Stop()

	for _, tc := range []struct {
		name      string
		configure func(*api.Config)
		check     func(*testing.T, *Swarm, *api.Config)
	}{
		{
			name:      "defaults",
			configure: nil,
			check: func(t *testing.T, s *Swarm, config *api.Config) {
				if s.config != config {
					t.Error("config is not the same object")
				}
				if s.backend != nil {
					t.Error("backend is not nil")
				}
				if s.privateKey == nil {
					t.Error("private key is not set")
				}
				if !s.config.HiveParams.Discovery {
					t.Error("config.HiveParams.Discovery is false, must be true regardless the configuration")
				}
				if s.dns != nil {
					t.Error("dns initialized, but it should not be")
				}
				if s.lstore == nil {
					t.Error("localstore not initialized")
				}
				if s.streamer == nil {
					t.Error("streamer not initialized")
				}
				if s.fileStore == nil {
					t.Error("fileStore not initialized")
				}
				if s.lstore.Validators == nil {
					t.Error("localstore validators not initialized")
				}
				if s.bzz == nil {
					t.Error("bzz not initialized")
				}
				if s.ps == nil {
					t.Error("pss not initialized")
				}
				if s.api == nil {
					t.Error("api not initialized")
				}
				if s.sfs == nil {
					t.Error("swarm filesystem not initialized")
				}
			},
		},
		{
			name: "with swap",
			configure: func(config *api.Config) {
				config.SwapAPI = ipcEndpoint
				config.SwapEnabled = true
			},
			check: func(t *testing.T, s *Swarm, _ *api.Config) {
				if s.backend == nil {
					t.Error("backend is nil")
				}
			},
		},
		{
			name: "with swap disabled",
			configure: func(config *api.Config) {
				config.SwapAPI = ipcEndpoint
				config.SwapEnabled = false
			},
			check: func(t *testing.T, s *Swarm, _ *api.Config) {
				if s.backend != nil {
					t.Error("backend is not nil")
				}
			},
		},
		{
			name: "with swap enabled and api endpoint blank",
			configure: func(config *api.Config) {
				config.SwapAPI = ""
				config.SwapEnabled = true
			},
			check: func(t *testing.T, s *Swarm, _ *api.Config) {
				if s.backend != nil {
					t.Error("backend is not nil")
				}
			},
		},
		{
			name: "ens",
			configure: func(config *api.Config) {
				config.EnsAPIs = []string{
					"http://127.0.0.1:8888",
				}
			},
			check: func(t *testing.T, s *Swarm, _ *api.Config) {
				if s.dns == nil {
					t.Error("dns is not initialized")
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := api.NewConfig()

			dir, err := ioutil.TempDir("", "swarm")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)

			config.Path = dir

			privkey, err := crypto.GenerateKey()
			if err != nil {
				t.Fatal(err)
			}

			config.Init(privkey)

			if tc.configure != nil {
				tc.configure(config)
			}

			s, err := NewSwarm(config, nil)
			if err != nil {
				t.Fatal(err)
			}

			if tc.check != nil {
				tc.check(t, s, config)
			}
		})
	}
}

func TestParseEnsAPIAddress(t *testing.T) {
	for _, x := range []struct {
		description string
		value       string
		tld         string
		endpoint    string
		addr        common.Address
	}{
		{
			description: "IPC endpoint",
			value:       "/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
		},
		{
			description: "HTTP endpoint",
			value:       "http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
		},
		{
			description: "WS endpoint",
			value:       "ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
		},
		{
			description: "IPC Endpoint and TLD",
			value:       "test:/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			tld:         "test",
		},
		{
			description: "HTTP endpoint and TLD",
			value:       "test:http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			tld:         "test",
		},
		{
			description: "WS endpoint and TLD",
			value:       "test:ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			tld:         "test",
		},
		{
			description: "IPC Endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "HTTP endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "WS endpoint and contract address",
			value:       "314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
		},
		{
			description: "IPC Endpoint, TLD and contract address",
			value:       "test:314159265dD8dbb310642f98f50C066173C1259b@/data/testnet/geth.ipc",
			endpoint:    "/data/testnet/geth.ipc",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "test",
		},
		{
			description: "HTTP endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@http://127.0.0.1:1234",
			endpoint:    "http://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "eth",
		},
		{
			description: "WS endpoint, TLD and contract address",
			value:       "eth:314159265dD8dbb310642f98f50C066173C1259b@ws://127.0.0.1:1234",
			endpoint:    "ws://127.0.0.1:1234",
			addr:        common.HexToAddress("314159265dD8dbb310642f98f50C066173C1259b"),
			tld:         "eth",
		},
	} {
		t.Run(x.description, func(t *testing.T) {
			tld, endpoint, addr := parseEnsAPIAddress(x.value)
			if endpoint != x.endpoint {
				t.Errorf("expected Endpoint %q, got %q", x.endpoint, endpoint)
			}
			if addr != x.addr {
				t.Errorf("expected ContractAddress %q, got %q", x.addr.String(), addr.String())
			}
			if tld != x.tld {
				t.Errorf("expected TLD %q, got %q", x.tld, tld)
			}
		})
	}
}

// TestLocalStoreAndRetrieve runs multiple tests where different size files are uploaded
// to a single Swarm instance using API Store and checked against the content returned
// by API Retrieve function.
//
// This test is intended to validate functionality of chunker store and join functions
// and their intergartion into Swarm, without comparing results with ones produced by
// another chunker implementation, as it is done in swarm/storage tests.
func TestLocalStoreAndRetrieve(t *testing.T) {
	config := api.NewConfig()

	dir, err := ioutil.TempDir("", "node")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	config.Path = dir

	privkey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	config.Init(privkey)

	swarm, err := NewSwarm(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	// by default, test only the lonely chunk cases
	sizes := []int{1, 60, 4097, 524288 + 1, 7*524288 + 1, 128*524288 + 1}

	if *longrunning {
		// test broader set of cases if -longruning flag is set
		sizes = append(sizes, 83, 179, 253, 1024, 4095, 4096, 8191, 8192, 8193, 12287, 12288, 12289, 123456, 2345678, 67298391, 524288, 524288+4096, 524288+4097, 7*524288, 7*524288+4096, 7*524288+4097, 128*524288, 128*524288+4096, 128*524288+4097, 816778334)
	}
	for _, n := range sizes {
		testLocalStoreAndRetrieve(t, swarm, n, true)
		testLocalStoreAndRetrieve(t, swarm, n, false)
	}
}

// testLocalStoreAndRetrieve is using a single Swarm instance, to upload
// a file of length n with optional random data using API Store function,
// and checks the output of API Retrieve function on the same instance.
// This is a regression test for issue
// https://github.com/ethersphere/go-ethereum/issues/639
// where pyramid chunker did not split correctly files with lengths that
// are edge cases for chunk and tree parameters, depending whether there
// is a tree chunk with only one data chunk and how the compress functionality
// changed the tree.
func testLocalStoreAndRetrieve(t *testing.T, swarm *Swarm, n int, randomData bool) {
	slice := make([]byte, n)
	if randomData {
		rand.Seed(time.Now().UnixNano())
		rand.Read(slice)
	}
	dataPut := string(slice)

	ctx := context.TODO()
	k, wait, err := swarm.api.Store(ctx, strings.NewReader(dataPut), int64(len(dataPut)), false)
	if err != nil {
		t.Fatal(err)
	}
	if wait != nil {
		err = wait(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}

	r, _ := swarm.api.Retrieve(context.TODO(), k)

	d, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	dataGet := string(d)

	if len(dataPut) != len(dataGet) {
		t.Fatalf("data not matched: length expected %v, got %v", len(dataPut), len(dataGet))
	} else {
		if dataPut != dataGet {
			t.Fatal("data not matched")
		}
	}
}
