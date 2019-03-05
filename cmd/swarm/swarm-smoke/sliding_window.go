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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

type uploadResult struct {
	hash   string
	digest []byte
}

func slidingWindowCmd(ctx *cli.Context, tuid string) error {
	errc := make(chan error)

	go func() {
		errc <- slidingWindow(ctx, tuid)
	}()

	err := <-errc
	if err != nil {
		metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", commandName), nil).Inc(1)
	}
	return err
}

func slidingWindow(ctx *cli.Context, tuid string) error {
	var hashes []uploadResult //swarm hashes of the uploads
	nodes := len(hosts)
	log.Info("sliding window test started", "tuid", tuid, "nodes", nodes, "filesize(kb)", filesize, "timeout", timeout)
	uploadedBytes := 0
	networkDepth := 0
	errored := false

outer:
	for {
		seed = int(time.Now().UTC().UnixNano())
		log.Info("uploading to "+httpEndpoint(hosts[0])+" and syncing", "seed", seed)

		t1 := time.Now()

		randomBytes := testutil.RandomBytes(seed, filesize*1000)

		hash, err := upload(randomBytes, httpEndpoint(hosts[0]))
		if err != nil {
			log.Error(err.Error())
			return err
		}

		metrics.GetOrRegisterResettingTimer("sliding-window.upload-time", nil).UpdateSince(t1)
		metrics.GetOrRegisterGauge("sliding-window.upload-depth", nil).Update(int64(len(hashes)))

		fhash, err := digest(bytes.NewReader(randomBytes))
		if err != nil {
			log.Error(err.Error())
			return err
		}

		log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash), "sleeping", syncDelay)
		hashes = append(hashes, uploadResult{hash: hash, digest: fhash})
		time.Sleep(time.Duration(syncDelay) * time.Second)
		uploadedBytes += filesize * 1000
		q := make(chan struct{}, 1)
		d := make(chan struct{})
		defer close(q)
		defer close(d)
		for i, v := range hashes {
			timeoutC := time.After(time.Duration(timeout) * time.Second)
			errored = false

		task:
			for {
				select {
				case q <- struct{}{}:
					go func() {
						var start time.Time
						done := false
						for !done {
							log.Info("trying to retrieve hash", "hash", v.hash)
							idx := 1 + rand.Intn(len(hosts)-1)
							ruid := uuid.New()[:8]
							start = time.Now()
							// fetch hangs when swarm dies out, so we have to jump through a bit more hoops to actually
							// catch the timeout, but also allow this retry logic
							err := fetch(v.hash, httpEndpoint(hosts[idx]), v.digest, ruid, "")
							if err != nil {
								log.Error("error fetching hash", "err", err)
								continue
							}
							done = true
						}
						metrics.GetOrRegisterResettingTimer("sliding-window.single.fetch-time", nil).UpdateSince(start)
						d <- struct{}{}
					}()
				case <-d:
					<-q
					break task
				case <-timeoutC:
					errored = true
					log.Error("error retrieving hash. timeout", "hash idx", i)
					metrics.GetOrRegisterCounter("sliding-window.single.error", nil).Inc(1)
					break outer
				default:
				}
			}

			networkDepth = i
			metrics.GetOrRegisterGauge("sliding-window.network-depth", nil).Update(int64(networkDepth))
			log.Info("sliding window test successfully fetched file", "currentDepth", networkDepth)
			// this test might take a long time to finish - but we'd like to see metrics while they accumulate and not just when
			// the test finishes. therefore emit the metrics on each iteration
			emitMetrics(ctx)
		}
	}

	log.Info("sliding window test finished", "errored?", errored, "networkDepth", networkDepth, "networkDepth(kb)", networkDepth*filesize)
	log.Info("stats", "uploadedFiles", len(hashes), "uploadedKb", uploadedBytes/1000, "filesizeKb", filesize)

	return nil
}
