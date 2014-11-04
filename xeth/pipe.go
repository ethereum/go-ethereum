package xeth

/*
 * eXtended ETHereum
 */

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/chain"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/vm"
)

var pipelogger = logger.NewLogger("XETH")

type VmVars struct {
	State *state.State
}

type XEth struct {
	obj          chain.EthManager
	blockManager *chain.BlockManager
	blockChain   *chain.ChainManager
	world        *World

	Vm VmVars
}

func New(obj chain.EthManager) *XEth {
	pipe := &XEth{
		obj:          obj,
		blockManager: obj.BlockManager(),
		blockChain:   obj.ChainManager(),
	}
	pipe.world = NewWorld(pipe)

	return pipe
}

func (self *XEth) Balance(addr []byte) *ethutil.Value {
	return ethutil.NewValue(self.World().safeGet(addr).Balance)
}

func (self *XEth) Nonce(addr []byte) uint64 {
	return self.World().safeGet(addr).Nonce
}

func (self *XEth) Execute(addr []byte, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	return self.ExecuteObject(&Object{self.World().safeGet(addr)}, data, value, gas, price)
}

func (self *XEth) ExecuteObject(object *Object, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	var (
		initiator = state.NewStateObject(self.obj.KeyManager().KeyPair().Address())
		block     = self.blockChain.CurrentBlock
	)

	self.Vm.State = self.World().State().Copy()

	evm := vm.New(NewEnv(self.Vm.State, block, value.BigInt(), initiator.Address()), vm.Type(ethutil.Config.VmType))

	msg := vm.NewExecution(evm, object.Address(), data, gas.BigInt(), price.BigInt(), value.BigInt())
	ret, err := msg.Exec(object.Address(), initiator)

	fmt.Println("returned from call", ret, err)

	return ret, err
}

func (self *XEth) Block(hash []byte) *chain.Block {
	return self.blockChain.GetBlock(hash)
}

func (self *XEth) Storage(addr, storageAddr []byte) *ethutil.Value {
	return self.World().safeGet(addr).GetStorage(ethutil.BigD(storageAddr))
}

func (self *XEth) ToAddress(priv []byte) []byte {
	pair, err := crypto.NewKeyPairFromSec(priv)
	if err != nil {
		return nil
	}

	return pair.Address()
}

func (self *XEth) Exists(addr []byte) bool {
	return self.World().Get(addr) != nil
}

func (self *XEth) TransactString(key *crypto.KeyPair, rec string, value, gas, price *ethutil.Value, data []byte) ([]byte, error) {
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

func (self *XEth) Transact(key *crypto.KeyPair, rec []byte, value, gas, price *ethutil.Value, data []byte) ([]byte, error) {
	var hash []byte
	var contractCreation bool
	if rec == nil {
		contractCreation = true
	}

	var tx *chain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		script, err := ethutil.Compile(string(data), false)
		if err != nil {
			return nil, err
		}

		tx = chain.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), script)
	} else {
		data := ethutil.StringToByteFunc(string(data), func(s string) (ret []byte) {
			slice := strings.Split(s, "\n")
			for _, dataItem := range slice {
				d := ethutil.FormatData(dataItem)
				ret = append(ret, d...)
			}
			return
		})

		tx = chain.NewTransactionMessage(hash, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	acc := self.blockManager.TransState().GetOrNewStateObject(key.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.blockManager.TransState().UpdateStateObject(acc)

	tx.Sign(key.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		addr := tx.CreationAddress(self.World().State())
		pipelogger.Infof("Contract addr %x\n", addr)

		return addr, nil
	}

	return tx.Hash(), nil
}

func (self *XEth) PushTx(tx *chain.Transaction) ([]byte, error) {
	self.obj.TxPool().QueueTransaction(tx)
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
