package types

import (
	"bytes"
	"errors"
	"fmt"

	"encoding/binary"
	"github.com/pavelkrolevets/go-ethereum/common"
	"github.com/pavelkrolevets/go-ethereum/crypto/sha3"
	"github.com/pavelkrolevets/go-ethereum/ethdb"
	"github.com/pavelkrolevets/go-ethereum/rlp"
	"github.com/pavelkrolevets/go-ethereum/trie"
)

type LCPContext struct {
	epochTrie     *trie.Trie
	delegateTrie  *trie.Trie
	voteTrie      *trie.Trie
	candidateTrie *trie.Trie
	mintCntTrie   *trie.Trie
	periodBlock   *trie.Trie
	maxValidators *trie.Trie
	epochInterval *trie.Trie

	db ethdb.Database
}

var (
	epochPrefix         = []byte("epoch-")
	delegatePrefix      = []byte("delegate-")
	votePrefix          = []byte("vote-")
	candidatePrefix     = []byte("candidate-")
	mintCntPrefix       = []byte("mintCnt-")
	periodBlockPrefix   = []byte("Period-")
	maxValidatorsPrefix = []byte("MaxValidator-")
	epochIntervalPrefix = []byte("EpochInterval-")
)

func NewEpochTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, epochPrefix, dbd)
}

func NewDelegateTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, delegatePrefix, dbd)
}

func NewVoteTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, votePrefix, dbd)
}

func NewCandidateTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, candidatePrefix, dbd)
}

func NewMintCntTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, mintCntPrefix, dbd)
}

func NewPeriodBlockTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, periodBlockPrefix, dbd)
}
func NewMaxValidatorsTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, maxValidatorsPrefix, dbd)
}
func NewEpochIntervalTrie(root common.Hash, db ethdb.Database) (*trie.Trie, error) {
	dbd := trie.NewDatabase(db)
	return trie.NewTrieWithPrefix(root, epochIntervalPrefix, dbd)
}

func NewLCPContext(db ethdb.Database) (*LCPContext, error) {
	epochTrie, err := NewEpochTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}

	periodTrie, err := NewPeriodBlockTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	maxValidatorTrie, err := NewMaxValidatorsTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	epochIntervalTrie, err := NewEpochIntervalTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &LCPContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		periodBlock:   periodTrie,
		maxValidators: maxValidatorTrie,
		epochInterval: epochIntervalTrie,
		db:            db,
	}, nil
}

