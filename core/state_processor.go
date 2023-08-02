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

package core

import (
	"fmt"

	"math/big"
	"runtime"
	"strings"
	"sync"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/log"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/params"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards
}
type CalculatedBlock struct {
	block *types.Block
	stop  bool
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
		engine: engine,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, cfg vm.Config, balanceFee map[common.Address]*big.Int) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	if common.TIPSigning.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(common.HexToAddress(common.BlockSigners))
	}
	parentState := statedb.Copy()
	InitSignerInTransactions(p.config, header, block.Transactions())
	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)
	for i, tx := range block.Transactions() {
		// check black-list txs after hf
		if (block.Number().Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			// check if sender is in black list
			if tx.From() != nil && common.Blacklist[*tx.From()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with sender in black-list: %v", tx.From().Hex())
			}
			// check if receiver is in black list
			if tx.To() != nil && common.Blacklist[*tx.To()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with receiver in black-list: %v", tx.To().Hex())
			}
		}
		// validate minFee slot for XDCZ
		if tx.IsXDCZApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateXDCZApplyTransaction(p.bc, block.Number(), copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}
		// validate balance slot, token decimal for XDCX
		if tx.IsXDCXApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateXDCXApplyTransaction(p.bc, block.Number(), copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, gas, err, tokenFeeUsed := ApplyTransaction(p.config, balanceFee, p.bc, nil, gp, statedb, tradingState, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
		if tokenFeeUsed {
			fee := new(big.Int).SetUint64(gas)
			if block.Header().Number.Cmp(common.TIPTRC21Fee) > 0 {
				fee = fee.Mul(fee, common.TRC21GasPrice)
			}
			balanceFee[*tx.To()] = new(big.Int).Sub(balanceFee[*tx.To()], fee)
			balanceUpdated[*tx.To()] = balanceFee[*tx.To()]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	state.UpdateTRC21Fee(statedb, balanceUpdated, totalFeeUsed)
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, parentState, block.Transactions(), block.Uncles(), receipts)
	return receipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) ProcessBlockNoValidator(cBlock *CalculatedBlock, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, cfg vm.Config, balanceFee map[common.Address]*big.Int) (types.Receipts, []*types.Log, uint64, error) {
	block := cBlock.block
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	if common.TIPSigning.Cmp(header.Number) == 0 {
		statedb.DeleteAddress(common.HexToAddress(common.BlockSigners))
	}
	if cBlock.stop {
		return nil, nil, 0, ErrStopPreparingBlock
	}
	parentState := statedb.Copy()
	InitSignerInTransactions(p.config, header, block.Transactions())
	balanceUpdated := map[common.Address]*big.Int{}
	totalFeeUsed := big.NewInt(0)

	if cBlock.stop {
		return nil, nil, 0, ErrStopPreparingBlock
	}
	// Iterate over and process the individual transactions
	receipts = make([]*types.Receipt, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		// check black-list txs after hf
		if (block.Number().Uint64() >= common.BlackListHFNumber) && !common.IsTestnet {
			// check if sender is in black list
			if tx.From() != nil && common.Blacklist[*tx.From()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with sender in black-list: %v", tx.From().Hex())
			}
			// check if receiver is in black list
			if tx.To() != nil && common.Blacklist[*tx.To()] {
				return nil, nil, 0, fmt.Errorf("Block contains transaction with receiver in black-list: %v", tx.To().Hex())
			}
		}
		// validate minFee slot for XDCZ
		if tx.IsXDCZApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateXDCZApplyTransaction(p.bc, block.Number(), copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}
		// validate balance slot, token decimal for XDCX
		if tx.IsXDCXApplyTransaction() {
			copyState := statedb.Copy()
			if err := ValidateXDCXApplyTransaction(p.bc, block.Number(), copyState, common.BytesToAddress(tx.Data()[4:])); err != nil {
				return nil, nil, 0, err
			}
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)
		receipt, gas, err, tokenFeeUsed := ApplyTransaction(p.config, balanceFee, p.bc, nil, gp, statedb, tradingState, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		if cBlock.stop {
			return nil, nil, 0, ErrStopPreparingBlock
		}
		receipts[i] = receipt
		allLogs = append(allLogs, receipt.Logs...)
		if tokenFeeUsed {
			fee := new(big.Int).SetUint64(gas)
			if block.Header().Number.Cmp(common.TIPTRC21Fee) > 0 {
				fee = fee.Mul(fee, common.TRC21GasPrice)
			}
			balanceFee[*tx.To()] = new(big.Int).Sub(balanceFee[*tx.To()], fee)
			balanceUpdated[*tx.To()] = balanceFee[*tx.To()]
			totalFeeUsed = totalFeeUsed.Add(totalFeeUsed, fee)
		}
	}
	state.UpdateTRC21Fee(statedb, balanceUpdated, totalFeeUsed)
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, parentState, block.Transactions(), block.Uncles(), receipts)
	return receipts, allLogs, *usedGas, nil
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *params.ChainConfig, tokensFee map[common.Address]*big.Int, bc *BlockChain, author *common.Address, gp *GasPool, statedb *state.StateDB, XDCxState *tradingstate.TradingStateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error, bool) {
	if tx.To() != nil && tx.To().String() == common.BlockSigners && config.IsTIPSigning(header.Number) {
		return ApplySignTransaction(config, statedb, header, tx, usedGas)
	}
	if tx.To() != nil && tx.To().String() == common.TradingStateAddr && config.IsTIPXDCX(header.Number) {
		return ApplyEmptyTransaction(config, statedb, header, tx, usedGas)
	}
	if tx.To() != nil && tx.To().String() == common.XDCXLendingAddress && config.IsTIPXDCX(header.Number) {
		return ApplyEmptyTransaction(config, statedb, header, tx, usedGas)
	}
	if tx.IsTradingTransaction() && config.IsTIPXDCX(header.Number) {
		return ApplyEmptyTransaction(config, statedb, header, tx, usedGas)
	}

	if tx.IsLendingFinalizedTradeTransaction() && config.IsTIPXDCX(header.Number) {
		return ApplyEmptyTransaction(config, statedb, header, tx, usedGas)
	}

	var balanceFee *big.Int
	if tx.To() != nil {
		if value, ok := tokensFee[*tx.To()]; ok {
			balanceFee = value
		}
	}
	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number), balanceFee, header.Number)
	if err != nil {
		return nil, 0, err, false
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, XDCxState, config, cfg)

	// If we don't have an explicit author (i.e. not mining), extract from the header
	var beneficiary common.Address
	if author == nil {
		beneficiary, _ = bc.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}

	coinbaseOwner := statedb.GetOwner(beneficiary)

	// Bypass blacklist address
	maxBlockNumber := new(big.Int).SetInt64(9147459)
	if header.Number.Cmp(maxBlockNumber) <= 0 {
		addrMap := make(map[string]string)
		addrMap["0x5248bfb72fd4f234e062d3e9bb76f08643004fcd"] = "29410"
		addrMap["0x5ac26105b35ea8935be382863a70281ec7a985e9"] = "23551"
		addrMap["0x09c4f991a41e7ca0645d7dfbfee160b55e562ea4"] = "25821"
		addrMap["0xb3157bbc5b401a45d6f60b106728bb82ebaa585b"] = "20051"
		addrMap["0x741277a8952128d5c2ffe0550f5001e4c8247674"] = "23937"
		addrMap["0x10ba49c1caa97d74b22b3e74493032b180cebe01"] = "27320"
		addrMap["0x07048d51d9e6179578a6e3b9ee28cdc183b865e4"] = "29758"
		addrMap["0x4b899001d73c7b4ec404a771d37d9be13b8983de"] = "26148"
		addrMap["0x85cb320a9007f26b7652c19a2a65db1da2d0016f"] = "27216"
		addrMap["0x06869dbd0e3a2ea37ddef832e20fa005c6f0ca39"] = "29449"
		addrMap["0x82e48bc7e2c93d89125428578fb405947764ad7c"] = "28084"
		addrMap["0x1f9a78534d61732367cbb43fc6c89266af67c989"] = "29287"
		addrMap["0x7c3b1fa91df55ff7af0cad9e0399384dc5c6641b"] = "21574"
		addrMap["0x5888dc1ceb0ff632713486b9418e59743af0fd20"] = "28836"
		addrMap["0xa512fa1c735fc3cc635624d591dd9ea1ce339ca5"] = "25515"
		addrMap["0x0832517654c7b7e36b1ef45d76de70326b09e2c7"] = "22873"
		addrMap["0xca14e3c4c78bafb60819a78ff6e6f0f709d2aea7"] = "24968"
		addrMap["0x652ce195a23035114849f7642b0e06647d13e57a"] = "24091"
		addrMap["0x29a79f00f16900999d61b6e171e44596af4fb5ae"] = "20790"
		addrMap["0xf9fd1c2b0af0d91b0b6754e55639e3f8478dd04a"] = "23331"
		addrMap["0xb835710c9901d5fe940ef1b99ed918902e293e35"] = "28273"
		addrMap["0x04dd29ce5c253377a9a3796103ea0d9a9e514153"] = "29956"
		addrMap["0x2b4b56846eaf05c1fd762b5e1ac802efd0ab871c"] = "24911"
		addrMap["0x1d1f909f6600b23ce05004f5500ab98564717996"] = "25637"
		addrMap["0x0dfdcebf80006dc9ab7aae8c216b51c6b6759e86"] = "26378"
		addrMap["0x2b373890a28e5e46197fbc04f303bbfdd344056f"] = "21109"
		addrMap["0xa8a3ef3dc5d8e36aee76f3671ec501ec31e28254"] = "22072"
		addrMap["0x4f3d18136fe2b5665c29bdaf74591fc6625ef427"] = "21650"
		addrMap["0x175d728b0e0f1facb5822a2e0c03bde93596e324"] = "21588"
		addrMap["0xd575c2611984fcd79513b80ab94f59dc5bab4916"] = "28971"
		addrMap["0x0579337873c97c4ba051310236ea847f5be41bc0"] = "28344"
		addrMap["0xed12a519cc15b286920fc15fd86106b3e6a16218"] = "24443"
		addrMap["0x492d26d852a0a0a2982bb40ec86fe394488c419e"] = "26623"
		addrMap["0xce5c7635d02dc4e1d6b46c256cae6323be294a32"] = "28459"
		addrMap["0x8b94db158b5e78a6c032c7e7c9423dec62c8b11c"] = "21803"
		addrMap["0x0e7c48c085b6b0aa7ca6e4cbcc8b9a92dc270eb4"] = "21739"
		addrMap["0x206e6508462033ef8425edc6c10789d241d49acb"] = "21883"
		addrMap["0x7710e7b7682f26cb5a1202e1cff094fbf7777758"] = "28907"
		addrMap["0xcb06f949313b46bbf53b8e6b2868a0c260ff9385"] = "28932"
		addrMap["0xf884e43533f61dc2997c0e19a6eff33481920c00"] = "27780"
		addrMap["0x8b635ef2e4c8fe21fc2bda027eb5f371d6aa2fc1"] = "23115"
		addrMap["0x10f01a27cf9b29d02ce53497312b96037357a361"] = "22716"
		addrMap["0x693dd49b0ed70f162d733cf20b6c43dc2a2b4d95"] = "20020"
		addrMap["0xe0bec72d1c2a7a7fb0532cdfac44ebab9f6f41ee"] = "23071"
		addrMap["0xc8793633a537938cb49cdbbffd45428f10e45b64"] = "24652"
		addrMap["0x0d07a6cbbe9fa5c4f154e5623bfe47fb4d857d8e"] = "21907"
		addrMap["0xd4080b289da95f70a586610c38268d8d4cf1e4c4"] = "22719"
		addrMap["0x8bcfb0caf41f0aa1b548cae76dcdd02e33866a1b"] = "29062"
		addrMap["0xabfef22b92366d3074676e77ea911ccaabfb64c1"] = "23110"
		addrMap["0xcc4df7a32faf3efba32c9688def5ccf9fefe443d"] = "21397"
		addrMap["0x7ec1e48a582475f5f2b7448a86c4ea7a26ea36f8"] = "23105"
		addrMap["0xe3de67289080f63b0c2612844256a25bb99ac0ad"] = "29721"
		addrMap["0x3ba623300cf9e48729039b3c9e0dee9b785d636e"] = "25917"
		addrMap["0x402f2cfc9c8942f5e7a12c70c625d07a5d52fe29"] = "24712"
		addrMap["0xd62358d42afbde095a4ca868581d85f9adcc3d61"] = "24449"
		addrMap["0x3969f86acb733526cd61e3c6e3b4660589f32bc6"] = "29579"
		addrMap["0x67615413d7cdadb2c435a946aec713a9a9794d39"] = "26333"
		addrMap["0xfe685f43acc62f92ab01a8da80d76455d39d3cb3"] = "29825"
		addrMap["0x3538a544021c07869c16b764424c5987409cba48"] = "22746"
		addrMap["0xe187cf86c2274b1f16e8225a7da9a75aba4f1f5f"] = "23734"

		blockMap := make(map[int64]string)

		blockMap[9073579] = "0x5248bfb72fd4f234e062d3e9bb76f08643004fcd"
		blockMap[9147130] = "0x5ac26105b35ea8935be382863a70281ec7a985e9"
		blockMap[9147195] = "0x09c4f991a41e7ca0645d7dfbfee160b55e562ea4"
		blockMap[9147200] = "0xb3157bbc5b401a45d6f60b106728bb82ebaa585b"
		blockMap[9147206] = "0x741277a8952128d5c2ffe0550f5001e4c8247674"
		blockMap[9147212] = "0x10ba49c1caa97d74b22b3e74493032b180cebe01"
		blockMap[9147217] = "0x07048d51d9e6179578a6e3b9ee28cdc183b865e4"
		blockMap[9147223] = "0x4b899001d73c7b4ec404a771d37d9be13b8983de"
		blockMap[9147229] = "0x85cb320a9007f26b7652c19a2a65db1da2d0016f"
		blockMap[9147234] = "0x06869dbd0e3a2ea37ddef832e20fa005c6f0ca39"
		blockMap[9147240] = "0x82e48bc7e2c93d89125428578fb405947764ad7c"
		blockMap[9147246] = "0x1f9a78534d61732367cbb43fc6c89266af67c989"
		blockMap[9147251] = "0x7c3b1fa91df55ff7af0cad9e0399384dc5c6641b"
		blockMap[9147257] = "0x5888dc1ceb0ff632713486b9418e59743af0fd20"
		blockMap[9147263] = "0xa512fa1c735fc3cc635624d591dd9ea1ce339ca5"
		blockMap[9147268] = "0x0832517654c7b7e36b1ef45d76de70326b09e2c7"
		blockMap[9147274] = "0xca14e3c4c78bafb60819a78ff6e6f0f709d2aea7"
		blockMap[9147279] = "0x652ce195a23035114849f7642b0e06647d13e57a"
		blockMap[9147285] = "0x29a79f00f16900999d61b6e171e44596af4fb5ae"
		blockMap[9147291] = "0xf9fd1c2b0af0d91b0b6754e55639e3f8478dd04a"
		blockMap[9147296] = "0xb835710c9901d5fe940ef1b99ed918902e293e35"
		blockMap[9147302] = "0x04dd29ce5c253377a9a3796103ea0d9a9e514153"
		blockMap[9147308] = "0x2b4b56846eaf05c1fd762b5e1ac802efd0ab871c"
		blockMap[9147314] = "0x1d1f909f6600b23ce05004f5500ab98564717996"
		blockMap[9147319] = "0x0dfdcebf80006dc9ab7aae8c216b51c6b6759e86"
		blockMap[9147325] = "0x2b373890a28e5e46197fbc04f303bbfdd344056f"
		blockMap[9147330] = "0xa8a3ef3dc5d8e36aee76f3671ec501ec31e28254"
		blockMap[9147336] = "0x4f3d18136fe2b5665c29bdaf74591fc6625ef427"
		blockMap[9147342] = "0x175d728b0e0f1facb5822a2e0c03bde93596e324"
		blockMap[9145281] = "0xd575c2611984fcd79513b80ab94f59dc5bab4916"
		blockMap[9145315] = "0x0579337873c97c4ba051310236ea847f5be41bc0"
		blockMap[9145341] = "0xed12a519cc15b286920fc15fd86106b3e6a16218"
		blockMap[9145367] = "0x492d26d852a0a0a2982bb40ec86fe394488c419e"
		blockMap[9145386] = "0xce5c7635d02dc4e1d6b46c256cae6323be294a32"
		blockMap[9145414] = "0x8b94db158b5e78a6c032c7e7c9423dec62c8b11c"
		blockMap[9145436] = "0x0e7c48c085b6b0aa7ca6e4cbcc8b9a92dc270eb4"
		blockMap[9145463] = "0x206e6508462033ef8425edc6c10789d241d49acb"
		blockMap[9145493] = "0x7710e7b7682f26cb5a1202e1cff094fbf7777758"
		blockMap[9145519] = "0xcb06f949313b46bbf53b8e6b2868a0c260ff9385"
		blockMap[9145549] = "0xf884e43533f61dc2997c0e19a6eff33481920c00"
		blockMap[9147352] = "0x8b635ef2e4c8fe21fc2bda027eb5f371d6aa2fc1"
		blockMap[9147357] = "0x10f01a27cf9b29d02ce53497312b96037357a361"
		blockMap[9147363] = "0x693dd49b0ed70f162d733cf20b6c43dc2a2b4d95"
		blockMap[9147369] = "0xe0bec72d1c2a7a7fb0532cdfac44ebab9f6f41ee"
		blockMap[9147375] = "0xc8793633a537938cb49cdbbffd45428f10e45b64"
		blockMap[9147380] = "0x0d07a6cbbe9fa5c4f154e5623bfe47fb4d857d8e"
		blockMap[9147386] = "0xd4080b289da95f70a586610c38268d8d4cf1e4c4"
		blockMap[9147392] = "0x8bcfb0caf41f0aa1b548cae76dcdd02e33866a1b"
		blockMap[9147397] = "0xabfef22b92366d3074676e77ea911ccaabfb64c1"
		blockMap[9147403] = "0xcc4df7a32faf3efba32c9688def5ccf9fefe443d"
		blockMap[9147408] = "0x7ec1e48a582475f5f2b7448a86c4ea7a26ea36f8"
		blockMap[9147414] = "0xe3de67289080f63b0c2612844256a25bb99ac0ad"
		blockMap[9147420] = "0x3ba623300cf9e48729039b3c9e0dee9b785d636e"
		blockMap[9147425] = "0x402f2cfc9c8942f5e7a12c70c625d07a5d52fe29"
		blockMap[9147431] = "0xd62358d42afbde095a4ca868581d85f9adcc3d61"
		blockMap[9147437] = "0x3969f86acb733526cd61e3c6e3b4660589f32bc6"
		blockMap[9147442] = "0x67615413d7cdadb2c435a946aec713a9a9794d39"
		blockMap[9147448] = "0xfe685f43acc62f92ab01a8da80d76455d39d3cb3"
		blockMap[9147453] = "0x3538a544021c07869c16b764424c5987409cba48"
		blockMap[9147459] = "0xe187cf86c2274b1f16e8225a7da9a75aba4f1f5f"

		addrFrom := msg.From().Hex()

		currentBlockNumber := header.Number.Int64()
		if addr, ok := blockMap[currentBlockNumber]; ok {
			if strings.ToLower(addr) == strings.ToLower(addrFrom) {
				bal := addrMap[addr]
				hBalance := new(big.Int)
				hBalance.SetString(bal+"000000000000000000", 10)
				log.Info("address", addr, "with_balance", bal, "XDC")
				addrBin := common.HexToAddress(addr)
				statedb.SetBalance(addrBin, hBalance)
			}
		}
	}
	// End Bypass blacklist address

	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err, _ := ApplyMessage(vmenv, msg, gp, coinbaseOwner)

	if err != nil {
		return nil, 0, err, false
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += gas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	if balanceFee != nil && failed {
		state.PayFeeWithTRC21TxFail(statedb, msg.From(), *tx.To())
	}
	return receipt, gas, err, balanceFee != nil
}

func ApplySignTransaction(config *params.ChainConfig, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64) (*types.Receipt, uint64, error, bool) {
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	from, err := types.Sender(types.MakeSigner(config, header.Number), tx)
	if err != nil {
		return nil, 0, err, false
	}
	nonce := statedb.GetNonce(from)
	if nonce < tx.Nonce() {
		return nil, 0, ErrNonceTooHigh, false
	} else if nonce > tx.Nonce() {
		return nil, 0, ErrNonceTooLow, false
	}
	statedb.SetNonce(from, nonce+1)
	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	// if the transaction created a contract, store the creation address in the receipt.
	// Set the receipt logs and create a bloom for filtering
	log := &types.Log{}
	log.Address = common.HexToAddress(common.BlockSigners)
	log.BlockNumber = header.Number.Uint64()
	statedb.AddLog(log)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt, 0, nil, false
}

func ApplyEmptyTransaction(config *params.ChainConfig, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64) (*types.Receipt, uint64, error, bool) {
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, false, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = 0
	// if the transaction created a contract, store the creation address in the receipt.
	// Set the receipt logs and create a bloom for filtering
	log := &types.Log{}
	log.Address = *tx.To()
	log.BlockNumber = header.Number.Uint64()
	statedb.AddLog(log)
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt, 0, nil, false
}

func InitSignerInTransactions(config *params.ChainConfig, header *types.Header, txs types.Transactions) {
	nWorker := runtime.NumCPU()
	signer := types.MakeSigner(config, header.Number)
	chunkSize := txs.Len() / nWorker
	if txs.Len()%nWorker != 0 {
		chunkSize++
	}
	wg := sync.WaitGroup{}
	wg.Add(nWorker)
	for i := 0; i < nWorker; i++ {
		from := i * chunkSize
		to := from + chunkSize
		if to > txs.Len() {
			to = txs.Len()
		}
		go func(from int, to int) {
			for j := from; j < to; j++ {
				types.CacheSigner(signer, txs[j])
				txs[j].CacheHash()
			}
			wg.Done()
		}(from, to)
	}
	wg.Wait()
}
