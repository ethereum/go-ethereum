package span

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/bor/abi"
	"github.com/ethereum/go-ethereum/consensus/bor/api"
	"github.com/ethereum/go-ethereum/consensus/bor/valset"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentValidatorsByBlockNrOrHash(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chainConfig := &params.ChainConfig{}
	validatorContractAddress := common.HexToAddress("0x1234567890123456789012345678901234567890")

	testCases := []struct {
		name               string
		blockNumber        uint64
		mockEthAPIExpected func(*api.MockCaller)
		mockAbiExpected    func(*abi.MockABI)
		expectedValidators []*valset.Validator
		expectError        bool
	}{
		{
			name:        "Successful retrieval of validators",
			blockNumber: 1000,
			mockEthAPIExpected: func(mockCaller *api.MockCaller) {
				mockCaller.EXPECT().Call(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"), nil).AnyTimes()
			},
			mockAbiExpected: func(mockAbi *abi.MockABI) {
				basicMocks(mockAbi)

				callCount := 0
				mockAbi.EXPECT().UnpackIntoInterface(
					gomock.Any(),
					gomock.Eq("producers"),
					gomock.Any(),
				).DoAndReturn(func(v interface{}, name string, data []byte) error {
					defer func() { callCount++ }()

					resp, _ := v.(*contractValidator)

					if callCount == 0 {
						*resp = contractValidator{
							Id:     big.NewInt(1),
							Signer: common.HexToAddress("0x1111111111111111111111111111111111111111"),
							Power:  big.NewInt(10),
						}
					}
					if callCount == 1 {
						*resp = contractValidator{
							Id:     big.NewInt(2),
							Signer: common.HexToAddress("0x2222222222222222222222222222222222222222"),
							Power:  big.NewInt(15),
						}
					}
					return nil
				}).AnyTimes()
			},
			expectedValidators: []*valset.Validator{
				{
					ID:          1,
					Address:     common.HexToAddress("0x1111111111111111111111111111111111111111"),
					VotingPower: 10,
				},
				{
					ID:          2,
					Address:     common.HexToAddress("0x2222222222222222222222222222222222222222"),
					VotingPower: 15,
				},
			},
			expectError: false,
		},
		{
			name:        "Successful retrieval of validators without id",
			blockNumber: 1000,
			mockEthAPIExpected: func(mockCaller *api.MockCaller) {
				mockCaller.EXPECT().Call(
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"), nil).AnyTimes()
			},
			mockAbiExpected: func(mockAbi *abi.MockABI) {
				basicMocks(mockAbi)

				mockAbi.EXPECT().UnpackIntoInterface(
					gomock.Any(),
					gomock.Eq("producers"),
					gomock.Any(),
				).DoAndReturn(func(v interface{}, name string, data []byte) error {
					return fmt.Errorf("failed")
				}).AnyTimes()
			},
			expectedValidators: []*valset.Validator{
				{
					ID:          0,
					Address:     common.HexToAddress("0x1111111111111111111111111111111111111111"),
					VotingPower: 10,
				},
				{
					ID:          0,
					Address:     common.HexToAddress("0x2222222222222222222222222222222222222222"),
					VotingPower: 15,
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockEthAPI := api.NewMockCaller(ctrl)
			mockValidatorSetABI := abi.NewMockABI(ctrl)

			// Setup
			chainSpanner := NewChainSpanner(
				mockEthAPI,
				mockValidatorSetABI,
				chainConfig,
				validatorContractAddress,
			)

			// Set up mock expectations
			tc.mockEthAPIExpected(mockEthAPI)
			tc.mockAbiExpected(mockValidatorSetABI)

			blockNumber := rpc.BlockNumber(tc.blockNumber)
			blockNrOrHash := rpc.BlockNumberOrHashWithNumber(blockNumber)

			// Execute method
			validators, err := chainSpanner.GetCurrentValidatorsByBlockNrOrHash(context.Background(), blockNrOrHash, tc.blockNumber)

			// Assertions
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, validators)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedValidators, validators)
			}
		})
	}
}

func basicMocks(mockAbi *abi.MockABI) {
	mockAbi.EXPECT().Pack(
		gomock.Any(),
		gomock.Any(),
	).Return(common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"), nil).AnyTimes()

	mockAbi.EXPECT().UnpackIntoInterface(
		gomock.Any(),
		gomock.Eq("FIRST_END_BLOCK"),
		gomock.Any(),
	).DoAndReturn(func(v interface{}, name string, data []byte) error {
		resp, _ := v.(**big.Int)
		*resp = big.NewInt(999)
		return nil
	}).AnyTimes()

	mockAbi.EXPECT().UnpackIntoInterface(
		gomock.Any(),
		gomock.Eq("getSpanByBlock"),
		gomock.Any(),
	).DoAndReturn(func(v interface{}, name string, data []byte) error {
		resp, _ := v.(**big.Int)
		*resp = big.NewInt(1)
		return nil
	}).AnyTimes()

	mockAbi.EXPECT().UnpackIntoInterface(
		gomock.Any(),
		gomock.Eq("getBorValidators"),
		gomock.Any(),
	).DoAndReturn(func(v interface{}, name string, data []byte) error {
		resp, _ := v.(*[]interface{})
		ret0, _ := (*resp)[0].(*[]common.Address)
		ret1, _ := (*resp)[1].(*[]*big.Int)

		*ret0 = []common.Address{common.HexToAddress("0x1111111111111111111111111111111111111111"), common.HexToAddress("0x2222222222222222222222222222222222222222")}
		*ret1 = []*big.Int{big.NewInt(10), big.NewInt(15)}

		return nil
	}).AnyTimes()
}
