// eXtended ETHereum
package xeth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/whisper"
)

var pipelogger = logger.NewLogger("XETH")

// to resolve the import cycle
type Backend interface {
	BlockProcessor() *core.BlockProcessor
	ChainManager() *core.ChainManager
	AccountManager() *accounts.Manager
	TxPool() *core.TxPool
	PeerCount() int
	IsListening() bool
	Peers() []*p2p.Peer
	BlockDb() common.Database
	StateDb() common.Database
	ExtraDb() common.Database
	EventMux() *event.TypeMux
	Whisper() *whisper.Whisper

	IsMining() bool
	StartMining() error
	StopMining()
	Version() string
}

// Frontend should be implemented by users of XEth. Its methods are
// called whenever XEth makes a decision that requires user input.
type Frontend interface {
	// UnlockAccount is called when a transaction needs to be signed
	// but the key corresponding to the transaction's sender is
	// locked.
	//
	// It should unlock the account with the given address and return
	// true if unlocking succeeded.
	UnlockAccount(address []byte) bool

	// This is called for all transactions inititated through
	// Transact. It should prompt the user to confirm the transaction
	// and return true if the transaction was acknowledged.
	//
	// ConfirmTransaction is not used for Call transactions
	// because they cannot change any state.
	ConfirmTransaction(tx *types.Transaction) bool
}

type XEth struct {
	eth            Backend
	blockProcessor *core.BlockProcessor
	chainManager   *core.ChainManager
	accountManager *accounts.Manager
	state          *State
	whisper        *Whisper

	frontend Frontend
}

// dummyFrontend is a non-interactive frontend that allows all
// transactions but cannot not unlock any keys.
type dummyFrontend struct{}

func (dummyFrontend) UnlockAccount([]byte) bool                  { return false }
func (dummyFrontend) ConfirmTransaction(*types.Transaction) bool { return true }

// New creates an XEth that uses the given frontend.
// If a nil Frontend is provided, a default frontend which
// confirms all transactions will be used.
func New(eth Backend, frontend Frontend) *XEth {
	xeth := &XEth{
		eth:            eth,
		blockProcessor: eth.BlockProcessor(),
		chainManager:   eth.ChainManager(),
		accountManager: eth.AccountManager(),
		whisper:        NewWhisper(eth.Whisper()),
		frontend:       frontend,
	}
	if frontend == nil {
		xeth.frontend = dummyFrontend{}
	}
	xeth.state = NewState(xeth, xeth.chainManager.TransState())
	return xeth
}

func (self *XEth) Backend() Backend { return self.eth }
func (self *XEth) WithState(statedb *state.StateDB) *XEth {
	xeth := &XEth{
		eth:            self.eth,
		blockProcessor: self.blockProcessor,
		chainManager:   self.chainManager,
		whisper:        self.whisper,
	}

	xeth.state = NewState(xeth, statedb)
	return xeth
}
func (self *XEth) State() *State { return self.state }

func (self *XEth) Whisper() *Whisper { return self.whisper }

func (self *XEth) BlockByHash(strHash string) *Block {
	hash := common.HexToHash(strHash)
	block := self.chainManager.GetBlock(hash)

	return NewBlock(block)
}

func (self *XEth) EthBlockByHash(strHash string) *types.Block {
	hash := common.HexToHash(strHash)
	block := self.chainManager.GetBlock(hash)

	return block
}

func (self *XEth) EthTransactionByHash(hash string) *types.Transaction {
	data, _ := self.eth.ExtraDb().Get(common.FromHex(hash))
	if len(data) != 0 {
		return types.NewTransactionFromBytes(data)
	}
	return nil
}

func (self *XEth) BlockByNumber(num int64) *Block {
	if num == -1 {
		return NewBlock(self.chainManager.CurrentBlock())
	}

	return NewBlock(self.chainManager.GetBlockByNumber(uint64(num)))
}

