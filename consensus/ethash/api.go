// Copyright 2018 The go-ethereum Authors
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

package ethash

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	common2 "github.com/silesiacoin/bls/common"
	"github.com/silesiacoin/bls/herumi"
	"time"
)

var errEthashStopped = errors.New("ethash stopped")

// API exposes ethash related methods for the RPC interface.
type API struct {
	ethash *Ethash
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//   result[0] - 32 bytes hex encoded current block header pow-hash
//   result[1] - 32 bytes hex encoded seed hash used for DAG
//   result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//   result[3] - hex encoded block number
func (api *API) GetWork() ([4]string, error) {
	if api.ethash.remote == nil {
		return [4]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [4]string, 1)
		errc   = make(chan error, 1)
	)
	select {
	case api.ethash.remote.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.ethash.remote.exitCh:
		return [4]string{}, errEthashStopped
	}
	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [4]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	if api.ethash.remote == nil {
		return false
	}

	var blsSignature *BlsSignatureBytes

	var errc = make(chan error, 1)
	select {
	case api.ethash.remote.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		blsSeal:   blsSignature,
		errc:      errc,
	}:
	case <-api.ethash.remote.exitCh:
		return false
	}
	err := <-errc
	return err == nil
}

// SubmitWorkBLS can be used by external miner to submit their POS solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
// This submit work contains BLS storing feature.
func (api *API) SubmitWorkBLS(nonce types.BlockNonce, hash common.Hash, hexSignatureString string) bool {
	if api.ethash.remote == nil {
		return false
	}

	signatureBytes := hexutil.MustDecode(hexSignatureString)
	blsSignatureBytes := new(BlsSignatureBytes)
	copy(blsSignatureBytes[:], signatureBytes[:])

	var errc = make(chan error, 1)

	select {
	case api.ethash.remote.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: common.BytesToHash(blsSignatureBytes[:32]),
		hash:      hash,
		blsSeal:   blsSignatureBytes,
		errc:      errc,
	}:
	case <-api.ethash.remote.exitCh:
		return false
	}
	err := <-errc
	return err == nil
}

// InsertMinimalConsensusInfo can be used for remote miners to fill MinimalConsensusInfo.
// It accepts the MinimalEpochConsensusInfo that should be calculated on Consensus side
// For now lets pass it as rlp-encoded hex string
// WARN: THIS SOLUTION IS TEMPORARY. THIS MUST BE SECURED. ONLY TRUSTED CONSENSUS NODE SHOULD BE ABLE TO PERFORM THIS.
func (api *API) InsertMinimalConsensusInfo(
	epoch uint64,
	validatorsList []string,
	epochTimeStartUnix uint64,
) bool {
	// Works only in pandora mode
	ethash := api.ethash
	consensusInfo := NewMinimalConsensusInfo(epoch).(*MinimalEpochConsensusInfo)
	consensusInfo.EpochTimeStartUnix = epochTimeStartUnix
	consensusInfo.EpochTimeStart = time.Unix(int64(epochTimeStartUnix), 0)
	consensusInfo.ValidatorsList = [validatorListLen]common2.PublicKey{}

	// Invalid payload
	if len(validatorsList) != validatorListLen {
		ethash.config.Log.Error(
			"Invalid validators list for epoch",
			"epoch",
			epoch,
			"validatorLen",
			len(validatorsList),
		)
		return false
	}

	for index, validator := range validatorsList {
		// For genesis slot 0 there is no validator, so we should just simply insert something
		genesisCheck := index == 0 && epoch == 0

		if genesisCheck {
			secretKey, _ := herumi.RandKey()
			consensusInfo.ValidatorsList[index] = secretKey.PublicKey()

			continue
		}

		pubKey, err := herumi.PublicKeyFromBytes(hexutil.MustDecode(validator))

		if nil != err {
			ethash.config.Log.Error(
				"Could not cast public key from bytes",
				"epoch",
				epoch,
				"index",
				index,
				"validator",
				validator,
				"err",
				err.Error(),
			)
			return false
		}

		consensusInfo.ValidatorsList[index] = pubKey
	}

	ethash.config.Log.Info(
		"Inserting minimal consensus info for epoch",
		"epoch",
		epoch,
		"timeStartUnix",
		consensusInfo.EpochTimeStartUnix,
	)

	err := ethash.InsertMinimalConsensusInfo(epoch, consensusInfo)

	if nil != err {
		ethash.config.Log.Error(
			"Could not insert minimal consensus info",
			"epoch",
			epoch,
			"err",
			err.Error(),
		)
	}

	return nil == err
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashRate(rate hexutil.Uint64, id common.Hash) bool {
	if api.ethash.remote == nil {
		return false
	}

	var done = make(chan struct{}, 1)
	select {
	case api.ethash.remote.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.ethash.remote.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done
	return true
}

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) GetHashrate() uint64 {
	return uint64(api.ethash.Hashrate())
}
