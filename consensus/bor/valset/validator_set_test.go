package valset

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func NewValidatorFromKey(key string, votingPower int64) *Validator {
	privKey, _ := crypto.HexToECDSA(key)

	return NewValidator(crypto.PubkeyToAddress(privKey.PublicKey), votingPower)
}

func GetValidators() [4]*Validator {
	const (
		// addr0 = 0x96C42C56fdb78294F96B0cFa33c92bed7D75F96a
		signer0 = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd9"

		// addr1 = 0x98925BE497f6dFF6A5a33dDA8B5933cA35262d69
		signer1 = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd8"

		//addr2 = 0x648Cf2A5b119E2c04061021834F8f75735B1D36b
		signer2 = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd7"

		//addr3 = 0x168f220B3b313D456eD4797520eFdFA9c57E6C45
		signer3 = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd6"
	)

	return [4]*Validator{
		NewValidatorFromKey(signer0, 100),
		NewValidatorFromKey(signer1, 200),
		NewValidatorFromKey(signer2, 300),
		NewValidatorFromKey(signer3, 400),
	}
}

func TestIncrementProposerPriority(t *testing.T) {
	t.Parallel()

	vals := GetValidators()

	// Validator set length = 1
	valSet := NewValidatorSet(vals[:1])

	expectedPropsers := []*Validator{vals[0], vals[0], vals[0], vals[0], vals[0], vals[0], vals[0], vals[0], vals[0], vals[0]}

	for i := 0; i < 10; i++ {
		valSet.IncrementProposerPriority(1)

		require.Equal(t, expectedPropsers[i].Address, valSet.GetProposer().Address)
	}

	// Validator set length = 2
	valSet = NewValidatorSet(vals[:2])

	expectedPropsers = []*Validator{vals[0], vals[1], vals[1], vals[0], vals[1], vals[1], vals[0], vals[1], vals[1], vals[0]}

	for i := 0; i < 10; i++ {
		valSet.IncrementProposerPriority(1)

		require.Equal(t, expectedPropsers[i].Address, valSet.GetProposer().Address)
	}

	// Validator set length = 3
	valSet = NewValidatorSet(vals[:3])

	expectedPropsers = []*Validator{vals[1], vals[2], vals[0], vals[1], vals[2], vals[2], vals[1], vals[2], vals[0], vals[1]}

	for i := 0; i < 10; i++ {
		valSet.IncrementProposerPriority(1)

		require.Equal(t, expectedPropsers[i].Address, valSet.GetProposer().Address)
	}

	// Validator set length = 4
	valSet = NewValidatorSet(vals[:4])

	expectedPropsers = []*Validator{vals[2], vals[1], vals[3], vals[2], vals[0], vals[3], vals[1], vals[2], vals[3], vals[3]}

	for i := 0; i < 10; i++ {
		valSet.IncrementProposerPriority(1)

		require.Equal(t, expectedPropsers[i].Address, valSet.GetProposer().Address)
	}
}

func TestRescalePriorities(t *testing.T) {
	t.Parallel()

	vals := GetValidators()

	// Validator set length = 1
	valSet := NewValidatorSet(vals[:1])

	valSet.RescalePriorities(10)

	expectedPriorities := []int64{0}
	for i, val := range valSet.Validators {
		require.Equal(t, expectedPriorities[i], val.ProposerPriority)
	}

	// Validator set length = 2

	valSet = NewValidatorSet(vals[:2])

	valSet.RescalePriorities(100)

	expectedPriorities = []int64{50, -50}
	for i, val := range valSet.Validators {
		require.Equal(t, expectedPriorities[i], val.ProposerPriority)
	}

	// Validator set length = 3

	valSet = NewValidatorSet(vals[:3])

	valSet.RescalePriorities(30)

	expectedPriorities = []int64{-17, 5, 11}
	for i, val := range valSet.Validators {
		require.Equal(t, expectedPriorities[i], val.ProposerPriority)
	}

	// Validator set length = 4

	valSet = NewValidatorSet(vals[:4])

	valSet.RescalePriorities(10)

	expectedPriorities = []int64{-6, 3, 1, 2}
	for i, val := range valSet.Validators {
		require.Equal(t, expectedPriorities[i], val.ProposerPriority)
	}
}

func TestGetValidatorByAddressAndIndex(t *testing.T) {
	t.Parallel()

	vals := GetValidators()
	valSet := NewValidatorSet(vals[:4])

	for _, val := range valSet.Validators {
		idx, valByAddress := valSet.GetByAddress(val.Address)
		addr, valByIndex := valSet.GetByIndex(idx)

		assert.DeepEqual(t, val, valByIndex)
		assert.DeepEqual(t, val, valByAddress)
		assert.DeepEqual(t, val.Address, addr)
	}

	tempAddress := common.HexToAddress("0x12345")

	// Negative Testcase
	idx, _ := valSet.GetByAddress(tempAddress)
	require.Equal(t, idx, -1)

	// checking for validator index out of range
	addr, _ := valSet.GetByIndex(100)
	require.Equal(t, addr, common.Address{})
}

