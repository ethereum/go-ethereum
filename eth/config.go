// Copyright 2014 The go-ethereum Authors
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

package eth

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
)

var DefaultConfig = Config{
	EthashCachesInMem:    2,
	EthashCachesOnDisk:   3,
	EthashDatasetsInMem:  1,
	EthashDatasetsOnDisk: 2,
	NetworkId:            1,
	LightPeers:           20,
	DatabaseCache:        128,
	GasPrice:             big.NewInt(20 * params.Shannon),
	GpoBlocks:            10,
	GpoPercentile:        50,
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "windows" {
		DefaultConfig.EthashDatasetDir = filepath.Join(home, "AppData", "Ethash")
	} else {
		DefaultConfig.EthashDatasetDir = filepath.Join(home, ".ethash")
	}
}

//go:generate gencodec -type Config -field-override configMarshaling -formats toml -out gen_config.go

type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Ethereum main net block is used.
	Genesis              *core.Genesis `toml:",omitempty"`
	NetworkId            int           // Network ID to use for selecting peers to connect to
	FastSync             bool          // Enables the state download based fast synchronisation algorithm
	LightMode            bool          // Running in light client mode
	LightServ            int           // Maximum percentage of time allowed for serving LES requests
	LightPeers           int           // Maximum number of LES client peers
	MaxPeers             int           // Maximum number of global peers
	SkipBcVersionCheck   bool          `toml:",omitempty"` // e.g. blockchain export
	DatabaseCache        int
	DatabaseHandles      int    `toml:"-"`
	DocRoot              string `toml:",omitempty"`
	PowFake              bool   `toml:",omitempty"`
	PowTest              bool   `toml:",omitempty"`
	PowShared            bool   `toml:",omitempty"`
	ExtraData            []byte
	EthashCacheDir       string `toml:",omitempty"`
	EthashCachesInMem    int
	EthashCachesOnDisk   int
	EthashDatasetDir     string `toml:",omitempty"`
	EthashDatasetsInMem  int
	EthashDatasetsOnDisk int
	Etherbase            common.Address `toml:",omitempty"`
	GasPrice             *big.Int
	MinerThreads         int    `toml:",omitempty"`
	SolcPath             string `toml:",omitempty"`

	GpoBlocks     int
	GpoPercentile int

	EnablePreimageRecording bool
}

type configMarshaling struct {
	ExtraData hexutil.Bytes
}
