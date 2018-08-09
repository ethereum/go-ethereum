package types

import (
	"testing"

	"github.com/pavelkrolevets/go-ethereum/common"
	"github.com/pavelkrolevets/go-ethereum/ethdb"
	"github.com/pavelkrolevets/go-ethereum/trie"
	"github.com/stretchr/testify/assert"
)

func TestDposContextSnapshot(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	dposContext, err := NewDposContext(db)
	assert.Nil(t, err)

	snapshot := dposContext.Snapshot()
	assert.Equal(t, dposContext.Root(), snapshot.Root())
	assert.NotEqual(t, dposContext, snapshot)

	// change dposContext
	assert.Nil(t, dposContext.BecomeCandidate(common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6c")))
	assert.NotEqual(t, dposContext.Root(), snapshot.Root())

	// revert snapshot
	dposContext.RevertToSnapShot(snapshot)
	assert.Equal(t, dposContext.Root(), snapshot.Root())
	assert.NotEqual(t, dposContext, snapshot)
}

func TestDposContextBecomeCandidate(t *testing.T) {
	candidates := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	db, _ := ethdb.NewMemDatabase()
	dposContext, err := NewDposContext(db)
	assert.Nil(t, err)
	for _, candidate := range candidates {
		assert.Nil(t, dposContext.BecomeCandidate(candidate))
	}

	candidateMap := map[common.Address]bool{}
	candidateIter := trie.NewIterator(dposContext.candidateTrie.NodeIterator(nil))
	for candidateIter.Next() {
		candidateMap[common.BytesToAddress(candidateIter.Value)] = true
	}
	assert.Equal(t, len(candidates), len(candidateMap))
	for _, candidate := range candidates {
		assert.True(t, candidateMap[candidate])
	}
}

func TestDposContextKickoutCandidate(t *testing.T) {
	candidates := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}
	db, _ := ethdb.NewMemDatabase()
	dposContext, err := NewDposContext(db)
	assert.Nil(t, err)
	for _, candidate := range candidates {
		assert.Nil(t, dposContext.BecomeCandidate(candidate))
		assert.Nil(t, dposContext.Delegate(candidate, candidate))
	}

	kickIdx := 1
	assert.Nil(t, dposContext.KickoutCandidate(candidates[kickIdx]))
	candidateMap := map[common.Address]bool{}
	candidateIter := trie.NewIterator(dposContext.candidateTrie.NodeIterator(nil))
	for candidateIter.Next() {
		candidateMap[common.BytesToAddress(candidateIter.Value)] = true
	}
	voteIter := trie.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	voteMap := map[common.Address]bool{}
	for voteIter.Next() {
		voteMap[common.BytesToAddress(voteIter.Value)] = true
	}
	for i, candidate := range candidates {
		delegateIter := trie.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
		if i == kickIdx {
			assert.False(t, delegateIter.Next())
			assert.False(t, candidateMap[candidate])
			assert.False(t, voteMap[candidate])
			continue
		}
		assert.True(t, delegateIter.Next())
		assert.True(t, candidateMap[candidate])
		assert.True(t, voteMap[candidate])
	}
}

func TestDposContextDelegateAndUnDelegate(t *testing.T) {
	candidate := common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e")
	newCandidate := common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2")
	delegator := common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670")
	db, _ := ethdb.NewMemDatabase()
	dposContext, err := NewDposContext(db)
	assert.Nil(t, err)
	assert.Nil(t, dposContext.BecomeCandidate(candidate))
	assert.Nil(t, dposContext.BecomeCandidate(newCandidate))

	// delegator delegate to not exist candidate
	candidateIter := trie.NewIterator(dposContext.candidateTrie.NodeIterator(nil))
	candidateMap := map[string]bool{}
	for candidateIter.Next() {
		candidateMap[string(candidateIter.Value)] = true
	}
	assert.NotNil(t, dposContext.Delegate(delegator, common.HexToAddress("0xab")))

	// delegator delegate to old candidate
	assert.Nil(t, dposContext.Delegate(delegator, candidate))
	delegateIter := trie.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	if assert.True(t, delegateIter.Next()) {
		assert.Equal(t, append(delegatePrefix, append(candidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		assert.Equal(t, delegator, common.BytesToAddress(delegateIter.Value))
	}
	voteIter := trie.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if assert.True(t, voteIter.Next()) {
		assert.Equal(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		assert.Equal(t, candidate, common.BytesToAddress(voteIter.Value))
	}

	// delegator delegate to new candidate
	assert.Nil(t, dposContext.Delegate(delegator, newCandidate))
	delegateIter = trie.NewIterator(dposContext.delegateTrie.PrefixIterator(candidate.Bytes()))
	assert.False(t, delegateIter.Next())
	delegateIter = trie.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	if assert.True(t, delegateIter.Next()) {
		assert.Equal(t, append(delegatePrefix, append(newCandidate.Bytes(), delegator.Bytes()...)...), delegateIter.Key)
		assert.Equal(t, delegator, common.BytesToAddress(delegateIter.Value))
	}
	voteIter = trie.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	if assert.True(t, voteIter.Next()) {
		assert.Equal(t, append(votePrefix, delegator.Bytes()...), voteIter.Key)
		assert.Equal(t, newCandidate, common.BytesToAddress(voteIter.Value))
	}

	// delegator undelegate to not exist candidate
	assert.NotNil(t, dposContext.UnDelegate(common.HexToAddress("0x00"), candidate))

	// delegator undelegate to old candidate
	assert.NotNil(t, dposContext.UnDelegate(delegator, candidate))

	// delegator undelegate to new candidate
	assert.Nil(t, dposContext.UnDelegate(delegator, newCandidate))
	delegateIter = trie.NewIterator(dposContext.delegateTrie.PrefixIterator(newCandidate.Bytes()))
	assert.False(t, delegateIter.Next())
	voteIter = trie.NewIterator(dposContext.voteTrie.NodeIterator(nil))
	assert.False(t, voteIter.Next())
}

func TestDposContextValidators(t *testing.T) {
	validators := []common.Address{
		common.HexToAddress("0x44d1ce0b7cb3588bca96151fe1bc05af38f91b6e"),
		common.HexToAddress("0xa60a3886b552ff9992cfcd208ec1152079e046c2"),
		common.HexToAddress("0x4e080e49f62694554871e669aeb4ebe17c4a9670"),
	}

	db, _ := ethdb.NewMemDatabase()
	dposContext, err := NewDposContext(db)
	assert.Nil(t, err)

	assert.Nil(t, dposContext.SetValidators(validators))

	result, err := dposContext.GetValidators()
	assert.Nil(t, err)
	assert.Equal(t, len(validators), len(result))
	validatorMap := map[common.Address]bool{}
	for _, validator := range validators {
		validatorMap[validator] = true
	}
	for _, validator := range result {
		assert.True(t, validatorMap[validator])
	}
}