func TestUpdateWithChangeSet(t *testing.T) {
	t.Parallel()

	vals := GetValidators()
	valSet := NewValidatorSet(vals[:4])

	// halved the power of vals[2] and doubled the power of vals[3]
	vals[2].VotingPower = 150
	vals[3].VotingPower = 800

	// Adding new temp validator in the set
	const tempSigner = "c8deb0bea5c41afe8e37b4d1bd84e31adff11b09c8c96ff4b605003cce067cd5"

	tempVal := NewValidatorFromKey(tempSigner, 250)

	// check totalVotingPower before updating validator set
	require.Equal(t, int64(1000), valSet.TotalVotingPower())

	err := valSet.UpdateWithChangeSet([]*Validator{vals[2], vals[3], tempVal})
	require.NoError(t, err)

	// check totalVotingPower after updating validator set
	require.Equal(t, int64(1500), valSet.TotalVotingPower())

	_, updatedVal2 := valSet.GetByAddress(vals[2].Address)
	require.Equal(t, int64(150), updatedVal2.VotingPower)

	_, updatedVal3 := valSet.GetByAddress(vals[3].Address)
	require.Equal(t, int64(800), updatedVal3.VotingPower)

	_, updatedTempVal := valSet.GetByAddress(tempVal.Address)
	require.Equal(t, int64(250), updatedTempVal.VotingPower)
}

func TestValidatorSet_IncludeIds(t *testing.T) {
	v1 := &Validator{
		Address:          common.HexToAddress("0x1111111111111111111111111111111111111111"),
		VotingPower:      100,
		ProposerPriority: 0,
		ID:               0,
	}
	v2 := &Validator{
		Address:          common.HexToAddress("0x2222222222222222222222222222222222222222"),
		VotingPower:      200,
		ProposerPriority: 0,
		ID:               0,
	}

	valSet := NewValidatorSet([]*Validator{v1, v2})

	valsWithId := []*Validator{
		{
			Address:          v1.Address,
			ID:               10, // new ID for v1
			VotingPower:      999,
			ProposerPriority: 999,
		},
		{
			Address:          v2.Address,
			ID:               20, // new ID for v2
			VotingPower:      999,
			ProposerPriority: 999,
		},
		{
			Address:     common.HexToAddress("0x3333333333333333333333333333333333333333"),
			ID:          30,
			VotingPower: 300,
		},
	}

	valSet.IncludeIds(valsWithId)

	assert.Equal(t, uint64(10), valSet.Validators[0].ID, "v1 ID should be updated to 10")
	assert.Equal(t, uint64(20), valSet.Validators[1].ID, "v2 ID should be updated to 20")

	assert.Equal(t, 2, len(valSet.Validators), "No extra validators should be added")

	assert.Equal(t, int64(100), valSet.Validators[0].VotingPower)
	assert.Equal(t, int64(200), valSet.Validators[1].VotingPower)
}

func TestValidatorSet_IncludeIds_EmptySet(t *testing.T) {
	valSet := NewValidatorSet(nil) // empty set

	valSet.IncludeIds([]*Validator{
		{
			Address: common.HexToAddress("0xabcdef"),
			ID:      42,
		},
	})

	assert.Equal(t, 0, valSet.Size(), "ValidatorSet remains empty")
}

func TestCheckEmptyId(t *testing.T) {
	tests := []struct {
		name         string
		validatorSet ValidatorSet
		expected     bool
	}{
		{
			name:         "Empty ValidatorSet",
			validatorSet: ValidatorSet{Validators: []*Validator{}},
			expected:     false,
		},
		{
			name: "All Validators with Non-Zero IDs",
			validatorSet: ValidatorSet{
				Validators: []*Validator{
					{ID: 1},
					{ID: 2},
					{ID: 3},
				},
			},
			expected: false,
		},
		{
			name: "One Validator with ID Zero",
			validatorSet: ValidatorSet{
				Validators: []*Validator{
					{ID: 0},
					{ID: 2},
					{ID: 3},
				},
			},
			expected: true,
		},
		{
			name: "All Validators with ID Zero",
			validatorSet: ValidatorSet{
				Validators: []*Validator{
					{ID: 0},
					{ID: 0},
					{ID: 0},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.validatorSet.CheckEmptyId()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
