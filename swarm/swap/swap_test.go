// Copyright 2018 The go-ethereum Authors
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

package swap

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/state"
	colorable "github.com/mattn/go-colorable"
)

var (
	p2pPort       = 30100
	ipcpath       = ".swarm.ipc"
	datadirPrefix = ".data_"
	stackW        = &sync.WaitGroup{}
	loglevel      = flag.Int("loglevel", 2, "verbosity of logs")
)

var testSpec = &protocols.Spec{
	Name:       "swapTestSpec",
	Version:    1,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		testExceedsPayAtMsg{},
		testExceedsDropAtMsg{},
	},
}

type testExceedsPayAtMsg struct{}
type testExceedsDropAtMsg struct{}

func (tmsg *testExceedsPayAtMsg) GetMsgPrice() *big.Int {
	diff := &big.Int{}
	return diff.Sub(payAt, big.NewInt(1))
}

func (tmsg *testExceedsDropAtMsg) GetMsgPrice() *big.Int {
	diff := &big.Int{}
	return diff.Sub(dropAt, big.NewInt(1))
}

func init() {
	flag.Parse()

	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

func TestLimits(t *testing.T) {
	if dropAt.Cmp(payAt) > -1 {
		t.Fatal(fmt.Sprintf("dropAt limit is not lower than payAt limit, dropAt: %s, payAt: %s", dropAt.String(), payAt.String()))
	}
}

func TestExceedsPayAt(t *testing.T) {
	dir, err := ioutil.TempDir("", "swap_test_store")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	stateStore, err2 := state.NewDBStore(dir)
	if err2 != nil {
		panic(err2)
	}
	swap, err3 := NewSwap(NewDefaultSwapParams().Params, stateStore)
	if err3 != nil {
		t.Fatal(err3)
	}
	id := adapters.RandomNodeConfig().ID
	testPeer := protocols.NewPeer(p2p.NewPeer(id, "testPeer", nil), nil, nil)
	sp := NewSwapPeer(testPeer, swap)
	sp.handlerFunc = msgHandler
	ctx := context.Background()
	sp.handleAccountedMsg(ctx, &testExceedsPayAtMsg{})
}

func TestExceedsDropAt(t *testing.T) {
	dir, err := ioutil.TempDir("", "swap_test_store")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	stateStore, err2 := state.NewDBStore(dir)
	if err2 != nil {
		panic(err2)
	}
	swap, err3 := NewSwap(NewDefaultSwapParams().Params, stateStore)
	if err3 != nil {
		t.Fatal(err3)
	}
	id := adapters.RandomNodeConfig().ID
	testPeer := protocols.NewPeer(p2p.NewPeer(id, "testPeer", nil), nil, nil)
	sp := NewSwapPeer(testPeer, swap)
	sp.handlerFunc = msgHandler
	ctx := context.Background()
	err = sp.Send(ctx, &testExceedsDropAtMsg{})
	if err != ErrInsufficientFunds {
		t.Fatal("Expected test to fail with insufficient funds, but it didn't")
	}
}

func TestProtocol(t *testing.T) {
	/*
		dir, err := ioutil.TempDir("", "swap_test_store")
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(dir)

		swap := NewSwap(NewDefaultSwapParams().Params, state.NewDBStore(dir))
		conf := adapters.RandomNodeConfig()

		protocol := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			peer := protocols.NewPeer(p, rw, testSpec)
			runProtocol(peer, swap)
		}
		protocolTester := p2ptest.NewProtocolTester(t, conf.ID, 2, protocol)
		protocolTester.TestExchanges(p2p.TestExchange{
			Label: "",
			Expects: []p2ptest.Expect{
				{
					Code: 5,
					Msg: &RetrieveRequestMsg{
						Addr:      hash0[:],
						SkipCheck: true,
					},
					Peer: peerID,
				},
			},
		})
	*/
}

func runProtocol(peer *protocols.Peer, swap *Swap) {
	sp := NewSwapPeer(peer, swap)
	sp.Peer.Run(msgHandler)
}

func msgHandler(ctx context.Context, msg interface{}) error {
	return nil
}

func TestSwapProtocol(t *testing.T) {

	// create the two nodes
	stack_one, err := newServiceNode(p2pPort, 0, 0)
	if err != nil {
		log.Crit("Create servicenode #1 fail", "err", err)
	}
	stack_two, err := newServiceNode(p2pPort+1, 0, 0)
	if err != nil {
		log.Crit("Create servicenode #2 fail", "err", err)
	}

	instance := NewSwapProtocol()
	// wrapper function for servicenode to start the service
	swapsvc := func(ctx *node.ServiceContext) (node.Service, error) {
		return &API{
			SwapProtocol: instance,
		}, nil
	}

	// register adds the service to the services the servicenode starts when started
	err = stack_one.Register(swapsvc)
	if err != nil {
		log.Crit("Register service in servicenode #1 fail", "err", err)
	}
	err = stack_two.Register(swapsvc)
	if err != nil {
		log.Crit("Register service in servicenode #2 fail", "err", err)
	}

	// start the nodes
	err = stack_one.Start()
	if err != nil {
		log.Crit("servicenode #1 start failed", "err", err)
	}
	err = stack_two.Start()
	if err != nil {
		log.Crit("servicenode #2 start failed", "err", err)
	}

	// connect to the servicenode RPCs
	rpcclient_one, err := rpc.Dial(filepath.Join(stack_one.DataDir(), ipcpath))
	if err != nil {
		log.Crit("connect to servicenode #1 IPC fail", "err", err)
	}
	defer os.RemoveAll(stack_one.DataDir())

	rpcclient_two, err := rpc.Dial(filepath.Join(stack_two.DataDir(), ipcpath))
	if err != nil {
		log.Crit("connect to servicenode #2 IPC fail", "err", err)
	}
	defer os.RemoveAll(stack_two.DataDir())

	// display that the initial pong counts are 0
	var balance int
	err = rpcclient_one.Call(&balance, "swap_balance")
	if err != nil {
		log.Crit("servicenode #1 pongcount RPC failed", "err", err)
	}
	log.Info("servicenode #1 before ping", "balance-1", balance)

	err = rpcclient_two.Call(&balance, "swap_balance")
	if err != nil {
		log.Crit("servicenode #2 pongcount RPC failed", "err", err)
	}
	log.Info("servicenode #2 before ping", "balance-2", balance)

	/*
		// get the server instances
		srv_one := stack_one.Server()
		srv_two := stack_two.Server()

			// subscribe to peerevents
			eventOneC := make(chan *p2p.PeerEvent)
			sub_one := srv_one.SubscribeEvents(eventOneC)

			eventTwoC := make(chan *p2p.PeerEvent)
			sub_two := srv_two.SubscribeEvents(eventTwoC)

			// connect the nodes
			p2pnode_two := srv_two.Self()
			srv_one.AddPeer(p2pnode_two)

			// fork and do the pinging
			stackW.Add(2)
			pingmax_one := 4
			pingmax_two := 2

			go func() {

				// when we get the add event, we know we are connected
				ev := <-eventOneC
				if ev.Type != "add" {
				log.Error("server #1 expected peer add", "eventtype", ev.Type)
					stackW.Done()
					return
				}
				log.Debug("server #1 connected", "peer", ev.Peer)

				// send the pings
				for i := 0; i < pingmax_one; i++ {
					err := rpcclient_one.Call(nil, "foo_ping", ev.Peer)
					if err != nil {
						log.Error("server #1 RPC ping fail", "err", err)
						stackW.Done()
						break
					}
				}

				// wait for all msgrecv events
				// pings we receive, and pongs we expect from pings we sent
				for i := 0; i < pingmax_two+pingmax_one; {
					ev := <-eventOneC
					log.Warn("msg", "type", ev.Type, "i", i)
					if ev.Type == "msgrecv" {
						i++
					}
				}

				stackW.Done()
			}()

			// mirrors the previous go func
			go func() {
				ev := <-eventTwoC
				if ev.Type != "add" {
					log.Error("expected peer add", "eventtype", ev.Type)
					stackW.Done()
					return
				}
				log.Debug("server #2 connected", "peer", ev.Peer)
				for i := 0; i < pingmax_two; i++ {
					err := rpcclient_two.Call(nil, "foo_ping", ev.Peer)
					if err != nil {
						log.Error("server #2 RPC ping fail", "err", err)
						stackW.Done()
						break
					}
				}

				for i := 0; i < pingmax_one+pingmax_two; {
					ev := <-eventTwoC
					if ev.Type == "msgrecv" {
						log.Warn("msg", "type", ev.Type, "i", i)
						i++
					}
				}

				stackW.Done()
			}()

			// wait for the two ping pong exchanges to finish
			stackW.Wait()

			// tell the API to shut down
			// this will disconnect the peers and close the channels connecting API and protocol
			err = rpcclient_one.Call(nil, "foo_quit", srv_two.Self().ID)
			if err != nil {
				log.Error("server #1 RPC quit fail", "err", err)
			}
			err = rpcclient_two.Call(nil, "foo_quit", srv_one.Self().ID)
			if err != nil {
				log.Error("server #2 RPC quit fail", "err", err)
			}

			// disconnect will generate drop events
			for {
				ev := <-eventOneC
				if ev.Type == "drop" {
					break
				}
			}
			for {
				ev := <-eventTwoC
				if ev.Type == "drop" {
					break
				}
			}

			// proudly inspect the results
			err = rpcclient_one.Call(&count, "foo_pongCount")
			if err != nil {
				log.Crit("servicenode #1 pongcount RPC failed", "err", err)
			}
			log.Info("servicenode #1 after ping", "pongcount", count)

			err = rpcclient_two.Call(&count, "foo_pongCount")
			if err != nil {
				log.Crit("servicenode #2 pongcount RPC failed", "err", err)
			}
			log.Info("servicenode #2 after ping", "pongcount", count)

			// bring down the servicenodes
			sub_one.Unsubscribe()
			sub_two.Unsubscribe()
			stack_one.Stop()
			stack_two.Stop()
	*/
}

func newServiceNode(port int, httpport int, wsport int, modules ...string) (*node.Node, error) {
	cfg := &node.DefaultConfig
	cfg.P2P.ListenAddr = fmt.Sprintf(":%d", port)
	cfg.P2P.EnableMsgEvents = true
	cfg.P2P.NoDiscovery = true
	cfg.IPCPath = ipcpath
	cfg.DataDir = fmt.Sprintf("%s%d", datadirPrefix, port)
	if httpport > 0 {
		cfg.HTTPHost = node.DefaultHTTPHost
		cfg.HTTPPort = httpport
	}
	if wsport > 0 {
		cfg.WSHost = node.DefaultWSHost
		cfg.WSPort = wsport
		cfg.WSOrigins = []string{"*"}
		for i := 0; i < len(modules); i++ {
			cfg.WSModules = append(cfg.WSModules, modules[i])
		}
	}
	stack, err := node.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("ServiceNode create fail: %v", err)
	}
	return stack, nil
}