func (self *XEth) EthBlockByNumber(num int64) *types.Block {
	if num == -1 {
		return self.chainManager.CurrentBlock()
	}

	return self.chainManager.GetBlockByNumber(uint64(num))
}

func (self *XEth) Block(v interface{}) *Block {
	if n, ok := v.(int32); ok {
		return self.BlockByNumber(int64(n))
	} else if str, ok := v.(string); ok {
		return self.BlockByHash(str)
	} else if f, ok := v.(float64); ok { // Don't ask ...
		return self.BlockByNumber(int64(f))
	}

	return nil
}

func (self *XEth) Accounts() []string {
	// TODO: check err?
	accounts, _ := self.eth.AccountManager().Accounts()
	accountAddresses := make([]string, len(accounts))
	for i, ac := range accounts {
		accountAddresses[i] = common.ToHex(ac.Address)
	}
	return accountAddresses
}

func (self *XEth) PeerCount() int {
	return self.eth.PeerCount()
}

func (self *XEth) IsMining() bool {
	return self.eth.IsMining()
}

func (self *XEth) SetMining(shouldmine bool) bool {
	ismining := self.eth.IsMining()
	if shouldmine && !ismining {
		err := self.eth.StartMining()
		return err == nil
	}
	if ismining && !shouldmine {
		self.eth.StopMining()
	}
	return self.eth.IsMining()
}

func (self *XEth) IsListening() bool {
	return self.eth.IsListening()
}

func (self *XEth) Coinbase() string {
	cb, _ := self.eth.AccountManager().Coinbase()
	return common.ToHex(cb)
}

func (self *XEth) NumberToHuman(balance string) string {
	b := common.Big(balance)

	return common.CurrencyToString(b)
}

func (self *XEth) StorageAt(addr, storageAddr string) string {
	storage := self.State().SafeGet(addr).StorageString(storageAddr)

	return common.ToHex(storage.Bytes())
}

func (self *XEth) BalanceAt(addr string) string {
	return self.State().SafeGet(addr).Balance().String()
}

func (self *XEth) TxCountAt(address string) int {
	return int(self.State().SafeGet(address).Nonce())
}

func (self *XEth) CodeAt(address string) string {
	return common.ToHex(self.State().SafeGet(address).Code())
}

func (self *XEth) IsContract(address string) bool {
	return len(self.State().SafeGet(address).Code()) > 0
}

func (self *XEth) SecretToAddress(key string) string {
	pair, err := crypto.NewKeyPairFromSec(common.FromHex(key))
	if err != nil {
		return ""
	}

	return common.ToHex(pair.Address())
}

type KeyVal struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (self *XEth) EachStorage(addr string) string {
	var values []KeyVal
	object := self.State().SafeGet(addr)
	it := object.Trie().Iterator()
	for it.Next() {
		values = append(values, KeyVal{common.ToHex(it.Key), common.ToHex(it.Value)})
	}

	valuesJson, err := json.Marshal(values)
	if err != nil {
		return ""
	}

	return string(valuesJson)
}

func (self *XEth) ToAscii(str string) string {
	padded := common.RightPadBytes([]byte(str), 32)

	return "0x" + common.ToHex(padded)
}

func (self *XEth) FromAscii(str string) string {
	if common.IsHex(str) {
		str = str[2:]
	}

	return string(bytes.Trim(common.FromHex(str), "\x00"))
}

func (self *XEth) FromNumber(str string) string {
	if common.IsHex(str) {
		str = str[2:]
	}

	return common.BigD(common.FromHex(str)).String()
}

func (self *XEth) PushTx(encodedTx string) (string, error) {
	tx := types.NewTransactionFromBytes(common.FromHex(encodedTx))
	err := self.eth.TxPool().Add(tx)
	if err != nil {
		return "", err
	}

	if tx.To() == nil {
		addr := core.AddressFromMessage(tx)
		return addr.Hex(), nil
	}
	return tx.Hash().Hex(), nil
}

