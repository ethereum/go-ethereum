// Copyright 2023 The go-ethereum Authors
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
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/hexutil"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/miner"
)

// MinerAPI provides an API to control the miner.
type MinerAPI struct {
	e     *Ethereum
	agent *miner.RemoteAgent
}

// NewMinerAPI create a new RPC service which controls the miner of this node.
func NewMinerAPI(e *Ethereum) *MinerAPI {
	agent := miner.NewRemoteAgent(e.BlockChain(), e.Engine())
	e.Miner().Register(agent)
	return &MinerAPI{e, agent}
}

// Start the miner with the given number of threads. If threads is nil the number
// of workers started is equal to the number of logical CPUs that are usable by
// this process. If mining is already running, this method adjust the number of
// threads allowed to use.
func (api *MinerAPI) Start(threads *int) error {
	// Set the number of threads if the seal engine supports it
	if threads == nil {
		threads = new(int)
	} else if *threads == 0 {
		*threads = -1 // Disable the miner from within
	}
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", *threads)
		th.SetThreads(*threads)
	}
	// Start the miner and return
	if !api.e.IsStaking() {
		// Propagate the initial price point to the transaction pool
		api.e.lock.RLock()
		// api.e.gasPrice is from MinerGasPriceFlag
		price := api.e.gasPrice
		api.e.lock.RUnlock()

		api.e.txPool.SetGasTip(price)
		return api.e.StartStaking(true)
	}
	return nil
}

// Stop the miner
func (api *MinerAPI) Stop() bool {
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := api.e.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	api.e.StopStaking()
	return true
}

// SetExtra sets the extra data string that is included when this miner mines a block.
func (api *MinerAPI) SetExtra(extra string) (bool, error) {
	if err := api.e.Miner().SetExtra([]byte(extra)); err != nil {
		return false, err
	}
	return true, nil
}

// SetGasPrice sets the minimum accepted gas price for the miner.
func (api *MinerAPI) SetGasPrice(gasPrice hexutil.Big) bool {
	tip := (*big.Int)(&gasPrice)
	if err := api.e.txPool.SetGasTip(tip); err != nil {
		return false
	}
	if err := api.e.Miner().SetGasTip(tip); err != nil {
		return false
	}

	api.e.lock.Lock()
	api.e.gasPrice = new(big.Int).Set(tip)
	api.e.lock.Unlock()
	return true
}

// SetEtherbase sets the etherbase of the miner
func (api *MinerAPI) SetEtherbase(etherbase common.Address) bool {
	log.Info("[MinerAPI] SetEtherbase", "addr", etherbase)
	api.e.SetEtherbase(etherbase)
	return true
}

// GetHashrate returns the current hashrate of the miner.
func (api *MinerAPI) GetHashrate() uint64 {
	return uint64(api.e.miner.HashRate())
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (api *MinerAPI) SubmitWork(nonce types.BlockNonce, solution, digest common.Hash) bool {
	return api.agent.SubmitWork(nonce, digest, solution)
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (api *MinerAPI) GetWork() ([3]string, error) {
	if !api.e.IsStaking() {
		if err := api.e.StartStaking(false); err != nil {
			return [3]string{}, err
		}
	}
	work, err := api.agent.GetWork()
	if err != nil {
		return work, fmt.Errorf("mining not ready: %v", err)
	}
	return work, nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate. This enables the node to report the combined
// hash rate of all miners which submit work through this node. It accepts the miner hash rate and an identifier which
// must be unique between nodes.
func (api *MinerAPI) SubmitHashrate(hashrate hexutil.Uint64, id common.Hash) bool {
	api.agent.SubmitHashrate(id, uint64(hashrate))
	return true
}
