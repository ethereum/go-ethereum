// Copyright 2015 The go-ethereum Authors
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

package tests

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

func TestBlockchainBAL(t *testing.T) {
	bt := new(testMatcher)

	// We are running most of GeneralStatetests to tests witness support, even
	// though they are ran as state tests too. Still, the performance tests are
	// less about state andmore about EVM number crunching, so skip those.
	bt.skipLoad(`^GeneralStateTests/VMTests/vmPerformance`)

	// Skip random failures due to selfish mining test
	bt.skipLoad(`.*bcForgedTest/bcForkUncle\.json`)

	// Slow tests
	bt.slow(`.*bcExploitTest/DelegateCallSpam.json`)
	bt.slow(`.*bcExploitTest/ShanghaiLove.json`)
	bt.slow(`.*bcExploitTest/SuicideIssue.json`)
	bt.slow(`.*/bcForkStressTest/`)
	bt.slow(`.*/bcGasPricerTest/RPC_API_Test.json`)
	bt.slow(`.*/bcWalletTest/`)

	// Very slow test
	bt.skipLoad(`.*/stTimeConsuming/.*`)
	// test takes a lot for time and goes easily OOM because of sha3 calculation on a huge range,
	// using 4.6 TGas
	bt.skipLoad(`.*randomStatetest94.json.*`)

	// After the merge we would accept side chains as canonical even if they have lower td
	bt.skipLoad(`.*bcMultiChainTest/ChainAtoChainB_difficultyB.json`)
	bt.skipLoad(`.*bcMultiChainTest/CallContractFromNotBestBlock.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/uncleBlockAtBlock3afterBlock4.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/lotsOfBranchesOverrideAtTheMiddle.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/sideChainWithMoreTransactions.json`)
	bt.skipLoad(`.*bcForkStressTest/ForkStressTest.json`)
	bt.skipLoad(`.*bcMultiChainTest/lotsOfLeafs.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain2.json`)

	// With chain history removal, TDs become unavailable, this transition tests based on TTD are unrunnable
	bt.skipLoad(`.*bcArrowGlacierToParis/powToPosBlockRejection.json`)

	// This directory contains no test.
	bt.skipLoad(`.*\.meta/.*`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		config, ok := Forks[test.json.Network]
		if !ok {
			t.Fatalf("unsupported fork: %s\n", test.json.Network)
		}
		gspec := test.genesis(config)
		// skip any tests which are not past the cancun fork (selfdestruct removal)
		if gspec.Config.CancunTime == nil || *gspec.Config.CancunTime != 0 {
			return
		}
		execBlockTest(t, bt, test, true)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

func TestBlockchain(t *testing.T) {
	bt := new(testMatcher)

	// We are running most of GeneralStatetests to tests witness support, even
	// though they are ran as state tests too. Still, the performance tests are
	// less about state andmore about EVM number crunching, so skip those.
	bt.skipLoad(`^GeneralStateTests/VMTests/vmPerformance`)

	// Skip random failures due to selfish mining test
	bt.skipLoad(`.*bcForgedTest/bcForkUncle\.json`)

	// Slow tests
	bt.slow(`.*bcExploitTest/DelegateCallSpam.json`)
	bt.slow(`.*bcExploitTest/ShanghaiLove.json`)
	bt.slow(`.*bcExploitTest/SuicideIssue.json`)
	bt.slow(`.*/bcForkStressTest/`)
	bt.slow(`.*/bcGasPricerTest/RPC_API_Test.json`)
	bt.slow(`.*/bcWalletTest/`)

	// Very slow test
	bt.skipLoad(`.*/stTimeConsuming/.*`)
	// test takes a lot for time and goes easily OOM because of sha3 calculation on a huge range,
	// using 4.6 TGas
	bt.skipLoad(`.*randomStatetest94.json.*`)

	// After the merge we would accept side chains as canonical even if they have lower td
	bt.skipLoad(`.*bcMultiChainTest/ChainAtoChainB_difficultyB.json`)
	bt.skipLoad(`.*bcMultiChainTest/CallContractFromNotBestBlock.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/uncleBlockAtBlock3afterBlock4.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/lotsOfBranchesOverrideAtTheMiddle.json`)
	bt.skipLoad(`.*bcTotalDifficultyTest/sideChainWithMoreTransactions.json`)
	bt.skipLoad(`.*bcForkStressTest/ForkStressTest.json`)
	bt.skipLoad(`.*bcMultiChainTest/lotsOfLeafs.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain.json`)
	bt.skipLoad(`.*bcFrontierToHomestead/blockChainFrontierWithLargerTDvsHomesteadBlockchain2.json`)

	// With chain history removal, TDs become unavailable, this transition tests based on TTD are unrunnable
	bt.skipLoad(`.*bcArrowGlacierToParis/powToPosBlockRejection.json`)

	// This directory contains no test.
	bt.skipLoad(`.*\.meta/.*`)

	bt.walk(t, blockTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test, false)
	})
	// There is also a LegacyTests folder, containing blockchain tests generated
	// prior to Istanbul. However, they are all derived from GeneralStateTests,
	// which run natively, so there's no reason to run them here.
}

func testExecutionSpecBlocktests(t *testing.T, testDir string, skip []string) {
	if !common.FileExist(testDir) {
		t.Skipf("directory %s does not exist", testDir)
	}
	bt := new(testMatcher)

	for _, skipTest := range skip {
		bt.skipLoad(skipTest)
	}
	// These tests require us to handle scenarios where a system contract is not deployed at a fork
	bt.skipLoad(".*prague/eip7251_consolidations/test_system_contract_deployment.json")
	bt.skipLoad(".*prague/eip7002_el_triggerable_withdrawals/test_system_contract_deployment.json")

	bt.walk(t, testDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test, true)
	})
}

// TestExecutionSpecBlocktests runs the test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktests(t *testing.T) {
	testExecutionSpecBlocktests(t, executionSpecBlockchainTestDir, []string{})
}

