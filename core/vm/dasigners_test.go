package vm

import (
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bn254util"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/precompiles/dasigners"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/suite"
)

type DASignersTestSuite struct {
	suite.Suite

	abi       abi.ABI
	statedb   *state.StateDB
	evm       *EVM
	dasigners *DASignersPrecompile
	signerOne common.Address
	signerTwo common.Address
}

func (suite *DASignersTestSuite) SetupTest() {
	suite.dasigners = NewDASignersPrecompile()
	suite.abi = suite.dasigners.abi

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	suite.statedb = statedb
	suite.evm = NewEVM(BlockContext{
		BlockNumber: big.NewInt(1),
	}, statedb, params.TestChainConfig, Config{})
	suite.signerOne = common.HexToAddress("0x0000000000000000000000000000000100000000")
	suite.signerTwo = common.HexToAddress("0x0000000000000000000000000000000100000001")
}

func (suite *DASignersTestSuite) runTx(input []byte, signer common.Address, gas uint64, value *uint256.Int, readonly bool) ([]byte, error) {
	suite.evm.Origin = signer
	bz, _, err := RunPrecompiledContract(suite.evm, suite.dasigners, signer, input, gas, value, readonly, nil)
	if err == nil {
		suite.statedb.Finalise(true)
		suite.statedb.Commit(suite.evm.Context.BlockNumber.Uint64(), true, false)
	}
	return bz, err
}

func (suite *DASignersTestSuite) registerSigner(testSigner common.Address, sk *big.Int) *dasigners.IDASignersSignerDetail {
	pkG1 := new(bn254.G1Affine).ScalarMultiplication(bn254util.GetG1Generator(), sk)
	pkG2 := new(bn254.G2Affine).ScalarMultiplication(bn254util.GetG2Generator(), sk)
	hash := dasigners.PubkeyRegistrationHash(testSigner, suite.evm.chainConfig.ChainID)
	signature := new(bn254.G1Affine).ScalarMultiplication(hash, sk)
	signer := dasigners.IDASignersSignerDetail{
		Signer: testSigner,
		Socket: "0.0.0.0:1234",
		PkG1:   dasigners.NewBN254G1Point(bn254util.SerializeG1(pkG1)),
		PkG2:   dasigners.NewBN254G2Point(bn254util.SerializeG2(pkG2)),
	}
	input, err := suite.abi.Pack(
		"registerSigner",
		signer,
		dasigners.NewBN254G1Point(bn254util.SerializeG1(signature)),
	)
	suite.Assert().NoError(err)

	oldLogs := suite.statedb.Logs()
	_, err = suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), false)
	suite.Assert().NoError(err)
	logs := suite.statedb.Logs()
	suite.Assert().EqualValues(len(logs), len(oldLogs)+2)

	_, err = suite.abi.Unpack("SocketUpdated", logs[len(logs)-1].Data)
	suite.Assert().NoError(err)
	_, err = suite.abi.Unpack("NewSigner", logs[len(logs)-2].Data)
	suite.Assert().NoError(err)
	return &signer
}

func (suite *DASignersTestSuite) makeEpoch(testSigner common.Address) {
	input, err := suite.abi.Pack(
		"makeEpoch",
	)
	suite.Assert().NoError(err)

	_, err = suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), false)
	suite.Assert().NoError(err)
}

func (suite *DASignersTestSuite) updateSocket(testSigner common.Address, signer *dasigners.IDASignersSignerDetail) {
	input, err := suite.abi.Pack(
		"updateSocket",
		"0.0.0.0:2345",
	)
	suite.Assert().NoError(err)

	oldLogs := suite.statedb.Logs()
	_, err = suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), false)
	suite.Assert().NoError(err)
	logs := suite.statedb.Logs()
	suite.Assert().EqualValues(len(logs), len(oldLogs)+1)

	_, err = suite.abi.Unpack("SocketUpdated", logs[len(logs)-1].Data)
	suite.Assert().NoError(err)

	signer.Socket = "0.0.0.0:2345"
}

func (suite *DASignersTestSuite) registerEpoch(testSigner common.Address, sk *big.Int) {
	epoch := suite.dasigners.epochNumber(suite.evm) + 1
	hash := dasigners.EpochRegistrationHash(testSigner, epoch, suite.evm.chainConfig.ChainID)
	signature := new(bn254.G1Affine).ScalarMultiplication(hash, sk)

	input, err := suite.abi.Pack(
		"registerNextEpoch",
		dasigners.NewBN254G1Point(bn254util.SerializeG1(signature)),
	)
	suite.Assert().NoError(err)

	_, err = suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), false)
	suite.Assert().NoError(err)
}