var (
	defaultGasPrice = big.NewInt(10000000000000)
	defaultGas      = big.NewInt(90000)
)

func (self *XEth) Call(fromStr, toStr, valueStr, gasStr, gasPriceStr, dataStr string) (string, error) {
	statedb := self.State().State() //self.chainManager.TransState()
	msg := callmsg{
		from:     statedb.GetOrNewStateObject(common.HexToAddress(fromStr)),
		to:       common.HexToAddress(toStr),
		gas:      common.Big(gasStr),
		gasPrice: common.Big(gasPriceStr),
		value:    common.Big(valueStr),
		data:     common.FromHex(dataStr),
	}
	if msg.gas.Cmp(big.NewInt(0)) == 0 {
		msg.gas = defaultGas
	}

	if msg.gasPrice.Cmp(big.NewInt(0)) == 0 {
		msg.gasPrice = defaultGasPrice
	}

	block := self.chainManager.CurrentBlock()
	vmenv := core.NewEnv(statedb, self.chainManager, msg, block)

	res, err := vmenv.Call(msg.from, msg.to, msg.data, msg.gas, msg.gasPrice, msg.value)
	return common.ToHex(res), err
}

func (self *XEth) Transact(fromStr, toStr, valueStr, gasStr, gasPriceStr, codeStr string) (string, error) {
	var (
		from             = common.HexToAddress(fromStr)
		to               = common.HexToAddress(toStr)
		value            = common.NewValue(valueStr)
		gas              = common.NewValue(gasStr)
		price            = common.NewValue(gasPriceStr)
		data             []byte
		contractCreation bool
	)

	data = common.FromHex(codeStr)
	if len(toStr) == 0 {
		contractCreation = true
	}

	var tx *types.Transaction
	if contractCreation {
		tx = types.NewContractCreationTx(value.BigInt(), gas.BigInt(), price.BigInt(), data)
	} else {
		tx = types.NewTransactionMessage(to, value.BigInt(), gas.BigInt(), price.BigInt(), data)
	}

	state := self.chainManager.TxState()
	nonce := state.NewNonce(from) //state.GetNonce(from)
	tx.SetNonce(nonce)

	if err := self.sign(tx, from, false); err != nil {
		return "", err
	}
	if err := self.eth.TxPool().Add(tx); err != nil {
		return "", err
	}
	//state.IncrementNonce(from)

	if contractCreation {
		addr := core.AddressFromMessage(tx)
		pipelogger.Infof("Contract addr %x\n", addr)

		return core.AddressFromMessage(tx).Hex(), nil
	}
	return tx.Hash().Hex(), nil
}

func (self *XEth) sign(tx *types.Transaction, from common.Address, didUnlock bool) error {
	sig, err := self.accountManager.Sign(accounts.Account{Address: from.Bytes()}, tx.Hash().Bytes())
	if err == accounts.ErrLocked {
		if didUnlock {
			return fmt.Errorf("sender account still locked after successful unlock")
		}
		if !self.frontend.UnlockAccount(from.Bytes()) {
			return fmt.Errorf("could not unlock sender account")
		}
		// retry signing, the account should now be unlocked.
		return self.sign(tx, from, true)
	} else if err != nil {
		return err
	}
	tx.SetSignatureValues(sig)
	return nil
}

// callmsg is the message type used for call transations.
type callmsg struct {
	from          *state.StateObject
	to            common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m callmsg) From() (common.Address, error) { return m.from.Address(), nil }
func (m callmsg) Nonce() uint64                 { return m.from.Nonce() }
func (m callmsg) To() *common.Address           { return &m.to }
func (m callmsg) GasPrice() *big.Int            { return m.gasPrice }
func (m callmsg) Gas() *big.Int                 { return m.gas }
func (m callmsg) Value() *big.Int               { return m.value }
func (m callmsg) Data() []byte                  { return m.data }