// TestExecutionSpecBlocktestsBAL runs the BAL release test fixtures from execution-spec-tests.
func TestExecutionSpecBlocktestsBAL(t *testing.T) {
	skips := []string{
		".*/Cancun/stEIP1153_transientStorage/10_revertUndoesStoreAfterReturnFiller.yml",
		".*/Cancun/stEIP1153_transientStorage/14_revertAfterNestedStaticcallFiller.yml",
		".*/Cancun/stEIP1153_transientStorage/17_tstoreGasFiller.yml",
		".*/Cancun/stEIP4844_blobtransactions/createBlobhashTxFiller.yml",
		".*/Cancun/stEIP5656_MCOPY/MCOPY_copy_costFiller.yml",
		".*/Cancun/stEIP5656_MCOPY/MCOPY_memory_expansion_costFiller.yml",
		".*/Cancun/stEIP5656_MCOPY/MCOPY_memory_hashFiller.yml",
		".*/Cancun/stEIP5656_MCOPY/MCOPYFiller.yml",
		".*/Shanghai/stEIP3855_push0/push0GasFiller.yml",
		".*/Shanghai/stEIP3860_limitmeterinitcode/create2InitCodeSizeLimitFiller.yml",
		".*/Shanghai/stEIP3860_limitmeterinitcode/createInitCodeSizeLimitFiller.yml",
		".*/Shanghai/stEIP3860_limitmeterinitcode/creationTxInitCodeSizeLimitFiller.yml",
		".*/stAttackTest/ContractCreationSpamFiller.json",
		".*/stAttackTest/CrashingTransactionFiller.json",
		".*/stBadOpcode/measureGasFiller.yml",
		".*/stBadOpcode/operationDiffGasFiller.yml",
		".*/stCallCodes/callcallcall_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcode_checkPCFiller.json",
		".*/stCallCodes/callcodecallcall_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcodecallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcodecallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallCodes/callcodecallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallCreateCallCodeTest/Call1024OOGFiller.json",
		".*/stCallCreateCallCodeTest/Callcode1024OOGFiller.json",
		".*/stCallCreateCallCodeTest/CallcodeLoseGasOOGFiller.json",
		".*/stCallCreateCallCodeTest/CallLoseGasOOGFiller.json",
		".*/stCallCreateCallCodeTest/callWithHighValueOOGinCallFiller.json",
		".*/stCallCreateCallCodeTest/contractCreationMakeCallThatAskMoreGasThenTransactionProvidedFiller.json",
		".*/stCallCreateCallCodeTest/createFailBalanceTooLowFiller.json",
		".*/stCallCreateCallCodeTest/createInitFailBadJumpDestination2Filler.json",
		".*/stCallCreateCallCodeTest/createInitFailBadJumpDestinationFiller.json",
		".*/stCallCreateCallCodeTest/createInitFailStackSizeLargerThan1024Filler.json",
		".*/stCallCreateCallCodeTest/createInitFailStackUnderflowFiller.json",
		".*/stCallCreateCallCodeTest/createInitFailUndefinedInstruction2Filler.json",
		".*/stCallCreateCallCodeTest/createInitFailUndefinedInstructionFiller.json",
		".*/stCallCreateCallCodeTest/createNameRegistratorPerTxsFiller.json",
		".*/stCallCreateCallCodeTest/createNameRegistratorPerTxsNotEnoughGasFiller.json",
		".*/stCallCreateCallCodeTest/createNameRegistratorPreStore1NotEnoughGasFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcodecallcall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcodecallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcodecallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesCallCodeHomestead/callcodecallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcodecallcall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcodecallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcodecallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stCallDelegateCodesHomestead/callcodecallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stChainId/chainIdFiller.json",
		".*/stChainId/chainIdGasCostFiller.json",
		".*/stCodeCopyTest/ExtCodeCopyTargetRangeLongerThanCodeTestsFiller.json",
		".*/stCodeCopyTest/ExtCodeCopyTestsParisFiller.json",
		".*/stCreate2/call_outsize_then_create2_successful_then_returndatasizeFiller.json",
		".*/stCreate2/call_then_create2_successful_then_returndatasizeFiller.json",
		".*/stCreate2/CREATE2_FirstByte_loopFiller.yml",
		".*/stCreate2/create2callPrecompilesFiller.json",
		".*/stCreate2/Create2OOGafterInitCodeReturndata2Filler.json",
		".*/stCreate2/Create2OOGFromCallRefundsFiller.yml",
		".*/stCreate2/create2SmartInitCodeFiller.json",
		".*/stCreate2/CreateMessageRevertedFiller.json",
		".*/stCreate2/CreateMessageRevertedOOGInInit2Filler.json",
		".*/stCreate2/returndatacopy_0_0_following_successful_createFiller.json",
		".*/stCreate2/returndatacopy_afterFailing_createFiller.json",
		".*/stCreate2/returndatacopy_following_revert_in_createFiller.json",
		".*/stCreate2/returndatasize_following_successful_createFiller.json",
		".*/stCreate2/RevertDepthCreate2OOGBerlinFiller.json",
		".*/stCreate2/RevertDepthCreate2OOGFiller.json",
		".*/stCreate2/RevertDepthCreateAddressCollisionBerlinFiller.json",
		".*/stCreate2/RevertDepthCreateAddressCollisionFiller.json",
		".*/stCreate2/RevertOpcodeCreateFiller.json",
		".*/stCreate2/RevertOpcodeInCreateReturnsCreate2Filler.json",
		".*/stCreateTest/CodeInConstructorFiller.yml",
		".*/stCreateTest/CREATE_EContract_ThenCALLToNonExistentAccFiller.json",
		".*/stCreateTest/CREATE_EContractCreateNEContractInInitOOG_TrFiller.json",
		".*/stCreateTest/CREATE_EmptyContractAndCallIt_0weiFiller.json",
		".*/stCreateTest/CREATE_EmptyContractAndCallIt_1weiFiller.json",
		".*/stCreateTest/CREATE_EmptyContractFiller.json",
		".*/stCreateTest/CREATE_EmptyContractWithBalanceFiller.json",
		".*/stCreateTest/CREATE_EmptyContractWithStorageAndCallIt_0weiFiller.json",
		".*/stCreateTest/CREATE_EmptyContractWithStorageAndCallIt_1weiFiller.json",
		".*/stCreateTest/CREATE_EmptyContractWithStorageFiller.json",
		".*/stCreateTest/CreateAddressWarmAfterFailFiller.yml",
		".*/stCreateTest/CreateCollisionResultsFiller.yml",
		".*/stCreateTest/CreateCollisionToEmpty2Filler.json",
		".*/stCreateTest/CreateOOGafterInitCodeReturndata2Filler.json",
		".*/stCreateTest/CreateOOGafterInitCodeRevert2Filler.json",
		".*/stCreateTest/CreateOOGFromCallRefundsFiller.yml",
		".*/stCreateTest/CreateResultsFiller.yml",
		".*/stCreateTest/TransactionCollisionToEmpty2Filler.json",
		".*/stDelegatecallTestHomestead/Call1024OOGFiller.json",
		".*/stDelegatecallTestHomestead/CallcodeLoseGasOOGFiller.json",
		".*/stDelegatecallTestHomestead/CallLoseGasOOGFiller.json",
		".*/stDelegatecallTestHomestead/Delegatecall1024OOGFiller.json",
		".*/stDelegatecallTestHomestead/delegatecallOOGinCallFiller.json",
		".*/stEIP150singleCodeGasPrices/gasCostBerlinFiller.yml",
		".*/stEIP150singleCodeGasPrices/gasCostFiller.yml",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasMemoryAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasMemoryFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasValueTransferAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasValueTransferFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasValueTransferMemoryAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallCodeGasValueTransferMemoryFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasValueTransferAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasValueTransferFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasValueTransferMemoryAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallGasValueTransferMemoryFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallMemoryGasAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCallMemoryGasFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCreateFailGasValueTransfer2Filler.json",
		".*/stEIP150singleCodeGasPrices/RawCreateFailGasValueTransferFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCreateGasFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCreateGasMemoryFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCreateGasValueTransferFiller.json",
		".*/stEIP150singleCodeGasPrices/RawCreateGasValueTransferMemoryFiller.json",
		".*/stEIP150singleCodeGasPrices/RawDelegateCallGasAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawDelegateCallGasFiller.json",
		".*/stEIP150singleCodeGasPrices/RawDelegateCallGasMemoryAskFiller.json",
		".*/stEIP150singleCodeGasPrices/RawDelegateCallGasMemoryFiller.json",
		".*/stEIP150Specific/CallAskMoreGasOnDepth2ThenTransactionHasFiller.json",
		".*/stEIP150Specific/CreateAndGasInsideCreateFiller.json",
		".*/stEIP150Specific/DelegateCallOnEIPFiller.json",
		".*/stEIP150Specific/NewGasPriceForCodesFiller.json",
		".*/stEIP150Specific/Transaction64Rule_d64e0Filler.json",
		".*/stEIP150Specific/Transaction64Rule_d64m1Filler.json",
		".*/stEIP150Specific/Transaction64Rule_d64p1Filler.json",
		".*/stEIP1559/baseFeeDiffPlacesOsakaFiller.yml",
		".*/stEIP1559/gasPriceDiffPlacesOsakaFiller.yml",
		".*/stEIP158Specific/EXP_EmptyFiller.json",
		".*/stEIP2930/addressOpcodesFiller.yml",
		".*/stEIP2930/coinbaseT01Filler.yml",
		".*/stEIP2930/coinbaseT2Filler.yml",
		".*/stEIP2930/manualCreateFiller.yml",
		".*/stEIP2930/storageCostsFiller.yml",
		".*/stEIP2930/transactionCostsFiller.yml",
		".*/stEIP2930/variedContextFiller.yml",
		".*/stEIP3607/initCollidingWithNonEmptyAccountFiller.yml",
		".*/stEIP3607/transactionCollidingWithNonEmptyAccount_init_ParisFiller.yml",
		".*/stExample/add11_ymlFiller.yml",
		".*/stExample/add11Filler.json",
		".*/stExample/basefeeExampleFiller.yml",
		".*/stExample/indexesOmitExampleFiller.yml",
		".*/stExample/labelsExampleFiller.yml",
		".*/stExample/rangesExampleFiller.yml",
		".*/stExtCodeHash/callToNonExistentFiller.json",
		".*/stExtCodeHash/callToSuicideThenExtcodehashFiller.json",
		".*/stExtCodeHash/createEmptyThenExtcodehashFiller.json",
		".*/stInitCodeTest/CallContractToCreateContractAndCallItOOGFiller.json",
		".*/stInitCodeTest/CallContractToCreateContractOOGBonusGasFiller.json",
		".*/stInitCodeTest/CallContractToCreateContractWhichWouldCreateContractIfCalledFiller.json",
		".*/stInitCodeTest/CallContractToCreateContractWhichWouldCreateContractInInitCodeFiller.json",
		".*/stInitCodeTest/CallTheContractToCreateEmptyContractFiller.json",
		".*/stInitCodeTest/OutOfGasContractCreationFiller.json",
		".*/stInitCodeTest/OutOfGasPrefundedContractCreationFiller.json",
		".*/stInitCodeTest/ReturnTest2Filler.json",
		".*/stInitCodeTest/StackUnderFlowContractCreationFiller.json",
		".*/stInitCodeTest/TransactionCreateRandomInitCodeFiller.json",
		".*/stInitCodeTest/TransactionCreateSuicideInInitcodeFiller.json",
		".*/stMemExpandingEIP150Calls/CallAskMoreGasOnDepth2ThenTransactionHasWithMemExpandingCallsFiller.json",
		".*/stMemExpandingEIP150Calls/CallGoesOOGOnSecondLevelWithMemExpandingCallsFiller.json",
		".*/stMemExpandingEIP150Calls/CreateAndGasInsideCreateWithMemExpandingCallsFiller.json",
		".*/stMemExpandingEIP150Calls/NewGasPriceForCodesWithMemExpandingCallsFiller.json",
		".*/stMemoryStressTest/RETURN_BoundsFiller.json",
		".*/stMemoryStressTest/SSTORE_BoundsFiller.json",
		".*/stMemoryTest/calldatacopy_dejavu2Filler.json",
		".*/stMemoryTest/mem0b_singleByteFiller.json",
		".*/stMemoryTest/mem31b_singleByteFiller.json",
		".*/stMemoryTest/mem32b_singleByteFiller.json",
		".*/stMemoryTest/mem32kb_singleByte-1Filler.json",
		".*/stMemoryTest/mem32kb_singleByte-31Filler.json",
		".*/stMemoryTest/mem32kb_singleByte-32Filler.json",
		".*/stMemoryTest/mem32kb_singleByte-33Filler.json",
		".*/stMemoryTest/mem32kb_singleByte+1Filler.json",
		".*/stMemoryTest/mem32kb_singleByte+31Filler.json",
		".*/stMemoryTest/mem32kb_singleByte+32Filler.json",
		".*/stMemoryTest/mem32kb_singleByte+33Filler.json",
		".*/stMemoryTest/mem32kb_singleByteFiller.json",
		".*/stMemoryTest/mem32kb-1Filler.json",
		".*/stMemoryTest/mem32kb-31Filler.json",
		".*/stMemoryTest/mem32kb-32Filler.json",
		".*/stMemoryTest/mem32kb-33Filler.json",
		".*/stMemoryTest/mem32kb+1Filler.json",
		".*/stMemoryTest/mem32kb+31Filler.json",
		".*/stMemoryTest/mem32kb+32Filler.json",
		".*/stMemoryTest/mem32kb+33Filler.json",
		".*/stMemoryTest/mem32kbFiller.json",
		".*/stMemoryTest/mem33b_singleByteFiller.json",
		".*/stMemoryTest/mem64kb_singleByte-1Filler.json",
		".*/stMemoryTest/mem64kb_singleByte-31Filler.json",
		".*/stMemoryTest/mem64kb_singleByte-32Filler.json",
		".*/stMemoryTest/mem64kb_singleByte-33Filler.json",
		".*/stMemoryTest/mem64kb_singleByte+1Filler.json",
		".*/stMemoryTest/mem64kb_singleByte+31Filler.json",
		".*/stMemoryTest/mem64kb_singleByte+32Filler.json",
		".*/stMemoryTest/mem64kb_singleByte+33Filler.json",
		".*/stMemoryTest/mem64kb_singleByteFiller.json",
		".*/stMemoryTest/mem64kb-1Filler.json",
		".*/stMemoryTest/mem64kb-31Filler.json",
		".*/stMemoryTest/mem64kb-32Filler.json",
		".*/stMemoryTest/mem64kb-33Filler.json",
		".*/stMemoryTest/mem64kb+1Filler.json",
		".*/stMemoryTest/mem64kb+31Filler.json",
		".*/stMemoryTest/mem64kb+32Filler.json",
		".*/stMemoryTest/mem64kb+33Filler.json",
		".*/stMemoryTest/mem64kbFiller.json",
		".*/stMemoryTest/oogFiller.yml",
		".*/stNonZeroCallsTest/NonZeroValue_CALL_ToEmpty_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_CALL_ToOneStorageKey_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_CALLCODE_ToEmpty_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_CALLCODE_ToOneStorageKey_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_CALLCODEFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_CALLFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_DELEGATECALL_ToEmpty_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_DELEGATECALL_ToNonNonZeroBalanceFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_DELEGATECALL_ToOneStorageKey_ParisFiller.json",
		".*/stNonZeroCallsTest/NonZeroValue_DELEGATECALLFiller.json",
		".*/stPreCompiledContracts/precompsEIP2929CancunFiller.yml",
		".*/stPreCompiledContracts2/CallEcrecover_OverflowFiller.yml",
		".*/stPreCompiledContracts2/ecrecoverShortBuffFiller.yml",
		".*/stPreCompiledContracts2/modexp_0_0_0_22000Filler.json",
		".*/stPreCompiledContracts2/modexp_0_0_0_25000Filler.json",
		".*/stPreCompiledContracts2/modexp_0_0_0_35000Filler.json",
		".*/stQuadraticComplexityTest/Call20KbytesContract50_1Filler.json",
		".*/stQuadraticComplexityTest/Return50000_2Filler.json",
		".*/stQuadraticComplexityTest/Return50000Filler.json",
		".*/stRandom/randomStatetest100Filler.json",
		".*/stRandom/randomStatetest102Filler.json",
		".*/stRandom/randomStatetest104Filler.json",
		".*/stRandom/randomStatetest105Filler.json",
		".*/stRandom/randomStatetest106Filler.json",
		".*/stRandom/randomStatetest107Filler.json",
		".*/stRandom/randomStatetest110Filler.json",
		".*/stRandom/randomStatetest112Filler.json",
		".*/stRandom/randomStatetest114Filler.json",
		".*/stRandom/randomStatetest115Filler.json",
		".*/stRandom/randomStatetest116Filler.json",
		".*/stRandom/randomStatetest117Filler.json",
		".*/stRandom/randomStatetest118Filler.json",
		".*/stRandom/randomStatetest119Filler.json",
		".*/stRandom/randomStatetest11Filler.json",
		".*/stRandom/randomStatetest120Filler.json",
		".*/stRandom/randomStatetest121Filler.json",
		".*/stRandom/randomStatetest122Filler.json",
		".*/stRandom/randomStatetest124Filler.json",
		".*/stRandom/randomStatetest129Filler.json",
		".*/stRandom/randomStatetest12Filler.json",
		".*/stRandom/randomStatetest130Filler.json",
		".*/stRandom/randomStatetest131Filler.json",
		".*/stRandom/randomStatetest137Filler.json",
		".*/stRandom/randomStatetest138Filler.json",
		".*/stRandom/randomStatetest139Filler.json",
		".*/stRandom/randomStatetest142Filler.json",
		".*/stRandom/randomStatetest143Filler.json",
		".*/stRandom/randomStatetest145Filler.json",
		".*/stRandom/randomStatetest147Filler.json",
		".*/stRandom/randomStatetest148Filler.json",
		".*/stRandom/randomStatetest14Filler.json",
		".*/stRandom/randomStatetest153Filler.json",
		".*/stRandom/randomStatetest155Filler.json",
		".*/stRandom/randomStatetest156Filler.json",
		".*/stRandom/randomStatetest158Filler.json",
		".*/stRandom/randomStatetest15Filler.json",
		".*/stRandom/randomStatetest161Filler.json",
		".*/stRandom/randomStatetest162Filler.json",
		".*/stRandom/randomStatetest164Filler.json",
		".*/stRandom/randomStatetest166Filler.json",
		".*/stRandom/randomStatetest167Filler.json",
		".*/stRandom/randomStatetest169Filler.json",
		".*/stRandom/randomStatetest173Filler.json",
		".*/stRandom/randomStatetest174Filler.json",
		".*/stRandom/randomStatetest175Filler.json",
		".*/stRandom/randomStatetest179Filler.json",
		".*/stRandom/randomStatetest17Filler.json",
		".*/stRandom/randomStatetest180Filler.json",
		".*/stRandom/randomStatetest183Filler.json",
		".*/stRandom/randomStatetest184Filler.json",
		".*/stRandom/randomStatetest187Filler.json",
		".*/stRandom/randomStatetest188Filler.json",
		".*/stRandom/randomStatetest191Filler.json",
		".*/stRandom/randomStatetest192Filler.json",
		".*/stRandom/randomStatetest194Filler.json",
		".*/stRandom/randomStatetest195Filler.json",
		".*/stRandom/randomStatetest196Filler.json",
		".*/stRandom/randomStatetest198Filler.json",
		".*/stRandom/randomStatetest199Filler.json",
		".*/stRandom/randomStatetest19Filler.json",
		".*/stRandom/randomStatetest200Filler.json",
		".*/stRandom/randomStatetest201Filler.json",
		".*/stRandom/randomStatetest202Filler.json",
		".*/stRandom/randomStatetest204Filler.json",
		".*/stRandom/randomStatetest206Filler.json",
		".*/stRandom/randomStatetest207Filler.json",
		".*/stRandom/randomStatetest208Filler.json",
		".*/stRandom/randomStatetest210Filler.json",
		".*/stRandom/randomStatetest212Filler.json",
		".*/stRandom/randomStatetest214Filler.json",
		".*/stRandom/randomStatetest215Filler.json",
		".*/stRandom/randomStatetest216Filler.json",
		".*/stRandom/randomStatetest217Filler.json",
		".*/stRandom/randomStatetest219Filler.json",
		".*/stRandom/randomStatetest220Filler.json",
		".*/stRandom/randomStatetest221Filler.json",
		".*/stRandom/randomStatetest222Filler.json",
		".*/stRandom/randomStatetest225Filler.json",
		".*/stRandom/randomStatetest227Filler.json",
		".*/stRandom/randomStatetest228Filler.json",
		".*/stRandom/randomStatetest22Filler.json",
		".*/stRandom/randomStatetest231Filler.json",
		".*/stRandom/randomStatetest232Filler.json",
		".*/stRandom/randomStatetest236Filler.json",
		".*/stRandom/randomStatetest237Filler.json",
		".*/stRandom/randomStatetest238Filler.json",
		".*/stRandom/randomStatetest23Filler.json",
		".*/stRandom/randomStatetest242Filler.json",
		".*/stRandom/randomStatetest243Filler.json",
		".*/stRandom/randomStatetest244Filler.json",
		".*/stRandom/randomStatetest245Filler.json",
		".*/stRandom/randomStatetest246Filler.json",
		".*/stRandom/randomStatetest247Filler.json",
		".*/stRandom/randomStatetest248Filler.json",
		".*/stRandom/randomStatetest249Filler.json",
		".*/stRandom/randomStatetest254Filler.json",
		".*/stRandom/randomStatetest259Filler.json",
		".*/stRandom/randomStatetest264Filler.json",
		".*/stRandom/randomStatetest267Filler.json",
		".*/stRandom/randomStatetest268Filler.json",
		".*/stRandom/randomStatetest269Filler.json",
		".*/stRandom/randomStatetest26Filler.json",
		".*/stRandom/randomStatetest270Filler.json",
		".*/stRandom/randomStatetest273Filler.json",
		".*/stRandom/randomStatetest276Filler.json",
		".*/stRandom/randomStatetest278Filler.json",
		".*/stRandom/randomStatetest279Filler.json",
		".*/stRandom/randomStatetest27Filler.json",
		".*/stRandom/randomStatetest280Filler.json",
		".*/stRandom/randomStatetest281Filler.json",
		".*/stRandom/randomStatetest283Filler.json",
		".*/stRandom/randomStatetest28Filler.json",
		".*/stRandom/randomStatetest290Filler.json",
		".*/stRandom/randomStatetest291Filler.json",
		".*/stRandom/randomStatetest293Filler.json",
		".*/stRandom/randomStatetest297Filler.json",
		".*/stRandom/randomStatetest298Filler.json",
		".*/stRandom/randomStatetest299Filler.json",
		".*/stRandom/randomStatetest29Filler.json",
		".*/stRandom/randomStatetest2Filler.json",
		".*/stRandom/randomStatetest301Filler.json",
		".*/stRandom/randomStatetest305Filler.json",
		".*/stRandom/randomStatetest30Filler.json",
		".*/stRandom/randomStatetest310Filler.json",
		".*/stRandom/randomStatetest311Filler.json",
		".*/stRandom/randomStatetest315Filler.json",
		".*/stRandom/randomStatetest316Filler.json",
		".*/stRandom/randomStatetest318Filler.json",
		".*/stRandom/randomStatetest31Filler.json",
		".*/stRandom/randomStatetest322Filler.json",
		".*/stRandom/randomStatetest325Filler.json",
		".*/stRandom/randomStatetest329Filler.json",
		".*/stRandom/randomStatetest332Filler.json",
		".*/stRandom/randomStatetest333Filler.json",
		".*/stRandom/randomStatetest334Filler.json",
		".*/stRandom/randomStatetest337Filler.json",
		".*/stRandom/randomStatetest338Filler.json",
		".*/stRandom/randomStatetest339Filler.json",
		".*/stRandom/randomStatetest342Filler.json",
		".*/stRandom/randomStatetest343Filler.json",
		".*/stRandom/randomStatetest348Filler.json",
		".*/stRandom/randomStatetest349Filler.json",
		".*/stRandom/randomStatetest351Filler.json",
		".*/stRandom/randomStatetest354Filler.json",
		".*/stRandom/randomStatetest356Filler.json",
		".*/stRandom/randomStatetest358Filler.json",
		".*/stRandom/randomStatetest360Filler.json",
		".*/stRandom/randomStatetest361Filler.json",
		".*/stRandom/randomStatetest362Filler.json",
		".*/stRandom/randomStatetest363Filler.json",
		".*/stRandom/randomStatetest364Filler.json",
		".*/stRandom/randomStatetest365Filler.json",
		".*/stRandom/randomStatetest366Filler.json",
		".*/stRandom/randomStatetest367Filler.json",
		".*/stRandom/randomStatetest368Filler.json",
		".*/stRandom/randomStatetest369Filler.json",
		".*/stRandom/randomStatetest371Filler.json",
		".*/stRandom/randomStatetest372Filler.json",
		".*/stRandom/randomStatetest376Filler.json",
		".*/stRandom/randomStatetest379Filler.json",
		".*/stRandom/randomStatetest37Filler.json",
		".*/stRandom/randomStatetest380Filler.json",
		".*/stRandom/randomStatetest381Filler.json",
		".*/stRandom/randomStatetest382Filler.json",
		".*/stRandom/randomStatetest383Filler.json",
		".*/stRandom/randomStatetest39Filler.json",
		".*/stRandom/randomStatetest3Filler.json",
		".*/stRandom/randomStatetest41Filler.json",
		".*/stRandom/randomStatetest43Filler.json",
		".*/stRandom/randomStatetest47Filler.json",
		".*/stRandom/randomStatetest49Filler.json",
		".*/stRandom/randomStatetest52Filler.json",
		".*/stRandom/randomStatetest58Filler.json",
		".*/stRandom/randomStatetest59Filler.json",
		".*/stRandom/randomStatetest60Filler.json",
		".*/stRandom/randomStatetest62Filler.json",
		".*/stRandom/randomStatetest63Filler.json",
		".*/stRandom/randomStatetest64Filler.json",
		".*/stRandom/randomStatetest66Filler.json",
		".*/stRandom/randomStatetest67Filler.json",
		".*/stRandom/randomStatetest69Filler.json",
		".*/stRandom/randomStatetest6Filler.json",
		".*/stRandom/randomStatetest73Filler.json",
		".*/stRandom/randomStatetest74Filler.json",
		".*/stRandom/randomStatetest75Filler.json",
		".*/stRandom/randomStatetest77Filler.json",
		".*/stRandom/randomStatetest80Filler.json",
		".*/stRandom/randomStatetest81Filler.json",
		".*/stRandom/randomStatetest83Filler.json",
		".*/stRandom/randomStatetest85Filler.json",
		".*/stRandom/randomStatetest87Filler.json",
		".*/stRandom/randomStatetest88Filler.json",
		".*/stRandom/randomStatetest89Filler.json",
		".*/stRandom/randomStatetest90Filler.json",
		".*/stRandom/randomStatetest92Filler.json",
		".*/stRandom/randomStatetest95Filler.json",
		".*/stRandom/randomStatetest96Filler.json",
		".*/stRandom/randomStatetest98Filler.json",
		".*/stRandom/randomStatetest9Filler.json",
		".*/stRandom2/randomStatetest384Filler.json",
		".*/stRandom2/randomStatetest385Filler.json",
		".*/stRandom2/randomStatetest386Filler.json",
		".*/stRandom2/randomStatetest388Filler.json",
		".*/stRandom2/randomStatetest389Filler.json",
		".*/stRandom2/randomStatetest395Filler.json",
		".*/stRandom2/randomStatetest398Filler.json",
		".*/stRandom2/randomStatetest399Filler.json",
		".*/stRandom2/randomStatetest402Filler.json",
		".*/stRandom2/randomStatetest405Filler.json",
		".*/stRandom2/randomStatetest406Filler.json",
		".*/stRandom2/randomStatetest407Filler.json",
		".*/stRandom2/randomStatetest408Filler.json",
		".*/stRandom2/randomStatetest409Filler.json",
		".*/stRandom2/randomStatetest411Filler.json",
		".*/stRandom2/randomStatetest412Filler.json",
		".*/stRandom2/randomStatetest413Filler.json",
		".*/stRandom2/randomStatetest416Filler.json",
		".*/stRandom2/randomStatetest419Filler.json",
		".*/stRandom2/randomStatetest421Filler.json",
		".*/stRandom2/randomStatetest424Filler.json",
		".*/stRandom2/randomStatetest425Filler.json",
		".*/stRandom2/randomStatetest426Filler.json",
		".*/stRandom2/randomStatetest429Filler.json",
		".*/stRandom2/randomStatetest430Filler.json",
		".*/stRandom2/randomStatetest435Filler.json",
		".*/stRandom2/randomStatetest436Filler.json",
		".*/stRandom2/randomStatetest437Filler.json",
		".*/stRandom2/randomStatetest438Filler.json",
		".*/stRandom2/randomStatetest439Filler.json",
		".*/stRandom2/randomStatetest440Filler.json",
		".*/stRandom2/randomStatetest442Filler.json",
		".*/stRandom2/randomStatetest446Filler.json",
		".*/stRandom2/randomStatetest447Filler.json",
		".*/stRandom2/randomStatetest450Filler.json",
		".*/stRandom2/randomStatetest451Filler.json",
		".*/stRandom2/randomStatetest452Filler.json",
		".*/stRandom2/randomStatetest455Filler.json",
		".*/stRandom2/randomStatetest457Filler.json",
		".*/stRandom2/randomStatetest460Filler.json",
		".*/stRandom2/randomStatetest461Filler.json",
		".*/stRandom2/randomStatetest462Filler.json",
		".*/stRandom2/randomStatetest464Filler.json",
		".*/stRandom2/randomStatetest465Filler.json",
		".*/stRandom2/randomStatetest466Filler.json",
		".*/stRandom2/randomStatetest470Filler.json",
		".*/stRandom2/randomStatetest471Filler.json",
		".*/stRandom2/randomStatetest473Filler.json",
		".*/stRandom2/randomStatetest474Filler.json",
		".*/stRandom2/randomStatetest475Filler.json",
		".*/stRandom2/randomStatetest477Filler.json",
		".*/stRandom2/randomStatetest480Filler.json",
		".*/stRandom2/randomStatetest482Filler.json",
		".*/stRandom2/randomStatetest483Filler.json",
		".*/stRandom2/randomStatetest487Filler.json",
		".*/stRandom2/randomStatetest488Filler.json",
		".*/stRandom2/randomStatetest489Filler.json",
		".*/stRandom2/randomStatetest491Filler.json",
		".*/stRandom2/randomStatetest493Filler.json",
		".*/stRandom2/randomStatetest495Filler.json",
		".*/stRandom2/randomStatetest497Filler.json",
		".*/stRandom2/randomStatetest500Filler.json",
		".*/stRandom2/randomStatetest501Filler.json",
		".*/stRandom2/randomStatetest502Filler.json",
		".*/stRandom2/randomStatetest503Filler.json",
		".*/stRandom2/randomStatetest505Filler.json",
		".*/stRandom2/randomStatetest506Filler.json",
		".*/stRandom2/randomStatetest511Filler.json",
		".*/stRandom2/randomStatetest512Filler.json",
		".*/stRandom2/randomStatetest514Filler.json",
		".*/stRandom2/randomStatetest516Filler.json",
		".*/stRandom2/randomStatetest517Filler.json",
		".*/stRandom2/randomStatetest518Filler.json",
		".*/stRandom2/randomStatetest519Filler.json",
		".*/stRandom2/randomStatetest520Filler.json",
		".*/stRandom2/randomStatetest521Filler.json",
		".*/stRandom2/randomStatetest526Filler.json",
		".*/stRandom2/randomStatetest532Filler.json",
		".*/stRandom2/randomStatetest533Filler.json",
		".*/stRandom2/randomStatetest534Filler.json",
		".*/stRandom2/randomStatetest535Filler.json",
		".*/stRandom2/randomStatetest537Filler.json",
		".*/stRandom2/randomStatetest539Filler.json",
		".*/stRandom2/randomStatetest541Filler.json",
		".*/stRandom2/randomStatetest542Filler.json",
		".*/stRandom2/randomStatetest544Filler.json",
		".*/stRandom2/randomStatetest545Filler.json",
		".*/stRandom2/randomStatetest546Filler.json",
		".*/stRandom2/randomStatetest548Filler.json",
		".*/stRandom2/randomStatetest550Filler.json",
		".*/stRandom2/randomStatetest552Filler.json",
		".*/stRandom2/randomStatetest553Filler.json",
		".*/stRandom2/randomStatetest555Filler.json",
		".*/stRandom2/randomStatetest556Filler.json",
		".*/stRandom2/randomStatetest559Filler.json",
		".*/stRandom2/randomStatetest564Filler.json",
		".*/stRandom2/randomStatetest565Filler.json",
		".*/stRandom2/randomStatetest571Filler.json",
		".*/stRandom2/randomStatetest574Filler.json",
		".*/stRandom2/randomStatetest577Filler.json",
		".*/stRandom2/randomStatetest578Filler.json",
		".*/stRandom2/randomStatetest580Filler.json",
		".*/stRandom2/randomStatetest581Filler.json",
		".*/stRandom2/randomStatetest584Filler.json",
		".*/stRandom2/randomStatetest585Filler.json",
		".*/stRandom2/randomStatetest586Filler.json",
		".*/stRandom2/randomStatetest587Filler.json",
		".*/stRandom2/randomStatetest588Filler.json",
		".*/stRandom2/randomStatetest592Filler.json",
		".*/stRandom2/randomStatetest596Filler.json",
		".*/stRandom2/randomStatetest599Filler.json",
		".*/stRandom2/randomStatetest600Filler.json",
		".*/stRandom2/randomStatetest602Filler.json",
		".*/stRandom2/randomStatetest603Filler.json",
		".*/stRandom2/randomStatetest605Filler.json",
		".*/stRandom2/randomStatetest607Filler.json",
		".*/stRandom2/randomStatetest608Filler.json",
		".*/stRandom2/randomStatetest610Filler.json",
		".*/stRandom2/randomStatetest612Filler.json",
		".*/stRandom2/randomStatetest615Filler.json",
		".*/stRandom2/randomStatetest616Filler.json",
		".*/stRandom2/randomStatetest620Filler.json",
		".*/stRandom2/randomStatetest621Filler.json",
		".*/stRandom2/randomStatetest627Filler.json",
		".*/stRandom2/randomStatetest628Filler.json",
		".*/stRandom2/randomStatetest629Filler.json",
		".*/stRandom2/randomStatetest630Filler.json",
		".*/stRandom2/randomStatetest633Filler.json",
		".*/stRandom2/randomStatetest635Filler.json",
		".*/stRandom2/randomStatetest637Filler.json",
		".*/stRandom2/randomStatetest638Filler.json",
		".*/stRandom2/randomStatetest641Filler.json",
		".*/stRandom2/randomStatetest643Filler.json",
		".*/stRandom2/randomStatetestFiller.json",
		".*/stRefundTest/refund_CallAFiller.json",
		".*/stRefundTest/refund_TxToSuicideFiller.json",
		".*/stRefundTest/refund50_2Filler.json",
		".*/stRefundTest/refund50percentCapFiller.json",
		".*/stRefundTest/refund600Filler.json",
		".*/stRefundTest/refundSuicide50procentCapFiller.json",
		".*/stReturnDataTest/call_outsize_then_create_successful_then_returndatasizeFiller.json",
		".*/stReturnDataTest/call_then_create_successful_then_returndatasizeFiller.json",
		".*/stReturnDataTest/create_callprecompile_returndatasizeFiller.json",
		".*/stReturnDataTest/modexp_modsize0_returndatasizeFiller.json",
		".*/stReturnDataTest/returndatacopy_0_0_following_successful_createFiller.json",
		".*/stReturnDataTest/returndatacopy_afterFailing_createFiller.json",
		".*/stReturnDataTest/returndatacopy_following_revert_in_createFiller.json",
		".*/stReturnDataTest/returndatasize_after_successful_callcodeFiller.json",
		".*/stReturnDataTest/returndatasize_following_successful_createFiller.json",
		".*/stReturnDataTest/tooLongReturnDataCopyFiller.yml",
		".*/stRevertTest/RevertDepth2Filler.json",
		".*/stRevertTest/RevertDepthCreateAddressCollisionFiller.json",
		".*/stRevertTest/RevertDepthCreateOOGFiller.json",
		".*/stRevertTest/RevertInCreateInInit_ParisFiller.json",
		".*/stRevertTest/RevertOpcodeCallsFiller.json",
		".*/stRevertTest/RevertOpcodeCreateFiller.json",
		".*/stRevertTest/RevertOpcodeDirectCallFiller.json",
		".*/stRevertTest/RevertOpcodeInCreateReturnsFiller.json",
		".*/stRevertTest/RevertOpcodeMultipleSubCallsFiller.json",
		".*/stRevertTest/RevertSubCallStorageOOG2Filler.json",
		".*/stRevertTest/RevertSubCallStorageOOGFiller.json",
		".*/stSelfBalance/selfBalanceCallTypesFiller.json",
		".*/stSelfBalance/selfBalanceEqualsBalanceFiller.json",
		".*/stSelfBalance/selfBalanceFiller.json",
		".*/stSelfBalance/selfBalanceGasCostFiller.json",
		".*/stSelfBalance/selfBalanceUpdateFiller.json",
		".*/stSLoadTest/sloadGasCostFiller.json",
		".*/stSolidityTest/CallLowLevelCreatesSolidityFiller.json",
		".*/stSolidityTest/RecursiveCreateContractsCreate4ContractsFiller.json",
		".*/stSolidityTest/TestOverflowFiller.json",
		".*/stSolidityTest/TestStructuresAndVariablessFiller.json",
		".*/stSpecialTest/deploymentErrorFiller.json",
		".*/stSpecialTest/FailedCreateRevertsDeletionParisFiller.json",
		".*/stSpecialTest/makeMoneyFiller.json",
		".*/stSpecialTest/selfdestructEIP2929Filler.json",
		".*/stSStoreTest/sstore_0to0Filler.json",
		".*/stSStoreTest/sstore_0to0to0Filler.json",
		".*/stSStoreTest/sstore_0to0toXFiller.json",
		".*/stSStoreTest/sstore_0toXFiller.json",
		".*/stSStoreTest/sstore_0toXto0Filler.json",
		".*/stSStoreTest/sstore_0toXto0toXFiller.json",
		".*/stSStoreTest/sstore_0toXtoXFiller.json",
		".*/stSStoreTest/sstore_0toXtoYFiller.json",
		".*/stSStoreTest/sstore_Xto0Filler.json",
		".*/stSStoreTest/sstore_Xto0to0Filler.json",
		".*/stSStoreTest/sstore_Xto0toXFiller.json",
		".*/stSStoreTest/sstore_Xto0toXto0Filler.json",
		".*/stSStoreTest/sstore_Xto0toYFiller.json",
		".*/stSStoreTest/sstore_XtoXFiller.json",
		".*/stSStoreTest/sstore_XtoXto0Filler.json",
		".*/stSStoreTest/sstore_XtoXtoXFiller.json",
		".*/stSStoreTest/sstore_XtoXtoYFiller.json",
		".*/stSStoreTest/sstore_XtoYFiller.json",
		".*/stSStoreTest/sstore_XtoYto0Filler.json",
		".*/stSStoreTest/sstore_XtoYtoXFiller.json",
		".*/stSStoreTest/sstore_XtoYtoYFiller.json",
		".*/stSStoreTest/sstore_XtoYtoZFiller.json",
		".*/stSStoreTest/sstoreGasFiller.yml",
		".*/stStackTests/shallowStackFiller.json",
		".*/stStackTests/stackOverflowDUPFiller.json",
		".*/stStackTests/stackOverflowFiller.json",
		".*/stStackTests/stackOverflowM1DUPFiller.json",
		".*/stStackTests/stackOverflowM1Filler.json",
		".*/stStackTests/stackOverflowM1PUSHFiller.json",
		".*/stStackTests/stackOverflowPUSHFiller.json",
		".*/stStackTests/stackOverflowSWAPFiller.json",
		".*/stStackTests/stacksanitySWAPFiller.json",
		".*/stStaticCall/static_ABAcalls3Filler.json",
		".*/stStaticCall/static_Call1024OOGFiller.json",
		".*/stStaticCall/static_Call10Filler.json",
		".*/stStaticCall/static_callcallcodecall_ABCB_RECURSIVE2Filler.json",
		".*/stStaticCall/static_callcallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stStaticCall/static_callcallcodecallcode_ABCB_RECURSIVE2Filler.json",
		".*/stStaticCall/static_callcallcodecallcode_ABCB_RECURSIVEFiller.json",
		".*/stStaticCall/static_callcode_checkPCFiller.json",
		".*/stStaticCall/static_callcodecallcall_ABCB_RECURSIVE2Filler.json",
		".*/stStaticCall/static_callcodecallcall_ABCB_RECURSIVEFiller.json",
		".*/stStaticCall/static_callcodecallcallcode_ABCB_RECURSIVE2Filler.json",
		".*/stStaticCall/static_callcodecallcallcode_ABCB_RECURSIVEFiller.json",
		".*/stStaticCall/static_callcodecallcodecall_110_SuicideEnd2Filler.json",
		".*/stStaticCall/static_callcodecallcodecall_110_SuicideEndFiller.json",
		".*/stStaticCall/static_callcodecallcodecall_ABCB_RECURSIVE2Filler.json",
		".*/stStaticCall/static_callcodecallcodecall_ABCB_RECURSIVEFiller.json",
		".*/stStaticCall/static_CallContractToCreateContractOOGFiller.json",
		".*/stStaticCall/static_CallContractToCreateContractWhichWouldCreateContractIfCalledFiller.json",
		".*/stStaticCall/static_CallLoseGasOOGFiller.json",
		".*/stStaticCall/static_CheckOpcodes5Filler.json",
		".*/stStaticCall/static_contractCreationMakeCallThatAskMoreGasThenTransactionProvidedFiller.json",
		".*/stStaticCall/static_CREATE_EmptyContractAndCallIt_0weiFiller.json",
		".*/stStaticCall/static_CREATE_EmptyContractWithStorageAndCallIt_0weiFiller.json",
		".*/stStaticCall/static_RETURN_BoundsFiller.json",
		".*/stStaticCall/static_RETURN_BoundsOOGFiller.json",
		".*/stStaticCall/static_ReturnTest2Filler.json",
		".*/stSystemOperationsTest/ABAcalls3Filler.json",
		".*/stSystemOperationsTest/Call10Filler.json",
		".*/stSystemOperationsTest/callcodeToNameRegistratorZeroMemExpanionFiller.json",
		".*/stSystemOperationsTest/CallRecursiveBomb3Filler.json",
		".*/stSystemOperationsTest/CallToNameRegistratorZeorSizeMemExpansionFiller.json",
		".*/stSystemOperationsTest/doubleSelfdestructTestFiller.yml",
		".*/stSystemOperationsTest/extcodecopyFiller.json",
		".*/stSystemOperationsTest/multiSelfdestructFiller.yml",
		".*/stTransactionTest/CreateMessageSuccessFiller.json",
		".*/stTransactionTest/CreateTransactionSuccessFiller.json",
		".*/stTransactionTest/InternalCallHittingGasLimit2Filler.json",
		".*/stTransactionTest/StoreGasOnCreateFiller.json",
		".*/stTransactionTest/SuicidesAndInternalCallSuicidesOOGFiller.json",
		".*/stTransitionTest/createNameRegistratorPerTxsAfterFiller.json",
		".*/stTransitionTest/createNameRegistratorPerTxsAtFiller.json",
		".*/stTransitionTest/createNameRegistratorPerTxsBeforeFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALL_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALL_ToEmpty_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALL_ToNonZeroBalance_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALL_ToOneStorageKey_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALLCODE_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALLCODE_ToEmpty_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALLCODE_ToNonZeroBalance_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_CALLCODE_ToOneStorageKey_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_DELEGATECALL_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_DELEGATECALL_ToEmpty_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_DELEGATECALL_ToNonZeroBalance_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_DELEGATECALL_ToOneStorageKey_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_SUICIDE_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_SUICIDE_ToEmpty_OOGRevert_ParisFiller.json",
		".*/stZeroCallsRevert/ZeroValue_SUICIDE_ToNonZeroBalance_OOGRevertFiller.json",
		".*/stZeroCallsRevert/ZeroValue_SUICIDE_ToOneStorageKey_OOGRevert_ParisFiller.json",
		".*/stZeroKnowledge/pointAddFiller.json",
		".*/stZeroKnowledge/pointAddTruncFiller.json",
		".*/stZeroKnowledge/pointMulAdd2Filler.json",
		".*/stZeroKnowledge/pointMulAddFiller.json",
		".*/VMTests/vmArithmeticTest/twoOpsFiller.yml",
	}
	testExecutionSpecBlocktests(t, executionSpecBALBlockchainTestDir, skips)
}