func NewLCPContextFromProto(db ethdb.Database, ctxProto *LCPContextProto) (*LCPContext, error) {
	epochTrie, err := NewEpochTrie(ctxProto.EpochHash, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(ctxProto.DelegateHash, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(ctxProto.VoteHash, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(ctxProto.CandidateHash, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(ctxProto.MintCntHash, db)
	if err != nil {
		return nil, err
	}
	periodTrie, err := NewPeriodBlockTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	maxValidatorTrie, err := NewMaxValidatorsTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	epochIntervalTrie, err := NewEpochIntervalTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &LCPContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		periodBlock:   periodTrie,
		maxValidators: maxValidatorTrie,
		epochInterval: epochIntervalTrie,
		db:            db,
	}, nil
}

func (d *LCPContext) Copy() *LCPContext {
	epochTrie := *d.epochTrie
	delegateTrie := *d.delegateTrie
	voteTrie := *d.voteTrie
	candidateTrie := *d.candidateTrie
	mintCntTrie := *d.mintCntTrie
	periodTrie := *d.periodBlock
	maxValidatorTrie := *d.maxValidators
	epochIntervalTrie := *d.epochInterval
	return &LCPContext{
		epochTrie:     &epochTrie,
		delegateTrie:  &delegateTrie,
		voteTrie:      &voteTrie,
		candidateTrie: &candidateTrie,
		mintCntTrie:   &mintCntTrie,
		periodBlock:   &periodTrie,
		maxValidators: &maxValidatorTrie,
		epochInterval: &epochIntervalTrie,
	}
}

func (d *LCPContext) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, d.epochTrie.Hash())
	rlp.Encode(hw, d.delegateTrie.Hash())
	rlp.Encode(hw, d.candidateTrie.Hash())
	rlp.Encode(hw, d.voteTrie.Hash())
	rlp.Encode(hw, d.mintCntTrie.Hash())
	rlp.Encode(hw, d.periodBlock.Hash())
	rlp.Encode(hw, d.maxValidators.Hash())
	rlp.Encode(hw, d.epochInterval.Hash())
	hw.Sum(h[:0])
	return h
}

func (d *LCPContext) Snapshot() *LCPContext {
	return d.Copy()
}

func (d *LCPContext) RevertToSnapShot(snapshot *LCPContext) {
	d.epochTrie = snapshot.epochTrie
	d.delegateTrie = snapshot.delegateTrie
	d.candidateTrie = snapshot.candidateTrie
	d.voteTrie = snapshot.voteTrie
	d.mintCntTrie = snapshot.mintCntTrie
	d.periodBlock = snapshot.periodBlock
	d.maxValidators = snapshot.maxValidators
	d.epochInterval = snapshot.epochInterval
}

func (d *LCPContext) FromProto(dcp *LCPContextProto) error {
	var err error
	d.epochTrie, err = NewEpochTrie(dcp.EpochHash, d.db)
	if err != nil {
		return err
	}
	d.delegateTrie, err = NewDelegateTrie(dcp.DelegateHash, d.db)
	if err != nil {
		return err
	}
	d.candidateTrie, err = NewCandidateTrie(dcp.CandidateHash, d.db)
	if err != nil {
		return err
	}
	d.voteTrie, err = NewVoteTrie(dcp.VoteHash, d.db)
	if err != nil {
		return err
	}
	d.mintCntTrie, err = NewMintCntTrie(dcp.MintCntHash, d.db)
	if err != nil {
		return err
	}
	d.periodBlock, err = NewPeriodBlockTrie(dcp.periodBlockHash, d.db)
	if err != nil {
		return err
	}
	d.maxValidators, err = NewPeriodBlockTrie(dcp.maxValidatorsHash, d.db)
	if err != nil {
		return err
	}
	d.epochInterval, err = NewPeriodBlockTrie(dcp.epochIntervalHash, d.db)
	if err != nil {
		return err
	}
	return err
}

type LCPContextProto struct {
	EpochHash         common.Hash `json:"epochRoot"        gencodec:"required"`
	DelegateHash      common.Hash `json:"delegateRoot"     gencodec:"required"`
	CandidateHash     common.Hash `json:"candidateRoot"    gencodec:"required"`
	VoteHash          common.Hash `json:"voteRoot"         gencodec:"required"`
	MintCntHash       common.Hash `json:"mintCntRoot"      gencodec:"required"`
	periodBlockHash   common.Hash `json:"periodBlockRoot"      gencodec:"required"`
	maxValidatorsHash common.Hash `json:"maxValidatorRoot"      gencodec:"required"`
	epochIntervalHash common.Hash `json:"epochIntervalRoot"      gencodec:"required"`
}

func (d *LCPContext) ToProto() *LCPContextProto {
	return &LCPContextProto{
		EpochHash:         d.epochTrie.Hash(),
		DelegateHash:      d.delegateTrie.Hash(),
		CandidateHash:     d.candidateTrie.Hash(),
		VoteHash:          d.voteTrie.Hash(),
		MintCntHash:       d.mintCntTrie.Hash(),
		periodBlockHash:   d.periodBlock.Hash(),
		maxValidatorsHash: d.maxValidators.Hash(),
		epochIntervalHash: d.epochInterval.Hash(),
	}
}

func (p *LCPContextProto) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, p.EpochHash)
	rlp.Encode(hw, p.DelegateHash)
	rlp.Encode(hw, p.CandidateHash)
	rlp.Encode(hw, p.VoteHash)
	rlp.Encode(hw, p.MintCntHash)
	rlp.Encode(hw, p.periodBlockHash)
	rlp.Encode(hw, p.maxValidatorsHash)
	rlp.Encode(hw, p.epochIntervalHash)
	hw.Sum(h[:0])
	return h
}

func (d *LCPContext) KickoutCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	err := d.candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	iter := trie.NewIterator(d.delegateTrie.PrefixIterator(candidate))
	for iter.Next() {
		delegator := iter.Value
		key := append(candidate, delegator...)
		err = d.delegateTrie.TryDelete(key)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		v, err := d.voteTrie.TryGet(delegator)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		if err == nil && bytes.Equal(v, candidate) {
			err = d.voteTrie.TryDelete(delegator)
			if err != nil {
				if _, ok := err.(*trie.MissingNodeError); !ok {
					return err
				}
			}
		}
	}
	return nil
}

func (d *LCPContext) BecomeCandidate(candidateAddr common.Address) error {
	candidate := candidateAddr.Bytes()
	return d.candidateTrie.TryUpdate(candidate, candidate)
}
func (d *LCPContext) BecomeDelegate(delegateAdr common.Address) error {
	delegate := delegateAdr.Bytes()
	return d.delegateTrie.TryUpdate(delegate, delegate)
}
func (d *LCPContext) Delegate(delegatorAddr, candidateAddr common.Address) error {
	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	// the candidate must be candidate
	candidateInTrie, err := d.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to delegate")
	}

	// delete old candidate if exists
	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	if oldCandidate != nil {
		d.delegateTrie.Delete(append(oldCandidate, delegator...))
	}
	if err = d.delegateTrie.TryUpdate(append(candidate, delegator...), delegator); err != nil {
		return err
	}
	return d.voteTrie.TryUpdate(delegator, candidate)
}

func (d *LCPContext) UnDelegate(delegatorAddr, candidateAddr common.Address) error {
	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	// the candidate must be candidate
	candidateInTrie, err := d.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to undelegate")
	}

	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate, oldCandidate) {
		return errors.New("mismatch candidate to undelegate")
	}

	if err = d.delegateTrie.TryDelete(append(candidate, delegator...)); err != nil {
		return err
	}
	return d.voteTrie.TryDelete(delegator)
}