func (suite *DASignersTestSuite) queryEpochNumber(testSigner common.Address, expected *big.Int) {
	input, err := suite.abi.Pack(
		"epochNumber",
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["epochNumber"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	suite.Assert().EqualValues(out[0].(*big.Int).Uint64(), expected.Uint64())
}

func (suite *DASignersTestSuite) queryQuorumCount(testSigner common.Address) {
	epoch := suite.dasigners.epochNumber(suite.evm)
	input, err := suite.abi.Pack(
		"quorumCount",
		big.NewInt(int64(epoch)),
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["quorumCount"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	suite.Assert().EqualValues(out[0], big.NewInt(1))
}

func (suite *DASignersTestSuite) queryGetSigner(testSigner common.Address, answer []dasigners.IDASignersSignerDetail) {
	input, err := suite.abi.Pack(
		"getSigner",
		[]common.Address{suite.signerOne, suite.signerTwo},
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["getSigner"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	suite.Assert().EqualValues(out[0], answer)
}

func (suite *DASignersTestSuite) queryIsSigner(testSigner common.Address) {
	input, err := suite.abi.Pack(
		"isSigner",
		suite.signerOne,
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["isSigner"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	suite.Assert().EqualValues(out[0], true)
}

func (suite *DASignersTestSuite) queryRegisteredEpoch(testSigner common.Address, account common.Address, epoch *big.Int) bool {
	input, err := suite.abi.Pack(
		"registeredEpoch",
		account,
		epoch,
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["registeredEpoch"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	return out[0].(bool)
}

func (suite *DASignersTestSuite) queryGetQuorum(testSigner common.Address) []common.Address {
	epoch := suite.dasigners.epochNumber(suite.evm)
	input, err := suite.abi.Pack(
		"getQuorum",
		big.NewInt(int64(epoch)),
		big.NewInt(0),
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["getQuorum"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	return out[0].([]common.Address)
}

func (suite *DASignersTestSuite) queryGetQuorumRow(testSigner common.Address, row uint32) common.Address {
	epoch := suite.dasigners.epochNumber(suite.evm)
	input, err := suite.abi.Pack(
		"getQuorumRow",
		big.NewInt(int64(epoch)),
		big.NewInt(0),
		row,
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["getQuorumRow"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	return out[0].(common.Address)
}

func (suite *DASignersTestSuite) queryGetAggPkG1(testSigner common.Address, bitmap []byte) struct {
	AggPkG1 dasigners.BN254G1Point
	Total   *big.Int
	Hit     *big.Int
} {
	epoch := suite.dasigners.epochNumber(suite.evm)
	input, err := suite.abi.Pack(
		"getAggPkG1",
		big.NewInt(int64(epoch)),
		big.NewInt(0),
		bitmap,
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, testSigner, 10000000, uint256.NewInt(0), true)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["getAggPkG1"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	return struct {
		AggPkG1 dasigners.BN254G1Point
		Total   *big.Int
		Hit     *big.Int
	}{
		AggPkG1: out[0].(dasigners.BN254G1Point),
		Total:   out[1].(*big.Int),
		Hit:     out[2].(*big.Int),
	}
}

func (suite *DASignersTestSuite) Test_DASigners() {
	// tx test
	signer1 := suite.registerSigner(suite.signerOne, big.NewInt(1))
	signer2 := suite.registerSigner(suite.signerTwo, big.NewInt(11))
	suite.updateSocket(suite.signerOne, signer1)
	suite.updateSocket(suite.signerTwo, signer2)
	suite.registerEpoch(suite.signerOne, big.NewInt(1))
	suite.registerEpoch(suite.signerTwo, big.NewInt(11))
	// move to next epochs
	daparams := suite.dasigners.params()
	suite.queryEpochNumber(suite.signerOne, big.NewInt(0))
	// move to epoch 1 & register
	suite.makeEpoch(suite.signerOne)
	suite.queryEpochNumber(suite.signerOne, big.NewInt(1))
	suite.makeEpoch(suite.signerOne)
	suite.queryEpochNumber(suite.signerOne, big.NewInt(1))
	suite.registerEpoch(suite.signerOne, big.NewInt(1))
	suite.registerEpoch(suite.signerTwo, big.NewInt(11))
	// move to epoch 2
	suite.evm.Context.BlockNumber = suite.evm.Context.BlockNumber.Add(suite.evm.Context.BlockNumber, daparams.EpochBlocks)
	suite.makeEpoch(suite.signerOne)
	suite.queryEpochNumber(suite.signerOne, big.NewInt(2))
	// query test
	suite.queryQuorumCount(suite.signerOne)
	suite.queryGetSigner(suite.signerOne, []dasigners.IDASignersSignerDetail{*signer1, *signer2})
	suite.queryIsSigner(suite.signerOne)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerOne, big.NewInt(0)), false)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerTwo, big.NewInt(0)), false)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerOne, big.NewInt(1)), true)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerTwo, big.NewInt(1)), true)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerOne, big.NewInt(2)), true)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerTwo, big.NewInt(2)), true)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerOne, big.NewInt(3)), false)
	suite.Assert().EqualValues(suite.queryRegisteredEpoch(suite.signerOne, suite.signerTwo, big.NewInt(3)), false)

	quorum := suite.queryGetQuorum(suite.signerOne)
	suite.Assert().EqualValues(len(quorum), daparams.EncodedSlices.Int64())
	cnt := map[common.Address]int{suite.signerOne: 0, suite.signerTwo: 0}
	onePos := len(quorum)
	twoPos := len(quorum)
	for i, v := range quorum {
		suite.Assert().EqualValues(suite.queryGetQuorumRow(suite.signerOne, uint32(i)), v)
		cnt[v] += 1
		if v == suite.signerOne {
			onePos = min(onePos, i)
		} else {
			twoPos = min(twoPos, i)
		}
	}
	suite.Assert().EqualValues(cnt[suite.signerOne], len(quorum)/2)
	suite.Assert().EqualValues(cnt[suite.signerTwo], len(quorum)/2)
	// suite.Assert().EqualValues(cnt[suite.signerOne], len(quorum)/3)
	// suite.Assert().EqualValues(cnt[suite.signerTwo], len(quorum)*2/3)

	bitMap := make([]byte, len(quorum)/8)
	bitMap[onePos/8] |= 1 << (onePos % 8)
	suite.Assert().EqualValues(suite.queryGetAggPkG1(suite.signerOne, bitMap), struct {
		AggPkG1 dasigners.BN254G1Point
		Total   *big.Int
		Hit     *big.Int
	}{
		AggPkG1: dasigners.NewBN254G1Point(bn254util.SerializeG1(new(bn254.G1Affine).ScalarMultiplication(bn254util.GetG1Generator(), big.NewInt(1)))),
		Total:   big.NewInt(int64(len(quorum))),
		Hit:     big.NewInt(int64(len(quorum) / 2)),
		// Hit:     big.NewInt(int64(len(quorum) / 3)),
	})

	bitMap[twoPos/8] |= 1 << (twoPos % 8)
	suite.Assert().EqualValues(suite.queryGetAggPkG1(suite.signerOne, bitMap), struct {
		AggPkG1 dasigners.BN254G1Point
		Total   *big.Int
		Hit     *big.Int
	}{
		AggPkG1: dasigners.NewBN254G1Point(bn254util.SerializeG1(new(bn254.G1Affine).ScalarMultiplication(bn254util.GetG1Generator(), big.NewInt(1+11)))),
		Total:   big.NewInt(int64(len(quorum))),
		Hit:     big.NewInt(int64(len(quorum))),
	})

}

func (suite *DASignersTestSuite) getParams() dasigners.IDASignersParams {
	input, err := suite.abi.Pack(
		"params",
	)
	suite.Assert().NoError(err)

	bz, err := suite.runTx(input, suite.signerOne, 10000000, uint256.NewInt(0), false)
	suite.Assert().NoError(err)
	out, err := suite.abi.Methods["params"].Outputs.Unpack(bz)
	suite.Assert().NoError(err)
	params := out[0].(dasigners.IDASignersParams)
	return params
}

func (suite *DASignersTestSuite) Test_Params() {
	daParams := suite.getParams()
	expected := suite.dasigners.params()
	suite.Assert().EqualValues(expected.TokensPerVote, daParams.TokensPerVote)
	suite.Assert().EqualValues(expected.MaxVotesPerSigner, daParams.MaxVotesPerSigner)
	suite.Assert().EqualValues(expected.MaxQuorums, daParams.MaxQuorums)
	suite.Assert().EqualValues(expected.EpochBlocks, daParams.EpochBlocks)
	suite.Assert().EqualValues(expected.EncodedSlices, daParams.EncodedSlices)
}

func TestKeeperSuite(t *testing.T) {
	suite.Run(t, new(DASignersTestSuite))
}
