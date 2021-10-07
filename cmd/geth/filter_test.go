package main

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"gopkg.in/urfave/cli.v1"
)

func TestGethFilters(t *testing.T) {
	// will capture the port assigned from scanning the log output.
	cliArgs := []string{"", "--dev", "-http", "--http.port=0", "--ws", "--ws.port=0"}

	// app created in init()
	app.Action = func(ctx *cli.Context) error {
		var ipcEndpoint string
		var httpEndpoint *net.TCPAddr
		var wsEndpoint string

		lookingC := make(chan struct{})
		stillLooking := true
		log.Root().SetHandler(log.FuncHandler(func(r *log.Record) error {
			if v := scanFor(r, "HTTP server started", "endpoint"); v != nil {
				httpEndpoint = v.(*net.TCPAddr)
				t.Log("http", httpEndpoint)
			}
			if v := scanFor(r, "IPC endpoint opened", "url"); v != nil {
				ipcEndpoint = v.(string)
				t.Log("ipc", ipcEndpoint)
			}
			if v := scanFor(r, "WebSocket enabled", "url"); v != nil {
				wsEndpoint = v.(string)
				t.Log("ws", wsEndpoint)

			}
			if stillLooking && ipcEndpoint != "" && httpEndpoint != nil && wsEndpoint != "" {
				// only want the chan send to happen once
				stillLooking = false
				lookingC <- struct{}{}
			}
			return nil
		}))

		prepare(ctx)
		stack, backend := makeFullNode(ctx)
		go startNode(ctx, stack, backend)

		t.Log("waiting")
		<-lookingC

		e := newFilterEnv(t, stack, backend)

		t.Log("deploy contract")
		e.deployContract(t)

		// call contract methods to emit events
		t.Log("call contract")
		e.embedLogs(t)

		// run filter tests
		t.Log("run filters")
		e.checkFilters(t)

		e.stack.Close()
		stack.Wait()

		return nil
	}
	app.Before = nil
	app.After = nil
	err := app.Run(cliArgs)
	if err != nil {
		t.Fatal(err)
	}
}

func scanFor(r *log.Record, msg string, key string) interface{} {
	if r.Msg == msg {
		for i, v := range r.Ctx {
			if v == key {
				fmt.Println(r.Ctx[i+1])
				return r.Ctx[i+1]
			}
		}
	}
	return nil
}

func (e *filterEnv) checkFilters(t *testing.T) {
	tests := []struct {
		filter   ethereum.FilterQuery
		expected []result
	}{
		{
			filter: ethereum.FilterQuery{}, // all of the events/logs
			expected: []result{
				{
					"Event1",
					nil,
				},
				{
					"Event2",
					map[string]interface{}{
						"src": e.auth.From,
					},
				},
				{
					"Event3",
					map[string]interface{}{
						"amt": big.NewInt(30000),
						"src": e.auth.From,
					},
				},
			},
		},
		{
			filter: ethereum.FilterQuery{
				Topics: [][]common.Hash{{e.contractAbi.Events["Event1"].ID}},
			},
			expected: []result{
				{
					"Event1",
					nil,
				},
			},
		},
		{
			filter: ethereum.FilterQuery{
				Topics: [][]common.Hash{{e.contractAbi.Events["Event2"].ID}},
			},
			expected: []result{
				{
					"Event2",
					map[string]interface{}{
						"src": e.auth.From,
					},
				},
			},
		},
	}
	for _, v := range tests {
		res := e.query(t, v.filter)
		if len(res) != len(v.expected) {
			t.Fatal("number of results does not match expected", len(res), len(v.expected))
		}
		for i, r := range res {
			err := checkResult(v.expected[i], r)
			if err != nil {
				t.Fatal(err)
			}

		}
	}
}

func checkResult(expected result, logEvent event) error {
	if logEvent.event.Name != expected.name {
		return fmt.Errorf("log/event names do not match: expected %s got %s", expected.name, logEvent.event.Name)
	}
	if expected.args == nil {
		expected.args = make(map[string]interface{})
	}
	if !reflect.DeepEqual(expected.args, logEvent.args) {
		return fmt.Errorf("log/event arguments do not match: expected %s got %s",
			fmt.Sprint(expected.args),
			fmt.Sprint(logEvent.args))
	}
	return nil
}

type filterEnv struct {
	stack       *node.Node
	backend     ethapi.Backend
	cid         *big.Int
	auth        *bind.TransactOpts
	ethclient   *ethclient.Client
	contractAbi abi.ABI
	contractBin string
	contract    *bind.BoundContract
}

func newFilterEnv(t *testing.T, stack *node.Node, backend ethapi.Backend) *filterEnv {
	rpc, err := stack.Attach()
	if err != nil {
		t.Fatal(err)
	}
	ec := ethclient.NewClient(rpc)

	a, err := abi.JSON(strings.NewReader(contractAbi))
	if err != nil {
		t.Fatal(err)
	}
	return &filterEnv{stack, backend, backend.ChainConfig().ChainID, devUserAuth(t, backend), ec, a, contractBin, nil}
}

