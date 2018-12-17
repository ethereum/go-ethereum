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
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/swarm/network"
	"github.com/ethereum/go-ethereum/swarm/network/simulation"
	cli "gopkg.in/urfave/cli.v1"
)

func verify(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		return errors.New("argument should be the filename to verify or write-to")
	}
	filename, err := touchPath(ctx.Args()[0])
	if err != nil {
		return err
	}
	err = verifySnapshot(filename)
	if err != nil {
		utils.Fatalf("Simulation failed: %s", err)
	}

	return err
}

func verifySnapshot(filename string) error {
	sim := simulation.New(map[string]simulation.ServiceFunc{
		"bzz": func(ctx *adapters.ServiceContext, b *sync.Map) (node.Service, func(), error) {
			addr := network.NewAddr(ctx.Config.Node())

			kp := network.NewKadParams()
			kp.MinProxBinSize = testMinProxBinSize

			kad := network.NewKademlia(addr.Over(), kp)
			hp := network.NewHiveParams()
			hp.KeepAliveInterval = time.Duration(200) * time.Millisecond
			hp.Discovery = true //discovery

			config := &network.BzzConfig{
				OverlayAddr:  addr.Over(),
				UnderlayAddr: addr.Under(),
				HiveParams:   hp,
			}
			return network.NewBzz(config, kad, nil, nil, nil), nil, nil

		},
	})
	defer sim.Close()
	err := sim.UploadSnapshot(filename)
	if err != nil {
		utils.Fatalf("%v", err)
	}

	ctx, cancelSimRun := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelSimRun()

	if _, err := sim.WaitTillHealthy(ctx, 2); err != nil {
		utils.Fatalf("%v", err)
	}

	return nil
}