// TestExecutionSpecZkevmBlocktests runs the zkevm test fixtures from execution-spec-tests
// and validates execution witnesses against geth's stateless execution.
func TestExecutionSpecZkevmBlocktests(t *testing.T) {
	if !common.FileExist(executionSpecZkevmBlockchainTestDir) {
		t.Skipf("directory %s does not exist", executionSpecZkevmBlockchainTestDir)
	}
	bt := new(testMatcher)

	bt.walk(t, executionSpecZkevmBlockchainTestDir, func(t *testing.T, name string, test *BlockTest) {
		execBlockTest(t, bt, test, true)

		// Validate execution witnesses for blocks that have them
		// TODO: Execution specs don't emit a witness when block is valid right now
		for _, b := range test.json.Blocks {
			if b.ExecutionWitness == nil || b.BlockHeader == nil {
				continue
			}
			block, err := b.decode()
			if err != nil {
				t.Fatalf("failed to decode block for witness validation: %v", err)
			}
			if err := test.validateExecutionWitness(block, b.ExecutionWitness); err != nil {
				t.Errorf("execution witness validation failed: %v", err)
			}

			// Validate that the SSZ-encoded statelessInputBytes witness matches the JSON executionWitness
			// TODO: long term, we will only have statelessInputBytes and we will have no redundancy
			if b.StatelessInputBytes != nil {
				if err := validateStatelessInputWitness([]byte(*b.StatelessInputBytes), b.ExecutionWitness); err != nil {
					t.Errorf("stateless input witness mismatch: %v", err)
				}
			}
		}
	})
}

var failures = 0

func execBlockTest(t *testing.T, bt *testMatcher, test *BlockTest, buildAndVerifyBAL bool) {
	// Define all the different flag combinations we should run the tests with,
	// picking only one for short tests.
	//
	// Note, witness building and self-testing is always enabled as it's a very
	// good test to ensure that we don't break it.
	var (
		snapshotConf = []bool{false, true}
		dbschemeConf = []string{rawdb.HashScheme, rawdb.PathScheme}
	)
	if testing.Short() {
		snapshotConf = []bool{snapshotConf[rand.Int()%2]}
		dbschemeConf = []string{dbschemeConf[rand.Int()%2]}
	}
	for _, snapshot := range snapshotConf {
		for _, dbscheme := range dbschemeConf {
			if err := bt.checkFailure(t, test.Run(snapshot, dbscheme, true, buildAndVerifyBAL, nil, nil)); err != nil {
				failures++
				/*
					if failures > 10 {
						panic("adsf")
					}
				*/
				t.Errorf("test with config {snapshotter:%v, scheme:%v} failed: %v", snapshot, dbscheme, err)
				return
			}
		}
	}
}
