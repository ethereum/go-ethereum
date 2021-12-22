package engine_v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/accounts"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"

	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/consensus/clique"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/params"
	lru "github.com/hashicorp/golang-lru"
)

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < utils.ExtraSeal {
		return common.Address{}, utils.ErrMissingSignature
	}
	signature := header.Extra[len(header.Extra)-utils.ExtraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(utils.SigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// XDPoS is the delegated-proof-of-stake consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type XDPoS_v1 struct {
	config *params.XDPoSConfig // Consensus engine configuration parameters
	db     ethdb.Database      // Database to store and retrieve snapshot checkpoints

	recents             *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures          *lru.ARCCache // Signatures of recent blocks to speed up mining
	validatorSignatures *lru.ARCCache // Signatures of recent blocks to speed up mining
	verifiedHeaders     *lru.ARCCache
	proposals           map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address  // Ethereum address of the signing key
	signFn clique.SignerFn // Signer function to authorize hashes with
	lock   sync.RWMutex    // Protects the signer fields

	HookReward            func(chain consensus.ChainReader, state *state.StateDB, parentState *state.StateDB, header *types.Header) (error, map[string]interface{})
	HookPenalty           func(chain consensus.ChainReader, blockNumberEpoc uint64) ([]common.Address, error)
	HookPenaltyTIPSigning func(chain consensus.ChainReader, header *types.Header, candidate []common.Address) ([]common.Address, error)
	HookValidator         func(header *types.Header, signers []common.Address) ([]byte, error)
	HookVerifyMNs         func(header *types.Header, signers []common.Address) error

	HookGetSignersFromContract func(blockHash common.Hash) ([]common.Address, error)
}

// New creates a XDPoS delegated-proof-of-stake consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.XDPoSConfig, db ethdb.Database) *XDPoS_v1 {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = utils.EpochLength
	}

	recents, _ := lru.NewARC(utils.InmemorySnapshots)
	signatures, _ := lru.NewARC(utils.InmemorySnapshots)
	validatorSignatures, _ := lru.NewARC(utils.InmemorySnapshots)
	verifiedHeaders, _ := lru.NewARC(utils.InmemorySnapshots)
	return &XDPoS_v1{
		config: &conf,
		db:     db,

		recents:             recents,
		signatures:          signatures,
		verifiedHeaders:     verifiedHeaders,
		validatorSignatures: validatorSignatures,
		proposals:           make(map[common.Address]bool),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (x *XDPoS_v1) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, x.signatures)
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (x *XDPoS_v1) VerifyHeader(chain consensus.ChainReader, header *types.Header, fullVerify bool) error {
	return x.verifyHeaderWithCache(chain, header, nil, fullVerify)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (x *XDPoS_v1) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, fullVerifies []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := x.verifyHeaderWithCache(chain, header, headers[:i], fullVerifies[i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (x *XDPoS_v1) verifyHeaderWithCache(chain consensus.ChainReader, header *types.Header, parents []*types.Header, fullVerify bool) error {
	_, check := x.verifiedHeaders.Get(header.Hash())
	if check {
		return nil
	}
	err := x.verifyHeader(chain, header, parents, fullVerify)
	if err == nil {
		x.verifiedHeaders.Add(header.Hash(), true)
	}
	return err
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (x *XDPoS_v1) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header, fullVerify bool) error {
	// If we're running a engine faking, accept any block as valid
	if x.config.SkipValidation {
		return nil
	}
	if common.IsTestnet {
		fullVerify = false
	}
	if header.Number == nil {
		return utils.ErrUnknownBlock
	}
	number := header.Number.Uint64()
	if fullVerify {
		if header.Number.Uint64() > x.config.Epoch && len(header.Validator) == 0 {
			return consensus.ErrNoValidatorSignature
		}
		// Don't waste time checking blocks from the future
		if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	// Checkpoint blocks need to enforce zero beneficiary
	checkpoint := (number % x.config.Epoch) == 0
	if checkpoint && header.Coinbase != (common.Address{}) {
		return utils.ErrInvalidCheckpointBeneficiary
	}

	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	if !bytes.Equal(header.Nonce[:], utils.NonceAuthVote) && !bytes.Equal(header.Nonce[:], utils.NonceDropVote) {
		return utils.ErrInvalidVote
	}
	if checkpoint && !bytes.Equal(header.Nonce[:], utils.NonceDropVote) {
		return utils.ErrInvalidCheckpointVote
	}
	// Check that the extra-data contains both the vanity and signature
	if len(header.Extra) < utils.ExtraVanity {
		return utils.ErrMissingVanity
	}
	if len(header.Extra) < utils.ExtraVanity+utils.ExtraSeal {
		return utils.ErrMissingSignature
	}
	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(header.Extra) - utils.ExtraVanity - utils.ExtraSeal
	if !checkpoint && signersBytes != 0 {
		return utils.ErrExtraSigners
	}
	if checkpoint && signersBytes%common.AddressLength != 0 {
		return utils.ErrInvalidCheckpointSigners
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return utils.ErrInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in XDPoS_v1
	if header.UncleHash != utils.UncleHash {
		return utils.ErrInvalidUncleHash
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return x.verifyCascadingFields(chain, header, parents, fullVerify)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (x *XDPoS_v1) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header, fullVerify bool) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to it's parent
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time.Uint64()+x.config.Period > header.Time.Uint64() {
		return utils.ErrInvalidTimestamp
	}

	if number%x.config.Epoch != 0 {
		return x.verifySeal(chain, header, parents, fullVerify)
	}

	/*
		BUG: snapshot returns wrong signers sometimes
		when it happens we get the signers list by requesting smart contract
	*/
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := x.snapshot(chain, number-1, header.ParentHash, parents, nil)
	if err != nil {
		return err
	}

	signers := snap.GetSigners()
	err = x.checkSignersOnCheckpoint(chain, header, signers)
	if err == nil {
		return x.verifySeal(chain, header, parents, fullVerify)
	}

	signers, err = x.getSignersFromContract(chain, header)
	if err != nil {
		return err
	}
	err = x.checkSignersOnCheckpoint(chain, header, signers)
	if err == nil {
		return x.verifySeal(chain, header, parents, fullVerify)
	}

	return err
}

func (x *XDPoS_v1) checkSignersOnCheckpoint(chain consensus.ChainReader, header *types.Header, signers []common.Address) error {
	number := header.Number.Uint64()
	// ignore signerCheck at checkpoint block.
	if common.IgnoreSignerCheckBlockArray[number] {
		return nil
	}
	penPenalties := []common.Address{}
	if x.HookPenalty != nil || x.HookPenaltyTIPSigning != nil {
		var err error
		if chain.Config().IsTIPSigning(header.Number) {
			penPenalties, err = x.HookPenaltyTIPSigning(chain, header, signers)
		} else {
			penPenalties, err = x.HookPenalty(chain, number)
		}
		if err != nil {
			return err
		}
		for _, address := range penPenalties {
			log.Debug("Penalty Info", "address", address, "number", number)
		}
		bytePenalties := common.ExtractAddressToBytes(penPenalties)
		if !bytes.Equal(header.Penalties, bytePenalties) {
			return utils.ErrInvalidCheckpointPenalties
		}
	}
	signers = common.RemoveItemFromArray(signers, penPenalties)
	for i := 1; i <= common.LimitPenaltyEpoch; i++ {
		if number > uint64(i)*x.config.Epoch {
			signers = removePenaltiesFromBlock(chain, signers, number-uint64(i)*x.config.Epoch)
		}
	}
	extraSuffix := len(header.Extra) - utils.ExtraSeal
	masternodesFromCheckpointHeader := common.ExtractAddressFromBytes(header.Extra[utils.ExtraVanity:extraSuffix])
	validSigners := utils.CompareSignersLists(masternodesFromCheckpointHeader, signers)

	if !validSigners {
		log.Error("Masternodes lists are different in checkpoint header and snapshot", "number", number, "masternodes_from_checkpoint_header", masternodesFromCheckpointHeader, "masternodes_in_snapshot", signers, "penList", penPenalties)
		return utils.ErrInvalidCheckpointSigners
	}
	if x.HookVerifyMNs != nil {
		err := x.HookVerifyMNs(header, signers)
		if err != nil {
			return err
		}
	}

	return nil
}

func (x *XDPoS_v1) IsAuthorisedAddress(header *types.Header, chain consensus.ChainReader, address common.Address) bool {
	snap, err := x.GetSnapshot(chain, header)
	if err != nil {
		log.Error("[IsAuthorisedAddress] Can't get snapshot with at ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
		return false
	}
	if _, ok := snap.Signers[address]; ok {
		return true
	}
	return false
}

func (x *XDPoS_v1) GetSnapshot(chain consensus.ChainReader, header *types.Header) (*SnapshotV1, error) {
	number := header.Number.Uint64()
	log.Trace("get snapshot", "number", number, "hash", header.Hash())
	snap, err := x.snapshot(chain, number, header.Hash(), nil, header)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (x *XDPoS_v1) GetAuthorisedSignersFromSnapshot(chain consensus.ChainReader, header *types.Header) ([]common.Address, error) {
	snap, err := x.GetSnapshot(chain, header)
	if err != nil {
		return nil, err
	}
	return snap.GetSigners(), nil
}

func (x *XDPoS_v1) StoreSnapshot(snap *SnapshotV1) error {
	return snap.store(x.db)
}

func position(list []common.Address, x common.Address) int {
	for i, item := range list {
		if item == x {
			return i
		}
	}
	return -1
}

func (x *XDPoS_v1) GetMasternodes(chain consensus.ChainReader, header *types.Header) []common.Address {
	n := header.Number.Uint64()
	e := x.config.Epoch
	switch {
	case n%e == 0:
		return x.GetMasternodesFromCheckpointHeader(header, n, e)
	case n%e != 0:
		h := chain.GetHeaderByNumber(n - (n % e))
		return x.GetMasternodesFromCheckpointHeader(h, n, e)
	default:
		return []common.Address{}
	}
}

func (x *XDPoS_v1) GetPeriod() uint64 { return x.config.Period }

func whoIsCreator(snap *SnapshotV1, header *types.Header) (common.Address, error) {
	if header.Number.Uint64() == 0 {
		return common.Address{}, errors.New("Don't take block 0")
	}
	m, err := ecrecover(header, snap.sigcache)
	if err != nil {
		return common.Address{}, err
	}
	return m, nil
}

func (x *XDPoS_v1) YourTurn(chain consensus.ChainReader, parent *types.Header, signer common.Address) (int, int, int, bool, error) {
	masternodes := x.GetMasternodes(chain, parent)

	// if common.IsTestnet {
	// 	// Only three mns hard code for XDC testnet.
	// 	masternodes = []common.Address{
	// 		common.HexToAddress("0x3Ea0A3555f9B1dE983572BfF6444aeb1899eC58C"),
	// 		common.HexToAddress("0x4F7900282F3d371d585ab1361205B0940aB1789C"),
	// 		common.HexToAddress("0x942a5885A8844Ee5587C8AC5e371Fc39FFE61896"),
	// 	}
	// }

	snap, err := x.GetSnapshot(chain, parent)
	if err != nil {
		log.Warn("Failed when trying to commit new work", "err", err)
		return 0, -1, -1, false, err
	}
	if len(masternodes) == 0 {
		return 0, -1, -1, false, errors.New("Masternodes not found")
	}
	pre := common.Address{}
	// masternode[0] has chance to create block 1
	preIndex := -1
	if parent.Number.Uint64() != 0 {
		pre, err = whoIsCreator(snap, parent)
		if err != nil {
			return 0, 0, 0, false, err
		}
		preIndex = position(masternodes, pre)
	}
	curIndex := position(masternodes, signer)
	if signer == x.signer {
		log.Debug("Masternodes cycle info", "number of masternodes", len(masternodes), "previous", pre, "position", preIndex, "current", signer, "position", curIndex)
	}
	for i, s := range masternodes {
		log.Debug("Masternode:", "index", i, "address", s.String())
	}
	if (preIndex+1)%len(masternodes) == curIndex {
		return len(masternodes), preIndex, curIndex, true, nil
	}
	return len(masternodes), preIndex, curIndex, false, nil
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (x *XDPoS_v1) snapshot(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header, selfHeader *types.Header) (*SnapshotV1, error) {
	// Search for a SnapshotV1 in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *SnapshotV1
	)
	for snap == nil {
		// If an in-memory SnapshotV1 was found, use that
		if s, ok := x.recents.Get(hash); ok {
			snap = s.(*SnapshotV1)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		// checkpoint snapshot = checkpoint - gap
		if (number+x.config.Gap)%x.config.Epoch == 0 {
			if s, err := loadSnapshot(x.config, x.signatures, x.db, hash); err == nil {
				log.Trace("Loaded voting snapshot form disk", "number", number, "hash", hash)
				snap = s
				break
			}
		}
		// If we're at block zero, make a snapshot
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := x.VerifyHeader(chain, genesis, true); err != nil {
				return nil, err
			}
			signers := make([]common.Address, (len(genesis.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[utils.ExtraVanity+i*common.AddressLength:])
			}
			snap = newSnapshot(x.config, x.signatures, 0, genesis.Hash(), signers)
			if err := snap.store(x.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting snapshot to disk")
			break
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else if selfHeader != nil && selfHeader.Hash() == hash {
			// it prevents db doesn't have current block info, can be removed by refactor blockchain.go reorg function call.
			header = selfHeader
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	x.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if (snap.Number+x.config.Gap)%x.config.Epoch == 0 {
		if err = snap.store(x.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (x *XDPoS_v1) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (x *XDPoS_v1) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return x.verifySeal(chain, header, nil, true)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
// verifySeal also checks the pair of creator-validator set in the header satisfies
// the double validation.
func (x *XDPoS_v1) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header, fullVerify bool) error {
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return utils.ErrUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := x.snapshot(chain, number-1, header.ParentHash, parents, nil)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	creator, err := ecrecover(header, x.signatures)
	if err != nil {
		return err
	}
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	difficulty := x.calcDifficulty(chain, parent, creator)
	log.Debug("verify seal block", "number", header.Number, "hash", header.Hash(), "block difficulty", header.Difficulty, "calc difficulty", difficulty, "creator", creator)
	// Ensure that the block's difficulty is meaningful (may not be correct at this point)
	if number > 0 {
		if header.Difficulty.Int64() != difficulty.Int64() {
			return utils.ErrInvalidDifficulty
		}
	}
	masternodes := x.GetMasternodes(chain, header)
	mstring := []string{}
	for _, m := range masternodes {
		mstring = append(mstring, m.String())
	}
	nstring := []string{}
	for _, n := range snap.GetSigners() {
		nstring = append(nstring, n.String())
	}
	if _, ok := snap.Signers[creator]; !ok {
		valid := false
		for _, m := range masternodes {
			if m == creator {
				valid = true
				break
			}
		}
		if !valid {
			log.Debug("Unauthorized creator found", "block number", number, "creator", creator.String(), "masternodes", mstring, "snapshot from parent block", nstring)
			return utils.ErrUnauthorized
		}
	}
	if len(masternodes) > 1 {
		for seen, recent := range snap.Recents {
			if recent == creator {
				// Signer is among recents, only fail if the current block doesn't shift it out
				// There is only case that we don't allow signer to create two continuous blocks.
				if limit := uint64(2); seen > number-limit {
					// Only take into account the non-epoch blocks
					if number%x.config.Epoch != 0 {
						return utils.ErrUnauthorized
					}
				}
			}
		}
	}

	// header must contain validator info following double validation design
	// start checking from epoch 2nd.
	if header.Number.Uint64() > x.config.Epoch && fullVerify {
		validator, err := x.RecoverValidator(header)
		if err != nil {
			return err
		}

		// verify validator
		assignedValidator, err := x.GetValidator(creator, chain, header)
		if err != nil {
			return err
		}
		if validator != assignedValidator {
			log.Debug("Bad block detected. Header contains wrong pair of creator-validator", "creator", creator, "assigned validator", assignedValidator, "wrong validator", validator)
			return utils.ErrFailedDoubleValidation
		}
	}
	return nil
}

func (x *XDPoS_v1) GetValidator(creator common.Address, chain consensus.ChainReader, header *types.Header) (common.Address, error) {
	epoch := x.config.Epoch
	no := header.Number.Uint64()
	cpNo := no
	if no%epoch != 0 {
		cpNo = no - (no % epoch)
	}
	if cpNo == 0 {
		return common.Address{}, nil
	}
	cpHeader := chain.GetHeaderByNumber(cpNo)
	if cpHeader == nil {
		if no%epoch == 0 {
			cpHeader = header
		} else {
			return common.Address{}, fmt.Errorf("couldn't find checkpoint header")
		}
	}
	m, err := GetM1M2FromCheckpointHeader(cpHeader, header, chain.Config())
	if err != nil {
		return common.Address{}, err
	}
	return m[creator], nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (x *XDPoS_v1) Prepare(chain consensus.ChainReader, header *types.Header) error {
	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	// Assemble the voting snapshot to check which votes make sense
	snap, err := x.snapshot(chain, number-1, header.ParentHash, nil, nil)
	if err != nil {
		return err
	}
	if number%x.config.Epoch != 0 {
		x.lock.RLock()

		// Gather all the proposals that make sense voting on
		addresses := make([]common.Address, 0, len(x.proposals))
		for address, authorize := range x.proposals {
			if snap.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		// If there's pending proposals, cast a vote on them
		if len(addresses) > 0 {
			header.Coinbase = addresses[rand.Intn(len(addresses))]
			if x.proposals[header.Coinbase] {
				copy(header.Nonce[:], utils.NonceAuthVote)
			} else {
				copy(header.Nonce[:], utils.NonceDropVote)
			}
		}
		x.lock.RUnlock()
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Set the correct difficulty
	header.Difficulty = x.calcDifficulty(chain, parent, x.signer)
	log.Debug("CalcDifficulty ", "number", header.Number, "difficulty", header.Difficulty)
	// Ensure the extra data has all it's components
	if len(header.Extra) < utils.ExtraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, utils.ExtraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:utils.ExtraVanity]
	masternodes := snap.GetSigners()
	if number >= x.config.Epoch && number%x.config.Epoch == 0 {
		if x.HookPenalty != nil || x.HookPenaltyTIPSigning != nil {
			var penMasternodes []common.Address
			var err error
			if chain.Config().IsTIPSigning(header.Number) {
				penMasternodes, err = x.HookPenaltyTIPSigning(chain, header, masternodes)
			} else {
				penMasternodes, err = x.HookPenalty(chain, number)
			}
			if err != nil {
				return err
			}
			if len(penMasternodes) > 0 {
				// penalize bad masternode(s)
				masternodes = common.RemoveItemFromArray(masternodes, penMasternodes)
				for _, address := range penMasternodes {
					log.Debug("Penalty status", "address", address, "number", number)
				}
				header.Penalties = common.ExtractAddressToBytes(penMasternodes)
			}
		}
		// Prevent penalized masternode(s) within 4 recent epochs
		for i := 1; i <= common.LimitPenaltyEpoch; i++ {
			if number > uint64(i)*x.config.Epoch {
				masternodes = removePenaltiesFromBlock(chain, masternodes, number-uint64(i)*x.config.Epoch)
			}
		}
		for _, masternode := range masternodes {
			header.Extra = append(header.Extra, masternode[:]...)
		}
		if x.HookValidator != nil {
			validators, err := x.HookValidator(header, masternodes)
			if err != nil {
				return err
			}
			header.Validators = validators
		}
	}
	header.Extra = append(header.Extra, make([]byte, utils.ExtraSeal)...)

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay

	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(x.config.Period))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}
	return nil
}

func (x *XDPoS_v1) UpdateMasternodes(chain consensus.ChainReader, header *types.Header, ms []utils.Masternode) error {
	number := header.Number.Uint64()
	log.Trace("take snapshot", "number", number, "hash", header.Hash())
	// get snapshot
	snap, err := x.snapshot(chain, number, header.Hash(), nil, header)
	if err != nil {
		return err
	}
	newMasternodes := make(map[common.Address]struct{})
	for _, m := range ms {
		newMasternodes[m.Address] = struct{}{}
	}
	snap.Signers = newMasternodes
	nm := []string{}
	for _, n := range ms {
		nm = append(nm, n.Address.String())
	}
	x.recents.Add(snap.Hash, snap)
	log.Info("New set of masternodes has been updated to snapshot", "number", snap.Number, "hash", snap.Hash, "new masternodes", nm)
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (x *XDPoS_v1) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, parentState *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// set block reward
	number := header.Number.Uint64()
	rCheckpoint := chain.Config().XDPoS.RewardCheckpoint

	// _ = c.CacheData(header, txs, receipts)

	if x.HookReward != nil && number%rCheckpoint == 0 {
		err, rewards := x.HookReward(chain, state, parentState, header)
		if err != nil {
			return nil, err
		}
		if len(common.StoreRewardFolder) > 0 {
			data, err := json.Marshal(rewards)
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(common.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()), data, 0644)
			}
			if err != nil {
				log.Error("Error when save reward info ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			}
		}
	}

	// the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	// Assemble and return the final block for sealing
	return types.NewBlock(header, txs, nil, receipts), nil
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (x *XDPoS_v1) Authorize(signer common.Address, signFn clique.SignerFn) {
	x.lock.Lock()
	defer x.lock.Unlock()

	x.signer = signer
	x.signFn = signFn
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (x *XDPoS_v1) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return nil, utils.ErrUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	// checkpoint blocks have no tx
	if x.config.Period == 0 && len(block.Transactions()) == 0 && number%x.config.Epoch != 0 {
		return nil, utils.ErrWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	x.lock.RLock()
	signer, signFn := x.signer, x.signFn
	x.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := x.snapshot(chain, number-1, header.ParentHash, nil, nil)
	if err != nil {
		return nil, err
	}
	masternodes := x.GetMasternodes(chain, header)
	if _, authorized := snap.Signers[signer]; !authorized {
		valid := false
		for _, m := range masternodes {
			if m == signer {
				valid = true
				break
			}
		}
		if !valid {
			return nil, utils.ErrUnauthorized
		}
	}
	// If we're amongst the recent signers, wait for the next block
	// only check recent signers if there are more than one signer.
	if len(masternodes) > 1 {
		for seen, recent := range snap.Recents {
			if recent == signer {
				// Signer is among recents, only wait if the current block doesn't shift it out
				// There is only case that we don't allow signer to create two continuous blocks.
				if limit := uint64(2); number < limit || seen > number-limit {
					// Only take into account the non-epoch blocks
					if number%x.config.Epoch != 0 {
						log.Info("Signed recently, must wait for others ", "len(masternodes)", len(masternodes), "number", number, "limit", limit, "seen", seen, "recent", recent.String(), "snap.Recents", snap.Recents)
						<-stop
						return nil, nil
					}
				}
			}
		}
	}
	select {
	case <-stop:
		return nil, nil
	default:
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, utils.SigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(header.Extra[len(header.Extra)-utils.ExtraSeal:], sighash)
	m2, err := x.GetValidator(signer, chain, header)
	if err != nil {
		return nil, fmt.Errorf("can't get block validator: %v", err)
	}
	if m2 == signer {
		header.Validator = sighash
	}
	return block.WithSeal(header), nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (x *XDPoS_v1) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return x.calcDifficulty(chain, parent, x.signer)
}

func (x *XDPoS_v1) calcDifficulty(chain consensus.ChainReader, parent *types.Header, signer common.Address) *big.Int {
	// If we're running a engine faking, skip calculation
	if x.config.SkipValidation {
		return big.NewInt(1)
	}
	len, preIndex, curIndex, _, err := x.YourTurn(chain, parent, signer)
	if err != nil {
		return big.NewInt(int64(len + curIndex - preIndex))
	}
	return big.NewInt(int64(len - utils.Hop(len, preIndex, curIndex)))
}

func (x *XDPoS_v1) RecoverSigner(header *types.Header) (common.Address, error) {
	return ecrecover(header, x.signatures)
}

func (x *XDPoS_v1) RecoverValidator(header *types.Header) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := x.validatorSignatures.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header.Validator
	// len equals 65 bytes
	if len(header.Validator) != utils.ExtraSeal {
		return common.Address{}, consensus.ErrFailValidatorSignature
	}
	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(utils.SigHash(header).Bytes(), header.Validator)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	x.validatorSignatures.Add(hash, signer)
	return signer, nil
}

// Get master nodes over extra data of previous checkpoint block.
func (x *XDPoS_v1) GetMasternodesFromCheckpointHeader(preCheckpointHeader *types.Header, n, e uint64) []common.Address {
	if preCheckpointHeader == nil {
		log.Info("Previous checkpoint's header is empty", "block number", n, "epoch", e)
		return []common.Address{}
	}
	masternodes := make([]common.Address, (len(preCheckpointHeader.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], preCheckpointHeader.Extra[utils.ExtraVanity+i*common.AddressLength:])
	}

	return masternodes
}

func (x *XDPoS_v1) GetDb() ethdb.Database {
	return x.db
}

// Extract validators from byte array.
func removePenaltiesFromBlock(chain consensus.ChainReader, masternodes []common.Address, epochNumber uint64) []common.Address {
	if epochNumber <= 0 {
		return masternodes
	}
	header := chain.GetHeaderByNumber(epochNumber)
	block := chain.GetBlock(header.Hash(), epochNumber)
	penalties := block.Penalties()
	if penalties != nil {
		prevPenalties := common.ExtractAddressFromBytes(penalties)
		masternodes = common.RemoveItemFromArray(masternodes, prevPenalties)
	}
	return masternodes
}

// Get masternodes address from checkpoint Header.
func GetMasternodesFromCheckpointHeader(checkpointHeader *types.Header) []common.Address {
	masternodes := make([]common.Address, (len(checkpointHeader.Extra)-utils.ExtraVanity-utils.ExtraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], checkpointHeader.Extra[utils.ExtraVanity+i*common.AddressLength:])
	}
	return masternodes
}

// Get m2 list from checkpoint block.
func GetM1M2FromCheckpointHeader(checkpointHeader *types.Header, currentHeader *types.Header, config *params.ChainConfig) (map[common.Address]common.Address, error) {
	if checkpointHeader.Number.Uint64()%common.EpocBlockRandomize != 0 {
		return nil, errors.New("This block is not checkpoint block epoc.")
	}
	// Get signers from this block.
	masternodes := GetMasternodesFromCheckpointHeader(checkpointHeader)
	validators := utils.ExtractValidatorsFromBytes(checkpointHeader.Validators)
	m1m2, _, err := utils.GetM1M2(masternodes, validators, currentHeader, config)
	if err != nil {
		return map[common.Address]common.Address{}, err
	}
	return m1m2, nil
}

func (x *XDPoS_v1) getSignersFromContract(chain consensus.ChainReader, checkpointHeader *types.Header) ([]common.Address, error) {
	startGapBlockHeader := checkpointHeader
	number := checkpointHeader.Number.Uint64()
	for step := uint64(1); step <= chain.Config().XDPoS.Gap; step++ {
		startGapBlockHeader = chain.GetHeader(startGapBlockHeader.ParentHash, number-step)
	}
	signers, err := x.HookGetSignersFromContract(startGapBlockHeader.Hash())
	if err != nil {
		return []common.Address{}, fmt.Errorf("Can't get signers from Smart Contract . Err: %v", err)
	}
	return signers, nil
}

func NewFaker(db ethdb.Database, config *params.XDPoSConfig) *XDPoS_v1 {
	var fakeEngine *XDPoS_v1
	// Set any missing consensus parameters to their defaults
	conf := config

	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(utils.InmemorySnapshots)
	signatures, _ := lru.NewARC(utils.InmemorySnapshots)
	validatorSignatures, _ := lru.NewARC(utils.InmemorySnapshots)
	verifiedHeaders, _ := lru.NewARC(utils.InmemorySnapshots)
	fakeEngine = &XDPoS_v1{
		config:              conf,
		db:                  db,
		recents:             recents,
		signatures:          signatures,
		verifiedHeaders:     verifiedHeaders,
		validatorSignatures: validatorSignatures,
		proposals:           make(map[common.Address]bool),
	}
	return fakeEngine
}
