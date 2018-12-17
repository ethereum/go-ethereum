// Copyright 2018 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	cli "gopkg.in/urfave/cli.v1"
)

const testMinProxBinSize = 2
const NoConnectionTimeout = 2 * time.Second

func create(ctx *cli.Context) error {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	if len(ctx.Args()) < 1 {
		return errors.New("argument should be the filename to verify or write-to")
	}
	filename, err := touchPath(ctx.Args()[0])
	if err != nil {
		return err
	}
	err = discoverySnapshot(filename, 10)
	if err != nil {
		utils.Fatalf("Simulation failed: %s", err)
	}

	return err
}

func discoverySnapshot(filename string, nodes int) error {
	//disable discovery if topology is specified
	discovery = topology == ""
	log.Debug("discoverySnapshot", "filename", filename, "nodes", nodes, "discovery", discovery)
	i := 0
	var lock sync.Mutex
	var pivotNodeID enode.ID
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			lock.Lock()
			i++
			if i == pivot {
				pivotNodeID = ctx.Config.ID
			}
			lock.Unlock()

			addr := network.NewAddr(ctx.Config.Node())
			kp := network.NewKadParams()
			kp.MinProxBinSize = testMinProxBinSize

			kad := network.NewKademlia(addr.Over(), kp)
			hp := network.NewHiveParams()
			hp.KeepAliveInterval = time.Duration(200) * time.Millisecond
			hp.Discovery = discovery

			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kad, nil, nil, nil), nil, nil
		},
	})
	defer sim.Close()

	_, err := sim.AddNodes(10)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	events := make(chan *simulations.Event)
	sub := sim.Net.Events().Subscribe(events)
	select {
	case ev := <-events:
		//only catch node up events
		if ev.Type == simulations.EventTypeConn {
			utils.Fatalf("this shouldn't happen as connections weren't initiated yet")
		}
	case <-time.After(NoConnectionTimeout):
	}

	sub.Unsubscribe()

	if len(sim.Net.Conns) > 0 {
		utils.Fatalf("no connections should exist after just adding nodes")
	}

	err := sim.Net.ConnectNodesRing(nil)
	if err != nil {
		utils.Fatalf("had an error connecting the nodes in a %v topology: %v", topology, err)
	}

	if discovery {
		ctx, cancelSimRun := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancelSimRun()

		if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	var snap *simulations.Snapshot
	if len(services) > 0 {
		var addServices []string
		var removeServices []string
		for _, osvc := range strings.Split(services, ",") {
			if strings.Index(osvc, "+") == 0 {
				addServices = append(addServices, osvc[1:])
			} else if strings.Index(osvc, "-") == 0 {
				removeServices = append(removeServices, osvc[1:])
			} else {
				panic("stick to the rules, you know what they are")
			}
		}
		snap, err = sim.Net.SnapshotWithServices(addServices, removeServices)
	} else {
		snap, err = sim.Net.Snapshot()
	}

	if err != nil {
		return errors.New("no shapshot dude")
	}
	jsonsnapshot, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("corrupt json snapshot: %v", err)
	}
	err = ioutil.WriteFile(filename, jsonsnapshot, 0666)
	if err != nil {
		return err
	}

	return nil
}
