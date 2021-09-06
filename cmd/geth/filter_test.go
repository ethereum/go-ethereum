package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
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
	app.Action = func(ctx *cli.Context) error {
		prepare(ctx)
		stack, backend := makeFullNode(ctx)
		startNode(ctx, stack, backend)
		go func() {
			e := newFilterEnv(t, stack, backend)
			e.deployContract(t)
			// call contract methods to emit events
			e.embedLogs(t)
			// run filter tests
			e.checkFilters(t)
			e.stack.Close()
		}()
		stack.Wait()

		return nil
	}
	app.Before = nil
	app.After = nil
	// verbosity=0 on command line doesn't seem to do anything
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(0), nil))
	dir, err := os.MkdirTemp(os.TempDir(), "ethdev")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	err = app.Run([]string{"", "--dev", "--datadir", dir, "--verbosity=0"})
	if err != nil {
		t.Fatal(err)
	}
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
					nil,
				},
				{
					"Event3",
					map[string]interface{}{
						"amt": big.NewInt(30000),
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
	}
	for _, v := range tests {
		res := e.query(t, v.filter)
		if len(res) != len(v.expected) {
			t.Fatal("number of results does not match expected")
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
	if logEvent.args == nil && expected.args == nil {
		return nil
	}
	if logEvent.args == nil {
		return fmt.Errorf("expected log args")
	}
	return checkResultArgs(expected.args, logEvent.args)
}

func checkResultArgs(map[string]interface{}, map[string]interface{}) error {
	// todo
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
