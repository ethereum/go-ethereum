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

func uploadAndSync(c *cli.Context) error {
	defer func(now time.Time) {
		totalTime := time.Since(now)
		log.Info("total time", "time", totalTime, "kb", filesize)
		metrics.GetOrRegisterResettingTimer("upload-and-sync.total-time", nil).Update(totalTime)
	}(time.Now())

	generateEndpoints(scheme, cluster, appName, from, to)
	seed := int(time.Now().UnixNano() / 1e6)
	log.Info("uploading to "+endpoints[0]+" and syncing", "seed", seed)

	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	t1 := time.Now()
	hash, err := upload(&randomBytes, endpoints[0])
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterResettingTimer("upload-and-sync.upload-time", nil).UpdateSince(t1)

	fhash, err := digest(bytes.NewReader(randomBytes))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))

	time.Sleep(time.Duration(syncDelay) * time.Second)

	wg := sync.WaitGroup{}
	if single {
		rand.Seed(time.Now().UTC().UnixNano())
		randIndex := 1 + rand.Intn(len(endpoints)-1)
		ruid := uuid.New()[:8]
		wg.Add(1)
		go func(endpoint string, ruid string) {
			for {
				start := time.Now()
				err := fetch(hash, endpoint, fhash, ruid)
				if err != nil {
					continue
				}

				metrics.GetOrRegisterResettingTimer("upload-and-sync.single.fetch-time", nil).UpdateSince(start)
				wg.Done()
				return
			}
		}(endpoints[randIndex], ruid)
	} else {
		for _, endpoint := range endpoints[1:] {
			ruid := uuid.New()[:8]
			wg.Add(1)
			go func(endpoint string, ruid string) {
				for {
					start := time.Now()
					err := fetch(hash, endpoint, fhash, ruid)
					if err != nil {
						continue
					}

					metrics.GetOrRegisterResettingTimer("upload-and-sync.each.fetch-time", nil).UpdateSince(start)
					wg.Done()
					return
				}
			}(endpoint, ruid)
		}
	}
	wg.Wait()
	log.Info("all endpoints synced random file successfully")

	return nil
}
