package bor

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
)

const (
	numVals = 100
)

func TestGetSignerSuccessionNumber_ProposerIsSigner(t *testing.T) {
	validators := buildRandomValidatorSet(numVals)
	validatorSet := NewValidatorSet(validators)
	snap := Snapshot{
		ValidatorSet: validatorSet,
	}

	// proposer is signer
	signer := validatorSet.Proposer.Address
	successionNumber, err := snap.GetSignerSuccessionNumber(signer)
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.Equal(t, 0, successionNumber)
}

func TestGetSignerSuccessionNumber_SignerIndexIsLarger(t *testing.T) {
	validators := buildRandomValidatorSet(numVals)

	// sort validators by address, which is what NewValidatorSet also does
	sort.Sort(ValidatorsByAddress(validators))
	proposerIndex := 32
	signerIndex := 56
	// give highest ProposerPriority to a particular val, so that they become the proposer
	validators[proposerIndex].VotingPower = 200
	snap := Snapshot{
		ValidatorSet: NewValidatorSet(validators),
	}

	// choose a signer at an index greater than proposer index
	signer := snap.ValidatorSet.Validators[signerIndex].Address
	successionNumber, err := snap.GetSignerSuccessionNumber(signer)
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.Equal(t, signerIndex-proposerIndex, successionNumber)
}

func TestGetSignerSuccessionNumber_SignerIndexIsSmaller(t *testing.T) {
	validators := buildRandomValidatorSet(numVals)
	proposerIndex := 98
	signerIndex := 11
	// give highest ProposerPriority to a particular val, so that they become the proposer
	validators[proposerIndex].VotingPower = 200
	snap := Snapshot{
		ValidatorSet: NewValidatorSet(validators),
	}

	// choose a signer at an index greater than proposer index
	signer := snap.ValidatorSet.Validators[signerIndex].Address
	successionNumber, err := snap.GetSignerSuccessionNumber(signer)
	if err != nil {
		t.Fatalf("%s", err)
	}
	assert.Equal(t, signerIndex+numVals-proposerIndex, successionNumber)
}

func TestGetSignerSuccessionNumber_ProposerNotFound(t *testing.T) {
	validators := buildRandomValidatorSet(numVals)
	snap := Snapshot{
		ValidatorSet: NewValidatorSet(validators),
	}
	dummyProposerAddress := randomAddress()
	snap.ValidatorSet.Proposer = &Validator{Address: dummyProposerAddress}
	// choose any signer
	signer := snap.ValidatorSet.Validators[3].Address
	_, err := snap.GetSignerSuccessionNumber(signer)
	assert.NotNil(t, err)
	e, ok := err.(*UnauthorizedProposerError)
	assert.True(t, ok)
	assert.Equal(t, dummyProposerAddress.Bytes(), e.Proposer)
}

func TestGetSignerSuccessionNumber_SignerNotFound(t *testing.T) {
	validators := buildRandomValidatorSet(numVals)
	snap := Snapshot{
		ValidatorSet: NewValidatorSet(validators),
	}
	dummySignerAddress := randomAddress()
	_, err := snap.GetSignerSuccessionNumber(dummySignerAddress)
	assert.NotNil(t, err)
	e, ok := err.(*UnauthorizedSignerError)
	assert.True(t, ok)
	assert.Equal(t, dummySignerAddress.Bytes(), e.Signer)
}

func buildRandomValidatorSet(numVals int) []*Validator {
	rand.Seed(time.Now().Unix())
	validators := make([]*Validator, numVals)
	for i := 0; i < numVals; i++ {
		validators[i] = &Validator{
			Address: randomAddress(),
			// cannot process validators with voting power 0, hence +1
			VotingPower: int64(rand.Intn(99) + 1),
		}
	}

	// sort validators by address, which is what NewValidatorSet also does
	sort.Sort(ValidatorsByAddress(validators))
	return validators
}

func randomAddress() common.Address {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return common.BytesToAddress(bytes)
}
