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

package bor

import (
	"bytes"
	"encoding/json"

	lru "github.com/hashicorp/golang-lru"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/internal/ethapi"
	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/params"
)

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.BorConfig // Consensus engine parameters to fine tune behavior
	ethAPI   *ethapi.PublicBlockChainAPI
	sigcache *lru.ARCCache // Cache of recent block signatures to speed up ecrecover

	Number       uint64                    `json:"number"`       // Block number where the snapshot was created
	Hash         common.Hash               `json:"hash"`         // Block hash where the snapshot was created
	ValidatorSet *ValidatorSet             `json:"validatorSet"` // Validator set at this moment
	Recents      map[uint64]common.Address `json:"recents"`      // Set of recent signers for spam protections
}

// signersAscending implements the sort interface to allow sorting a list of addresses
type signersAscending []common.Address

func (s signersAscending) Len() int           { return len(s) }
func (s signersAscending) Less(i, j int) bool { return bytes.Compare(s[i][:], s[j][:]) < 0 }
func (s signersAscending) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(
	config *params.BorConfig,
	sigcache *lru.ARCCache,
	number uint64,
	hash common.Hash,
	validators []*Validator,
	ethAPI *ethapi.PublicBlockChainAPI,
) *Snapshot {
	snap := &Snapshot{
		config:       config,
		ethAPI:       ethAPI,
		sigcache:     sigcache,
		Number:       number,
		Hash:         hash,
		ValidatorSet: NewValidatorSet(validators),
		Recents:      make(map[uint64]common.Address),
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.BorConfig, sigcache *lru.ARCCache, db ethdb.Database, hash common.Hash, ethAPI *ethapi.PublicBlockChainAPI) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("bor-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache
	snap.ethAPI = ethAPI

	// update total voting power
	if err := snap.ValidatorSet.updateTotalVotingPower(); err != nil {
		return nil, err
	}

	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db ethdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("bor-"), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:       s.config,
		ethAPI:       s.ethAPI,
		sigcache:     s.sigcache,
		Number:       s.Number,
		Hash:         s.Hash,
		ValidatorSet: s.ValidatorSet.Copy(),
		Recents:      make(map[uint64]common.Address),
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}

	return cpy
}

func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errOutOfRangeChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errOutOfRangeChain
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()

	for _, header := range headers {
		// Remove any votes on checkpoint blocks
		number := header.Number.Uint64()

		// Delete the oldest signer from the recent list to allow it signing again
		if number >= s.config.Sprint && number-s.config.Sprint >= 0 {
			delete(snap.Recents, number-s.config.Sprint)
		}

		// Resolve the authorization key and check against signers
		signer, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}

		// check if signer is in validator set
		if !snap.ValidatorSet.HasAddress(signer.Bytes()) {
			return nil, errUnauthorizedSigner
		}

		if _, err = snap.GetSignerSuccessionNumber(signer); err != nil {
			return nil, err
		}

		// add recents
		snap.Recents[number] = signer

		// change validator set and change proposer
		if number > 0 && (number+1)%s.config.Sprint == 0 {
			if err := validateHeaderExtraField(header.Extra); err != nil {
				return nil, err
			}
			validatorBytes := header.Extra[extraVanity : len(header.Extra)-extraSeal]

			// get validators from headers and use that for new validator set
			newVals, _ := ParseValidators(validatorBytes)
			v := getUpdatedValidatorSet(snap.ValidatorSet.Copy(), newVals)
			v.IncrementProposerPriority(1)
			snap.ValidatorSet = v
		}
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// GetSignerSuccessionNumber returns the relative position of signer in terms of the in-turn proposer
func (s *Snapshot) GetSignerSuccessionNumber(signer common.Address) (int, error) {
	validators := s.ValidatorSet.Validators
	proposer := s.ValidatorSet.GetProposer().Address
	proposerIndex, _ := s.ValidatorSet.GetByAddress(proposer)
	if proposerIndex == -1 {
		return -1, &ProposerNotFoundError{proposer}
	}
	signerIndex, _ := s.ValidatorSet.GetByAddress(signer)
	if signerIndex == -1 {
		return -1, &SignerNotFoundError{signer}
	}
	limit := len(validators)/2 + 1

	tempIndex := signerIndex
	if proposerIndex != tempIndex && limit > 0 {
		if tempIndex < proposerIndex {
			tempIndex = tempIndex + len(validators)
		}

		if tempIndex-proposerIndex > limit {
			log.Info("errRecentlySigned", "proposerIndex", validators[proposerIndex].Address.Hex(), "signerIndex", validators[signerIndex].Address.Hex())
			return -1, errRecentlySigned
		}
	}
	return tempIndex - proposerIndex, nil
}

// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) signers() []common.Address {
	sigs := make([]common.Address, 0, len(s.ValidatorSet.Validators))
	for _, sig := range s.ValidatorSet.Validators {
		sigs = append(sigs, sig.Address)
	}
	return sigs
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(number uint64, signer common.Address, epoch uint64) uint64 {
	// if signer is empty
	if bytes.Compare(signer.Bytes(), common.Address{}.Bytes()) == 0 {
		return 1
	}

	validators := s.ValidatorSet.Validators
	proposer := s.ValidatorSet.GetProposer().Address
	totalValidators := len(validators)

	proposerIndex, _ := s.ValidatorSet.GetByAddress(proposer)
	signerIndex, _ := s.ValidatorSet.GetByAddress(signer)

	// temp index
	tempIndex := signerIndex
	if tempIndex < proposerIndex {
		tempIndex = tempIndex + totalValidators
	}

	return uint64(totalValidators - (tempIndex - proposerIndex))
}
