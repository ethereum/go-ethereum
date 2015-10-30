package contract

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
)

type Contract struct {
	backend core.Backend
	abi.ABI
	Address common.Address
}

func getAccount(backend core.Backend) *state.StateObject {
	statedb, _ := state.New(backend.BlockChain().CurrentHeader().Root, backend.ChainDb())
	accounts, err := backend.AccountManager().Accounts()
	if err != nil || len(accounts) == 0 {
		return statedb.GetOrNewStateObject(common.Address{})
	} else {
		return statedb.GetOrNewStateObject(accounts[0].Address)
	}
}

func New(definition string, addr common.Address, backend core.Backend) (*Contract, error) {
	abi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		return nil, err
	}

	return &Contract{
		backend: backend,
		ABI:     abi,
		Address: addr,
	}, nil
}

func (c *Contract) Call(method string, args ...interface{}) ([]byte, *big.Int, error) {
	msg := CallMessage{
		gas:      new(big.Int).Set(common.MaxBig),
		gasPrice: new(big.Int),
		value:    new(big.Int),
		from:     getAccount(c.backend),
	}
	return c.CallWithMessage(msg, method, args...)
}

func (c *Contract) CallWithMessage(msg CallMessage, method string, args ...interface{}) ([]byte, *big.Int, error) {
	input, err := c.ABI.Pack(method, args...)
	if err != nil {
		return nil, nil, err
	}
	msg.data = input
	msg.to = &c.Address

	header := c.backend.BlockChain().CurrentHeader()
	statedb, err := state.New(header.Root, c.backend.ChainDb())
	if err != nil {
		return nil, nil, err
	}
	vmenv := core.NewEnv(statedb, c.backend.BlockChain(), msg, header)
	gp := new(core.GasPool).AddGas(common.MaxBig)

	return core.ApplyMessage(vmenv, msg, gp)
}

// CallMessage is the message type used for call transations.
type CallMessage struct {
	from          *state.StateObject
	to            *common.Address
	gas, gasPrice *big.Int
	value         *big.Int
	data          []byte
}

// accessor boilerplate to implement core.Message
func (m CallMessage) From() (common.Address, error) { return m.from.Address(), nil }
func (m CallMessage) Nonce() uint64                 { return m.from.Nonce() }
func (m CallMessage) To() *common.Address           { return m.to }
func (m CallMessage) GasPrice() *big.Int            { return m.gasPrice }
func (m CallMessage) Gas() *big.Int                 { return m.gas }
func (m CallMessage) Value() *big.Int               { return m.value }
func (m CallMessage) Data() []byte                  { return m.data }
