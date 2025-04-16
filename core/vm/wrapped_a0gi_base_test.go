package vm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	wrappeda0gibase "github.com/ethereum/go-ethereum/core/vm/precompiles/wrapped_a0gi_base"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/suite"
)

type WrappedA0giBaseTestSuite struct {
	suite.Suite

	abi       abi.ABI
	statedb   *state.StateDB
	evm       *EVM
	wa0gibase *WrappedA0giBasePrecompile
	signerOne common.Address
	signerTwo common.Address
}

func (suite *WrappedA0giBaseTestSuite) SetupTest() {
	suite.wa0gibase = NewWrappedA0giBasePrecompile()
	suite.abi = suite.wa0gibase.abi

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	suite.statedb = statedb
	suite.evm = NewEVM(BlockContext{
		BlockNumber: big.NewInt(1),
	}, statedb, params.TestChainConfig, Config{})
	suite.signerOne = common.HexToAddress("0x0000000000000000000000000000000100000000")
	suite.signerTwo = common.HexToAddress("0x0000000000000000000000000000000100000001")
}

func (suite *WrappedA0giBaseTestSuite) runTx(input []byte, signer common.Address, gas uint64, value *uint256.Int, readonly bool) ([]byte, error) {
	suite.evm.Origin = signer
	bz, _, err := RunPrecompiledContract(suite.evm, suite.wa0gibase, signer, input, gas, value, readonly, nil)
	if err == nil {
		suite.statedb.Finalise(true)
		suite.statedb.Commit(suite.evm.Context.BlockNumber.Uint64(), true, false)
	}
	return bz, err
}

func (s *WrappedA0giBaseTestSuite) TestGetW0GI() {
	method := WrappedA0GIBaseFunctionGetWA0GI

	testCases := []struct {
		name        string
		malleate    func() []byte
		postCheck   func(bz []byte)
		gas         uint64
		expErr      bool
		errContains string
	}{
		{
			"success",
			func() []byte {
				input, err := s.abi.Pack(
					method,
				)
				s.Assert().NoError(err)
				return input
			},
			func(data []byte) {
				out, err := s.abi.Methods[method].Outputs.Unpack(data)
				s.Require().NoError(err, "failed to unpack output")
				wa0gi := out[0].(common.Address)
				s.Require().Equal(wa0gi, s.wa0gibase.getWA0GI())
				// fmt.Println(wa0gi)
			},
			100000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			bz, err := s.runTx(tc.malleate(), s.signerOne, 10000000, uint256.NewInt(0), true)

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (s *WrappedA0giBaseTestSuite) TestMinterSupply() {
	method := WrappedA0GIBaseFunctionMinterSupply
	agency := s.wa0gibase.getAgency()

	testCases := []struct {
		name        string
		malleate    func() []byte
		postCheck   func(bz []byte)
		gas         uint64
		expErr      bool
		errContains string
	}{
		{
			"non-empty",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerOne,
				)
				s.Assert().NoError(err)
				return input
			},
			func(data []byte) {
				out, err := s.abi.Methods[method].Outputs.Unpack(data)
				s.Require().NoError(err, "failed to unpack output")
				supply := out[0].(wrappeda0gibase.Supply)
				fmt.Println(supply)
				s.Require().Equal(supply.Cap, big.NewInt(8e18))
				s.Require().Equal(supply.InitialSupply, big.NewInt(4e18))
				s.Require().Equal(supply.Supply, big.NewInt(4e18+1e18))
				// fmt.Println(wa0gi)
			},
			100000,
			false,
			"",
		}, {
			"empty",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerTwo,
				)
				s.Assert().NoError(err)
				return input
			},
			func(data []byte) {
				out, err := s.abi.Methods[method].Outputs.Unpack(data)
				s.Require().NoError(err, "failed to unpack output")
				supply := out[0].(wrappeda0gibase.Supply)
				s.Require().Equal(supply.Cap.Bytes(), big.NewInt(0).Bytes())
				s.Require().Equal(supply.InitialSupply.Bytes(), big.NewInt(0).Bytes())
				s.Require().Equal(supply.Supply.Bytes(), big.NewInt(0).Bytes())
				// fmt.Println(wa0gi)
			},
			100000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			// set minter cap
			input, err := s.abi.Pack(
				WrappedA0GIBaseFunctionSetMinterCap,
				s.signerOne,
				big.NewInt(8e18),
				big.NewInt(4e18),
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
			s.Assert().NoError(err)
			// mint
			input, err = s.abi.Pack(
				WrappedA0GIBaseFunctionMint,
				s.signerOne,
				big.NewInt(1e18),
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, s.wa0gibase.getWA0GI(), 10000000, uint256.NewInt(0), false)
			s.Assert().NoError(err)

			bz, err := s.runTx(tc.malleate(), s.signerOne, 10000000, uint256.NewInt(0), true)

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				s.Require().NotNil(bz)
				tc.postCheck(bz)
			}
		})
	}
}

