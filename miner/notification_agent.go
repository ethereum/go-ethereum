// Copyright 2015 The go-ethereum Authors
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

package miner

import (
	"sync"

	"bytes"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/log"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// The notification agent pushes new work packages to a preset list of HTTP endpoints
type NotificationAgent struct {
	wg       sync.WaitGroup
	workCh   chan *Work
	stop     chan struct{}
	returnCh chan<- *Result
	targets  []string
	client   *http.Client

	chain  consensus.ChainReader
	engine consensus.Engine
}

func NewNotificationAgent(chain consensus.ChainReader, engine consensus.Engine, targets string) *NotificationAgent {
	miner := &NotificationAgent{
		chain:   chain,
		engine:  engine,
		stop:    make(chan struct{}, 1),
		workCh:  make(chan *Work, 1),
		targets: strings.Split(targets, ","),
		client:  &http.Client{Timeout: time.Second * 10},
	}
	return miner
}

func (self *NotificationAgent) Work() chan<- *Work            { return self.workCh }
func (self *NotificationAgent) SetReturnCh(ch chan<- *Result) { self.returnCh = ch }

func (self *NotificationAgent) Stop() {
	self.stop <- struct{}{}
done:
	// Empty work channel
	for {
		select {
		case <-self.workCh:
		default:
			break done
		}
	}
}

func (self *NotificationAgent) Start() {
	go self.update()
}

func (self *NotificationAgent) update() {
out:
	for {
		select {
		case work := <-self.workCh:
			var res [3]string

			block := work.Block

			res[0] = block.HashNoNonce().Hex()
			seedHash := ethash.SeedHash(block.NumberU64())
			res[1] = common.BytesToHash(seedHash).Hex()
			// Calculate the "target" to be returned to the external miner
			n := big.NewInt(1)
			n.Lsh(n, 255)
			n.Div(n, block.Difficulty())
			n.Lsh(n, 1)
			res[2] = common.BytesToHash(n.Bytes()).Hex()

			resJson, err := json.Marshal(res)
			if err != nil {
				log.Error("Unable to marshal work package into JSON string")
				continue
			}

			self.wg.Add(len(self.targets))
			for _, target := range self.targets {
				func(t string) {
					_, err := self.client.Post(t, "application/json", bytes.NewBuffer(resJson))
					if err != nil {
						log.Error("Error invoking work notification handler", "handler", t, "err", err)
					}
					self.wg.Done()
				}(target)
			}
			self.wg.Wait()
		case <-self.stop:
			break out
		}
	}
}

func (self *NotificationAgent) GetHashRate() int64 {
	return 0
}
