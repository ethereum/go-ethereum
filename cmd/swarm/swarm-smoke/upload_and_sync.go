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
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadAndSyncCmd(ctx *cli.Context, tuid string) error {
	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	errc := make(chan error)

	go func() {
		errc <- uplaodAndSync(ctx, randomBytes, tuid)
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", commandName), nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("%s.timeout", commandName), nil).Inc(1)

		// trigger debug functionality on randomBytes

		return fmt.Errorf("timeout after %v sec", timeout)
	}
}

func uplaodAndSync(c *cli.Context, randomBytes []byte, tuid string) error {
	log.Info("uploading to "+httpEndpoint(hosts[0])+" and syncing", "tuid", tuid, "seed", seed)

	t1 := time.Now()
	hash, err := upload(randomBytes, httpEndpoint(hosts[0]))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	t2 := time.Since(t1)
	metrics.GetOrRegisterResettingTimer("upload-and-sync.upload-time", nil).Update(t2)

	fhash, err := digest(bytes.NewReader(randomBytes))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "tuid", tuid, "hash", hash, "took", t2, "digest", fmt.Sprintf("%x", fhash))

	time.Sleep(time.Duration(syncDelay) * time.Second)

	wg := sync.WaitGroup{}
	if single {
		randIndex := 1 + rand.Intn(len(hosts)-1)
		ruid := uuid.New()[:8]
		wg.Add(1)
		go func(endpoint string, ruid string) {
			for {
				start := time.Now()
				err := fetch(hash, endpoint, fhash, ruid, tuid)
				if err != nil {
					continue
				}
				ended := time.Since(start)

				metrics.GetOrRegisterResettingTimer("upload-and-sync.single.fetch-time", nil).Update(ended)
				log.Info("fetch successful", "tuid", tuid, "ruid", ruid, "took", ended, "endpoint", endpoint)
				wg.Done()
				return
			}
		}(httpEndpoint(hosts[randIndex]), ruid)
	} else {
		for _, endpoint := range hosts[1:] {
			ruid := uuid.New()[:8]
			wg.Add(1)
			go func(endpoint string, ruid string) {
				for {
					start := time.Now()
					err := fetch(hash, endpoint, fhash, ruid, tuid)
					if err != nil {
						continue
					}
					ended := time.Since(start)

					metrics.GetOrRegisterResettingTimer("upload-and-sync.each.fetch-time", nil).Update(ended)
					log.Info("fetch successful", "tuid", tuid, "ruid", ruid, "took", ended, "endpoint", endpoint)
					wg.Done()
					return
				}
			}(httpEndpoint(endpoint), ruid)
		}
	}
	wg.Wait()
	log.Info("all hosts synced random file successfully")

	return nil
}
