// Copyright 2016 The go-ethereum Authors
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

package api

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ubiq/go-ubiq/common"
	"github.com/ubiq/go-ubiq/crypto"
)

var (
	hexprvkey     = "65138b2aa745041b372153550584587da326ab440576b2a1191dd95cee30039c"
	defaultConfig = `{
    "ChunkDbPath": "` + filepath.Join("TMPDIR", "chunks") + `",
    "DbCapacity": 5000000,
    "CacheCapacity": 5000,
    "Radius": 0,
    "Branches": 128,
    "Hash": "SHA3",
    "CallInterval": 3000000000,
    "KadDbPath": "` + filepath.Join("TMPDIR", "bzz-peers.json") + `",
    "MaxProx": 8,
    "ProxBinSize": 2,
    "BucketSize": 4,
    "PurgeInterval": 151200000000000,
    "InitialRetryInterval": 42000000,
    "MaxIdleInterval": 42000000000,
    "ConnRetryExp": 2,
    "Swap": {
        "BuyAt": 20000000000,
        "SellAt": 20000000000,
        "PayAt": 100,
        "DropAt": 10000,
        "AutoCashInterval": 300000000000,
        "AutoCashThreshold": 50000000000000,
        "AutoDepositInterval": 300000000000,
        "AutoDepositThreshold": 50000000000000,
        "AutoDepositBuffer": 100000000000000,
        "PublicKey": "0x045f5cfd26692e48d0017d380349bcf50982488bc11b5145f3ddf88b24924299048450542d43527fbe29a5cb32f38d62755393ac002e6bfdd71b8d7ba725ecd7a3",
        "Contract": "0x0000000000000000000000000000000000000000",
        "Beneficiary": "0x0d2f62485607cf38d9d795d93682a517661e513e"
    },
    "RequestDbPath": "` + filepath.Join("TMPDIR", "requests") + `",
    "RequestDbBatchSize": 512,
    "KeyBufferSize": 1024,
    "SyncBatchSize": 128,
    "SyncBufferSize": 128,
    "SyncCacheSize": 1024,
    "SyncPriorities": [
        2,
        1,
        1,
        0,
        0
    ],
    "SyncModes": [
        true,
        true,
        true,
        true,
        false
    ],
    "Path": "TMPDIR",
    "ListenAddr": "127.0.0.1",
    "Port": "8500",
    "PublicKey": "0x045f5cfd26692e48d0017d380349bcf50982488bc11b5145f3ddf88b24924299048450542d43527fbe29a5cb32f38d62755393ac002e6bfdd71b8d7ba725ecd7a3",
    "BzzKey": "0xe861964402c0b78e2d44098329b8545726f215afa737d803714a4338552fcb81",
    "EnsRoot": "0x112234455c3a32fd11230c42e7bccd4a84e02010",
    "NetworkId": 323
}`
)

func TestConfigWriteRead(t *testing.T) {
	tmp, err := ioutil.TempDir(os.TempDir(), "bzz-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	prvkey, err := crypto.HexToECDSA(hexprvkey)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}
	orig, err := NewConfig(tmp, common.Address{}, prvkey, 323)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	data, err := ioutil.ReadFile(filepath.Join(orig.Path, "config.json"))
	if err != nil {
		t.Fatalf("default config file cannot be read: %v", err)
	}
	exp := strings.Replace(defaultConfig, "TMPDIR", orig.Path, -1)
	exp = strings.Replace(exp, "\\", "\\\\", -1)
	if string(data) != exp {
		t.Fatalf("default config mismatch:\nexpected: %v\ngot: %v", exp, string(data))
	}

	conf, err := NewConfig(tmp, common.Address{}, prvkey, 323)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if conf.Swap.Beneficiary.Hex() != orig.Swap.Beneficiary.Hex() {
		t.Fatalf("expected beneficiary from loaded config %v to match original %v", conf.Swap.Beneficiary.Hex(), orig.Swap.Beneficiary.Hex())
	}

}