//function to write tries to memory
func (d *LCPContext) CommitTo(db ethdb.Database) (*LCPContextProto, error) {
	epochRoot, err := d.epochTrie.Commit(nil)
	if err != nil {
		return nil, err
	}
	delegateRoot, err := d.delegateTrie.Commit(nil)
	if err != nil {
		return nil, err
	}
	voteRoot, err := d.voteTrie.Commit(nil)
	if err != nil {
		return nil, err
	}
	candidateRoot, err := d.candidateTrie.Commit(nil)
	if err != nil {
		return nil, err
	}
	mintCntRoot, err := d.mintCntTrie.Commit(nil)
	if err != nil {
		return nil, err
	}

	blockPeriod, err := d.periodBlock.Commit(nil)
	if err != nil {
		return nil, err
	}
	maxVal, err := d.maxValidators.Commit(nil)
	if err != nil {
		return nil, err
	}
	epochInterv, err := d.epochInterval.Commit(nil)
	if err != nil {
		return nil, err
	}
	return &LCPContextProto{
		EpochHash:         epochRoot,
		DelegateHash:      delegateRoot,
		VoteHash:          voteRoot,
		CandidateHash:     candidateRoot,
		MintCntHash:       mintCntRoot,
		periodBlockHash:   blockPeriod,
		maxValidatorsHash: maxVal,
		epochIntervalHash: epochInterv,
	}, nil
}

func (d *LCPContext) CandidateTrie() *trie.Trie          { return d.candidateTrie }
func (d *LCPContext) DelegateTrie() *trie.Trie           { return d.delegateTrie }
func (d *LCPContext) VoteTrie() *trie.Trie               { return d.voteTrie }
func (d *LCPContext) EpochTrie() *trie.Trie              { return d.epochTrie }
func (d *LCPContext) MintCntTrie() *trie.Trie            { return d.mintCntTrie }
func (d *LCPContext) periodTrie() *trie.Trie             { return d.periodBlock }
func (d *LCPContext) maxValidatorTrie() *trie.Trie       { return d.maxValidators }
func (d *LCPContext) epochIntervalTrie() *trie.Trie      { return d.epochInterval }
func (d *LCPContext) DB() ethdb.Database                 { return d.db }
func (dc *LCPContext) SetEpoch(epoch *trie.Trie)         { dc.epochTrie = epoch }
func (dc *LCPContext) SetDelegate(delegate *trie.Trie)   { dc.delegateTrie = delegate }
func (dc *LCPContext) SetVote(vote *trie.Trie)           { dc.voteTrie = vote }
func (dc *LCPContext) SetCandidate(candidate *trie.Trie) { dc.candidateTrie = candidate }
func (dc *LCPContext) SetMintCnt(mintCnt *trie.Trie)     { dc.mintCntTrie = mintCnt }

func (dc *LCPContext) GetValidators() ([]common.Address, error) {
	var validators []common.Address
	key := []byte("validator")
	validatorsRLP := dc.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

func (dc *LCPContext) SetValidators(validators []common.Address) error {
	key := []byte("validator")
	validatorsRLP, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return fmt.Errorf("failed to encode validators to rlp bytes: %s", err)
	}
	dc.epochTrie.Update(key, validatorsRLP)
	return nil
}

// Set and Get block creation speed
func (dc *LCPContext) SetPeriodBlock(period int64) error {
	key := []byte("BlockPeriod")
	periodByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(periodByte, uint64(period))
	dc.epochTrie.Update(key, periodByte)
	return nil
}

func (dc *LCPContext) GetPeriodBlock() int64 {
	key := []byte("BlockPeriod")
	periodRLP := dc.epochTrie.Get(key)
	period := int64(binary.LittleEndian.Uint64(periodRLP))
	return period
}

//Set and Get maximim validators
func (dc *LCPContext) SetMaxValidators(maxVal int64) error {
	key := []byte("MaxValidators")
	maxValByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(maxValByte, uint64(maxVal))
	dc.epochTrie.Update(key, maxValByte)
	return nil
}

func (dc *LCPContext) GetMaxValidators() int64 {
	key := []byte("MaxValidators")
	maxValRLP := dc.epochTrie.Get(key)
	maxVal := int64(binary.LittleEndian.Uint64(maxValRLP))
	return maxVal
}

//Set and Get epoch interval
func (dc *LCPContext) SetEpochInterval(epochInt int64) error {
	key := []byte("EpochInterval")
	epochIntByte := make([]byte, 8)
	binary.LittleEndian.PutUint64(epochIntByte, uint64(epochInt))
	dc.epochTrie.Update(key, epochIntByte)
	return nil
}
func (dc *LCPContext) GetEpochInterval() int64 {
	key := []byte("EpochInterval")
	epochIntRLP := dc.epochTrie.Get(key)
	period := int64(binary.LittleEndian.Uint64(epochIntRLP))
	return period
}
