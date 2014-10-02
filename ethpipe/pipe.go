package ethpipe

import (
	"fmt"
	"strings"

	"github.com/ethereum/eth-go/ethchain"
	"github.com/ethereum/eth-go/ethcrypto"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethvm"
)

var logger = ethlog.NewLogger("PIPE")

type VmVars struct {
	State *ethstate.State
}

type Pipe struct {
	obj          ethchain.EthManager
	stateManager *ethchain.StateManager
	blockChain   *ethchain.BlockChain
	world        *World

	Vm VmVars
}

func New(obj ethchain.EthManager) *Pipe {
	pipe := &Pipe{
		obj:          obj,
		stateManager: obj.StateManager(),
		blockChain:   obj.BlockChain(),
	}
	pipe.world = NewWorld(pipe)

	return pipe
}

func (self *Pipe) Balance(addr []byte) *ethutil.Value {
	return ethutil.NewValue(self.World().safeGet(addr).Balance)
}

func (self *Pipe) Nonce(addr []byte) uint64 {
	return self.World().safeGet(addr).Nonce
}

func (self *Pipe) Execute(addr []byte, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	return self.ExecuteObject(&Object{self.World().safeGet(addr)}, data, value, gas, price)
}

func (self *Pipe) ExecuteObject(object *Object, data []byte, value, gas, price *ethutil.Value) ([]byte, error) {
	var (
		initiator = ethstate.NewStateObject(self.obj.KeyManager().KeyPair().Address())
		block     = self.blockChain.CurrentBlock
	)

	self.Vm.State = self.World().State().Copy()

	vm := ethvm.New(NewEnv(self.Vm.State, block, value.BigInt(), initiator.Address()))
	vm.Verbose = true

	msg := ethvm.NewMessage(vm, object.Address(), data, gas.BigInt(), price.BigInt(), value.BigInt())
	ret, err := msg.Exec(object.Address(), initiator)

	fmt.Println("returned from call", ret, err)

	return ret, err
}

func (self *Pipe) Block(hash []byte) *ethchain.Block {
	return self.blockChain.GetBlock(hash)
}

func (self *Pipe) Storage(addr, storageAddr []byte) *ethutil.Value {
	return self.World().safeGet(addr).GetStorage(ethutil.BigD(storageAddr))
}

func (self *Pipe) ToAddress(priv []byte) []byte {
	pair, err := ethcrypto.NewKeyPairFromSec(priv)
	if err != nil {
		return nil
	}

	return pair.Address()
}

func (self *Pipe) Exists(addr []byte) bool {
	return self.World().Get(addr) != nil
}

func (self *Pipe) TransactString(key *ethcrypto.KeyPair, rec string, value, gas, price *ethutil.Value, data []byte) ([]byte, error) {
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

func (self *Pipe) Transact(key *ethcrypto.KeyPair, rec []byte, value, gas, price *ethutil.Value, data []byte) ([]byte, error) {
	var hash []byte
	var contractCreation bool
	if rec == nil {
		contractCreation = true
	}

	var tx *ethchain.Transaction
	// Compile and assemble the given data
	if contractCreation {
		script, err := ethutil.Compile(string(data), false)
		if err != nil {
			return nil, err
		}

		tx = ethchain.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), script)
	} else {
		data := ethutil.StringToByteFunc(string(data), func(s string) (ret []byte) {
			slice := strings.Split(s, "\n")
			for _, dataItem := range slice {
				d := ethutil.FormatData(dataItem)
				ret = append(ret, d...)
			}
			return
		})

		tx = ethchain.NewTransactionMessage(hash, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	acc := self.stateManager.TransState().GetOrNewStateObject(key.Address())
	tx.Nonce = acc.Nonce
	acc.Nonce += 1
	self.stateManager.TransState().UpdateStateObject(acc)

	tx.Sign(key.PrivateKey)
	self.obj.TxPool().QueueTransaction(tx)

	if contractCreation {
		addr := tx.CreationAddress(self.World().State())
		logger.Infof("Contract addr %x\n", addr)

		return addr, nil
	}

	return tx.Hash(), nil
}

func (self *Pipe) PushTx(tx *ethchain.Transaction) ([]byte, error) {
	self.obj.TxPool().QueueTransaction(tx)
	if tx.Recipient == nil {
		addr := tx.CreationAddress(self.World().State())
		logger.Infof("Contract addr %x\n", addr)
		return addr, nil
	}
	return tx.Hash(), nil
}

func (self *Pipe) CompileMutan(code string) ([]byte, error) {
	data, err := ethutil.Compile(code, false)
	if err != nil {
		return nil, err
	}

	return data, nil
}