func (s *WrappedA0giBaseTestSuite) TestMint() {
	method := WrappedA0GIBaseFunctionMint
	agency := s.wa0gibase.getAgency()

	testCases := []struct {
		name          string
		malleate      func() []byte
		postCheck     func()
		gas           uint64
		expErr        bool
		errContains   string
		isSenderWA0GI bool
	}{
		{
			"success",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerOne,
					big.NewInt(1e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {
				supply, err := s.wa0gibase.getMinterSupply(s.evm, s.signerOne)
				s.Assert().NoError(err)
				s.Require().Equal(supply.Cap, big.NewInt(8e18))
				s.Require().Equal(supply.InitialSupply, big.NewInt(4e18))
				s.Require().Equal(supply.Supply, big.NewInt(4e18+1e18))
			},
			100000,
			false,
			"",
			true,
		}, {
			"fail",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerOne,
					big.NewInt(9e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {},
			100000,
			true,
			"insufficient mint cap",
			true,
		}, {
			"invalid sender",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerTwo,
					big.NewInt(9e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {},
			100000,
			true,
			"sender is not WA0GI",
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			input, err := s.abi.Pack(
				WrappedA0GIBaseFunctionSetMinterCap,
				s.signerOne,
				big.NewInt(8e18),
				big.NewInt(4e18),
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
			s.Assert().NoError(err)

			if tc.isSenderWA0GI {
				_, err = s.runTx(tc.malleate(), s.wa0gibase.getWA0GI(), 10000000, uint256.NewInt(0), false)
			} else {
				_, err = s.runTx(tc.malleate(), s.signerTwo, 10000000, uint256.NewInt(0), false)
			}

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}

func (s *WrappedA0giBaseTestSuite) TestBurn() {
	method := WrappedA0GIBaseFunctionBurn
	agency := s.wa0gibase.getAgency()

	testCases := []struct {
		name          string
		malleate      func() []byte
		postCheck     func()
		gas           uint64
		expErr        bool
		errContains   string
		isSenderWA0GI bool
	}{
		{
			"success",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerOne,
					big.NewInt(1e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {
				supply, err := s.wa0gibase.getMinterSupply(s.evm, s.signerOne)
				s.Assert().NoError(err)
				s.Require().Equal(supply.Cap, big.NewInt(8e18))
				s.Require().Equal(supply.InitialSupply, big.NewInt(4e18))
				s.Require().Equal(supply.Supply, big.NewInt(3e18))
				// fmt.Println(wa0gi)
			},
			100000,
			false,
			"",
			true,
		}, {
			"fail",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerOne,
					big.NewInt(9e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {},
			100000,
			true,
			"insufficient mint supply",
			true,
		}, {
			"invalid sender",
			func() []byte {
				input, err := s.abi.Pack(
					method,
					s.signerTwo,
					big.NewInt(9e18),
				)
				s.Assert().NoError(err)
				return input
			},
			func() {},
			100000,
			true,
			"sender is not WA0GI",
			false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			input, err := s.abi.Pack(
				WrappedA0GIBaseFunctionSetMinterCap,
				s.signerOne,
				big.NewInt(8e18),
				big.NewInt(4e18),
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
			s.Assert().NoError(err)

			if tc.isSenderWA0GI {
				_, err = s.runTx(tc.malleate(), s.wa0gibase.getWA0GI(), 10000000, uint256.NewInt(0), false)
			} else {
				_, err = s.runTx(tc.malleate(), s.signerTwo, 10000000, uint256.NewInt(0), false)
			}

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}

func (s *WrappedA0giBaseTestSuite) TestSetMinterCap() {
	agency := s.wa0gibase.getAgency()

	testCases := []struct {
		name string
		caps []struct {
			account       common.Address
			cap           *big.Int
			initialSupply *big.Int
		}
	}{
		{
			name: "success",
			caps: []struct {
				account       common.Address
				cap           *big.Int
				initialSupply *big.Int
			}{
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
					cap:           big.NewInt(100000),
					initialSupply: big.NewInt(50000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000001"),
					cap:           big.NewInt(200000),
					initialSupply: big.NewInt(100000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000002"),
					cap:           big.NewInt(300000),
					initialSupply: big.NewInt(150000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000003"),
					cap:           big.NewInt(400000),
					initialSupply: big.NewInt(200000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000002"),
					cap:           big.NewInt(500000),
					initialSupply: big.NewInt(250000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000001"),
					cap:           big.NewInt(600000),
					initialSupply: big.NewInt(300000),
				},
				{
					account:       common.HexToAddress("0x0000000000000000000000000000000000000000"),
					cap:           big.NewInt(700000),
					initialSupply: big.NewInt(350000),
				},
			},
		},
	}
	s.Run("invalid authority", func() {
		s.SetupTest()
		input, err := s.abi.Pack(
			WrappedA0GIBaseFunctionSetMinterCap,
			s.signerOne,
			big.NewInt(8e18),
			big.NewInt(4e18),
		)
		s.Assert().NoError(err)
		_, err = s.runTx(input, s.signerOne, 10000000, uint256.NewInt(0), false)
		s.Require().Error(err)
		s.Require().Contains(err.Error(), "sender is not agency")
	})
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()

			c := make(map[common.Address]struct {
				Cap           *big.Int
				InitialSupply *big.Int
			})
			for _, cap := range tc.caps {
				input, err := s.abi.Pack(
					WrappedA0GIBaseFunctionSetMinterCap,
					cap.account,
					cap.cap,
					cap.initialSupply,
				)
				s.Assert().NoError(err)
				_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
				s.Require().NoError(err)

				supply, err := s.wa0gibase.getMinterSupply(s.evm, cap.account)
				s.Require().NoError(err)
				s.Require().Equal(supply.Cap, cap.cap)
				s.Require().Equal(supply.InitialSupply, cap.initialSupply)
				s.Require().Equal(supply.Supply, cap.initialSupply)
				c[cap.account] = struct {
					Cap           *big.Int
					InitialSupply *big.Int
				}{
					Cap:           cap.cap,
					InitialSupply: cap.initialSupply,
				}
			}
			for account, cap := range c {
				supply, err := s.wa0gibase.getMinterSupply(s.evm, account)
				s.Require().NoError(err)
				s.Require().Equal(supply.Cap, cap.Cap)
				s.Require().Equal(supply.InitialSupply, cap.InitialSupply)
				s.Require().Equal(supply.Supply, cap.InitialSupply)
			}
		})
	}
}

type MintBurn struct {
	IsMint  bool
	Minter  common.Address
	Amount  *big.Int
	Success bool
}

func (s *WrappedA0giBaseTestSuite) TestSetMintBurn() {
	agency := s.wa0gibase.getAgency()

	minter1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	minter2 := common.HexToAddress("0x0000000000000000000000000000000000000002")

	// set mint cap of minter 1 to 8 a0gi
	input, err := s.abi.Pack(
		WrappedA0GIBaseFunctionSetMinterCap,
		minter1,
		big.NewInt(8e18),
		big.NewInt(0),
	)
	s.Assert().NoError(err)
	_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
	s.Require().NoError(err)
	// set mint cap of minter 2 to 5 a0gi
	input, err = s.abi.Pack(
		WrappedA0GIBaseFunctionSetMinterCap,
		minter2,
		big.NewInt(5e18),
		big.NewInt(0),
	)
	s.Assert().NoError(err)
	_, err = s.runTx(input, agency, 10000000, uint256.NewInt(0), false)
	s.Require().NoError(err)

	testCases := []MintBurn{
		// #0, failed burn
		{
			IsMint:  false,
			Minter:  minter1,
			Amount:  big.NewInt(1e18),
			Success: false,
		},
		// #1, mint 5 a0gi by minter 1
		{
			IsMint:  true,
			Minter:  minter1,
			Amount:  big.NewInt(5e18),
			Success: true,
		},
		// #2, burn 0.5 a0gi by minter 1
		{
			IsMint:  false,
			Minter:  minter1,
			Amount:  big.NewInt(5e17),
			Success: true,
		},
		// #3, mint 0.7 a0gi by minter 2
		{
			IsMint:  true,
			Minter:  minter2,
			Amount:  big.NewInt(7e17),
			Success: true,
		},
		// #4, mint 2 a0gi by minter 2
		{
			IsMint:  true,
			Minter:  minter2,
			Amount:  big.NewInt(2e18),
			Success: true,
		},
		// #5, burn 0.3 a0gi by minter 2
		{
			IsMint:  false,
			Minter:  minter1,
			Amount:  big.NewInt(3e17),
			Success: true,
		},
		// #6, failed to mint 4 a0gi by minter 1
		{
			IsMint:  true,
			Minter:  minter1,
			Amount:  big.NewInt(4e18),
			Success: false,
		},
		// #7, mint 3.5 a0gi by minter 1
		{
			IsMint:  true,
			Minter:  minter1,
			Amount:  big.NewInt(3e18 + 5e17),
			Success: true,
		},
	}
	minted := big.NewInt(0)
	supplied := make(map[common.Address]*big.Int)
	for _, c := range testCases {
		if c.IsMint {
			input, err = s.abi.Pack(
				WrappedA0GIBaseFunctionMint,
				c.Minter,
				c.Amount,
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, s.wa0gibase.getWA0GI(), 10000000, uint256.NewInt(0), false)
		} else {
			input, err = s.abi.Pack(
				WrappedA0GIBaseFunctionBurn,
				c.Minter,
				c.Amount,
			)
			s.Assert().NoError(err)
			_, err = s.runTx(input, s.wa0gibase.getWA0GI(), 10000000, uint256.NewInt(0), false)
		}
		if c.Success {
			if c.IsMint {
				minted.Add(minted, c.Amount)
				if amt, ok := supplied[c.Minter]; ok {
					amt.Add(amt, c.Amount)
				} else {
					supplied[c.Minter] = new(big.Int).Set(c.Amount)
				}
			} else {
				minted.Sub(minted, c.Amount)
				if amt, ok := supplied[c.Minter]; ok {
					amt.Sub(amt, c.Amount)
				} else {
					supplied[c.Minter] = new(big.Int).Set(c.Amount)
				}
			}
			s.Require().NoError(err)
			supply, err := s.wa0gibase.getMinterSupply(s.evm, c.Minter)
			s.Require().NoError(err)
			s.Require().Equal(supplied[c.Minter].Bytes(), supply.Supply.Bytes())
		} else {
			s.Require().Error(err)
		}
		balance := s.evm.StateDB.GetBalance(s.wa0gibase.getWA0GI())
		s.Require().Equal(balance.ToBig().Bytes(), minted.Bytes())
	}
}

func TestWrappedA0giBaseTestSuite(t *testing.T) {
	suite.Run(t, new(WrappedA0giBaseTestSuite))
}
