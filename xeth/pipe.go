package xeth

/*
 * eXtended ETHereum
 */

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

var pipelogger = logger.NewLogger("XETH")

type VmVars struct {
	State *state.StateDB
}

type XEth struct {
	obj          core.EthManager
	blockManager *core.BlockManager
	blockChain   *core.ChainManager
	world        *World

	Vm VmVars
}

func New(obj core.EthManager) *XEth {
	pipe := &XEth{
		obj:          obj,
		blockManager: obj.BlockManager(),
		blockChain:   obj.ChainManager(),
	}
	pipe.world = NewWorld(pipe)

	return pipe
}

/*
 * State / Account accessors
 */
func (self *XEth) Balance(addr []byte) *ethutil.Value {
	return ethutil.NewValue(self.World().safeGet(addr).Balance)
}

func (self *XEth) Nonce(addr []byte) uint64 {
	return self.World().safeGet(addr).Nonce
}

func (self *XEth) Block(hash []byte) *types.Block {
	return self.blockChain.GetBlock(hash)
}

func (self *XEth) Storage(addr, storageAddr []byte) *ethutil.Value {
	return self.World().safeGet(addr).GetStorage(ethutil.BigD(storageAddr))
}

func (self *XEth) Exists(addr []byte) bool {
	return self.World().Get(addr) != nil
}

// Converts the given private key to an address
func (self *XEth) ToAddress(priv []byte) []byte {
	pair, err := crypto.NewKeyPairFromSec(priv)
	if err != nil {
		return nil
	}

	return pair.Address()
}

/*
 * Execution helpers
 */
func (self *XEth) Execute(addr []byte, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	return self.ExecuteObject(&Object{self.World().safeGet(addr)}, data, value, gas, price)
}

func (self *XEth) ExecuteObject(object *Object, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	var (
		initiator = state.NewStateObject(self.obj.KeyManager().KeyPair().Address())
		block     = self.blockChain.CurrentBlock
	)

	self.Vm.State = self.World().State().Copy()

	vmenv := NewEnv(self.Vm.State, block, value.BigInt(), initiator.Address())
	return vmenv.Call(initiator, object.Address(), data, gas.BigInt(), price.BigInt(), value.BigInt())
}

/*
 * Transactional methods
 */
func (self *XEth) TransactString(key *crypto.KeyPair, rec string, value, gas, price *ethutil.Value, data []byte) (*types.Transaction, error) {
	// Check if an address is stored by this address
	var hash []byte
	addr := self.World().Config().Get("NameReg").StorageString(rec).Bytes()
	if len(addr) > 0 {
		hash = addr
	} else if ethutil.IsHex(rec) {
		hash = ethutil.Hex2Bytes(rec[2:])
	} else {
		hash = ethutil.Hex2Bytes(rec)
	}

	return self.Transact(key, hash, value, gas, price, data)
}

func (self *XEth) Transact(key *crypto.KeyPair, to []byte, value, gas, price *ethutil.Value, data []byte) (*types.Transaction, error) {
	var hash []byte
	var contractCreation bool
	if types.IsContractAddr(to) {
		contractCreation = true
	} else {
		// Check if an address is stored by this address
		addr := self.World().Config().Get("NameReg").Storage(to).Bytes()
		if len(addr) > 0 {
			hash = addr
		} else {
			hash = to
		}
	}

	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), data)
	} else {
		tx = types.NewTransactionMessage(hash, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	state := self.blockManager.TransState()
	nonce := state.GetNonce(key.Address())

	tx.Nonce = nonce
	tx.Sign(key.PrivateKey)

	// Do some pre processing for our "pre" events  and hooks
	block := self.blockChain.NewBlock(key.Address())
	coinbase := state.GetStateObject(key.Address())
	coinbase.SetGasPool(block.GasLimit)
	self.blockManager.ApplyTransactions(coinbase, state, block, types.Transactions{tx}, true)

	err := self.obj.TxPool().Add(tx)
	if err != nil {
		return nil, err
	}
	state.SetNonce(key.Address(), nonce+1)

	if contractCreation {
		addr := tx.CreationAddress(self.World().State())
		pipelogger.Infof("Contract addr %x\n", addr)
	}

	return tx, nil
}

func (self *XEth) PushTx(tx *types.Transaction) ([]byte, error) {
	err := self.obj.TxPool().Add(tx)
	if err != nil {
		return nil, err
	}

	if tx.Recipient == nil {
		addr := tx.CreationAddress(self.World().State())
		pipelogger.Infof("Contract addr %x\n", addr)
		return addr, nil
	}
	return tx.Hash(), nil
}

func (self *XEth) CompileMutan(code string) ([]byte, error) {
	data, err := ethutil.Compile(code, false)
	if err != nil {
		return nil, err
	}

	return data, nil
}