func (e *filterEnv) deployContract(t *testing.T) {
	_, tx, bc, err := bind.DeployContract(e.auth, e.contractAbi, common.FromHex(e.contractBin), e.ethclient)
	if err != nil {
		t.Fatal(err)
	}
	ctx, canc := context.WithTimeout(context.Background(), time.Second*5)
	defer canc()
	_, err = bind.WaitDeployed(ctx, e.ethclient, tx)
	if err != nil {
		t.Fatal(err)
	}
	e.contract = bc
}

func (e *filterEnv) callContract(t *testing.T, fname string) {
	tx, err := e.contract.Transact(e.auth, fname)
	if err != nil {
		t.Fatal(err)
	}
	ctx, canc := context.WithTimeout(context.Background(), time.Second*5)
	defer canc()
	_, err = bind.WaitMined(ctx, e.ethclient, tx)
	if err != nil {
		t.Fatal(err)
	}
}

func (e *filterEnv) embedLogs(t *testing.T) {
	e.callContract(t, "event1")
	e.callContract(t, "event2")
	e.callContract(t, "event3")
}

type event struct {
	event abi.Event
	args  map[string]interface{}
}

type result struct {
	name string
	args map[string]interface{}
}

func (e *filterEnv) query(t *testing.T, q ethereum.FilterQuery) []event {
	ctx, canc := context.WithTimeout(context.Background(), time.Second*5)
	defer canc()
	logs, err := e.ethclient.FilterLogs(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	res := make([]event, 0)
	for _, v := range logs {
		ev, m, err := eventFromLog(e.contract, e.contractAbi, &v)
		if err != nil {
			t.Fatal(err)
		}
		res = append(res, event{ev, m})
	}
	return res
}

func devUserAuth(t *testing.T, be ethapi.Backend) *bind.TransactOpts {
	// this is a dev instance, there should be at least one account
	ws := be.AccountManager().Wallets()
	if len(ws) == 0 {
		t.Fatal("no wallets found")
	}
	accts := ws[0].Accounts()
	if len(accts) == 0 {
		t.Fatal("need one account to sign transactions")
	}
	return &bind.TransactOpts{
		From: accts[0].Address,
		Signer: func(_ common.Address, tx *types.Transaction) (*types.Transaction, error) {
			return ws[0].SignTx(accts[0], tx, be.ChainConfig().ChainID)
		},
	}
}

func eventFromLog(c *bind.BoundContract, a abi.ABI, l *types.Log) (abi.Event, map[string]interface{}, error) {
	if len(l.Topics) == 0 {
		return abi.Event{}, nil, fmt.Errorf("no topics")
	}

	for k, v := range a.Events {
		if v.ID == l.Topics[0] {
			out := make(map[string]interface{})
			err := c.UnpackLogIntoMap(out, k, *l)
			if err != nil {
				return abi.Event{}, nil, err
			}
			return v, out, nil
		}
	}
	return abi.Event{}, nil, fmt.Errorf("event not found")
}

const contractAbi = `[{"anonymous":false,"inputs":[],"name":"Event1","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"src","type":"address"}],"name":"Event2","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"src","type":"address"},{"indexed":false,"internalType":"uint256","name":"amt","type":"uint256"}],"name":"Event3","type":"event"},{"inputs":[],"name":"event1","outputs":[],"stateMutability":"payable","type":"function"},{"inputs":[],"name":"event2","outputs":[],"stateMutability":"payable","type":"function"},{"inputs":[],"name":"event3","outputs":[],"stateMutability":"payable","type":"function"}]`

const contractBin = `608060405234801561001057600080fd5b50610198806100206000396000f3fe6080604052600436106100345760003560e01c8063ee45446214610039578063f609b92014610043578063f8280c101461004d575b600080fd5b610041610057565b005b61004b610085565b005b6100556100d7565b005b7feb0647d2fd18e9d48417458d5ceb82e5e455b589086e6a791507d38a59f9f5cb60405160405180910390a1565b3373ffffffffffffffffffffffffffffffffffffffff167f7f32d075ae25f22fef851731f202dc59873b7f08b4c98e8772ebde6051e3fb336175306040516100cd919061012b565b60405180910390a2565b3373ffffffffffffffffffffffffffffffffffffffff167ff4c21e8c024fc52804d008eb9080763568b332e32c9ed12b1a7c9ba1a70cc90e60405160405180910390a2565b61012581610150565b82525050565b6000602082019050610140600083018461011c565b92915050565b6000819050919050565b600061015b82610146565b905091905056fea26469706673582212209c4bd38c4b6ec793b6bca3aab04ae87540fb58754f2a36559408c4376ea3682b64736f6c63430008040033`

/*
pragma solidity ^0.8.4;

contract FilterLog {
	 event Event1 ();
	 event Event2(address indexed src);
	 event Event3 ( address indexed src, uint amt);

	 function event1 () public payable {
	 	  emit Event1();
	 }

	 function event2 () public payable {
	 	  emit Event2 ( msg.sender );
	 }

	 function event3 () public payable {
	 	  emit Event3 (msg.sender, 30000);
	 }
}
*/
