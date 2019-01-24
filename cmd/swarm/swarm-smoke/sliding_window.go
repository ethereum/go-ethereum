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

func slidingWindow(c *cli.Context) error {
	// test dscription:
	// 1. upload repeatedly the same file size, maintain a slice in which swarm hashes are stored, first hash at idx=0
	// 2. select a random node, start downloading the hashes, starting with the LAST one first (it should always be availble), till the FIRST hash
	// 3. when

	defer func(now time.Time) {
		totalTime := time.Since(now)

		log.Info("total time", "time", totalTime)
		metrics.GetOrRegisterCounter("sliding-window.total-time", nil).Inc(int64(totalTime))
	}(time.Now())

	generateEndpoints(scheme, cluster, appName, from, to)
	storeSize = storeSize * 4096 //store size is in chunks - transform to bytes
	hashes := []uploadResult{}   //swarm hashes of the uploads
	nodes := to - from
	networkCapacity := float64(storeSize) * float64(nodes)
	const iterationTimeout = 30 * time.Second
	log.Info("sliding window test started", "store size(kb)", int(storeSize/1000), "nodes", nodes, "filesize(kb)", int(filesize/1000), "network capacity(kb)", int(networkCapacity/1000), "timeout", timeout)
	uploadedBytes := 0
	networkDepth := 0
	errored := false

outer:
	for {
		seed := int(time.Now().UnixNano() / 1e6)
		log.Info("uploading to "+endpoints[0]+" and syncing", "seed", seed)

		randomBytes := testutil.RandomBytes(seed, filesize)

		t1 := time.Now()
		hash, err := upload(&randomBytes, endpoints[0])
		if err != nil {
			log.Error(err.Error())
			return err
		}
		metrics.GetOrRegisterCounter("sliding-window.upload-time", nil).Inc(int64(time.Since(t1)))

		fhash, err := digest(bytes.NewReader(randomBytes))
		if err != nil {
			log.Error(err.Error())
			return err
		}

		log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash), "sleeping", syncDelay)
		hashes = append(hashes, uploadResult{hash: hash, digest: fhash})
		time.Sleep(time.Duration(syncDelay) * time.Second)
		uploadedBytes += filesize

		for i, v := range hashes {
			rand.Seed(time.Now().UTC().UnixNano())
			randIndex := 1 + rand.Intn(len(endpoints)-1)
			ruid := uuid.New()[:8]
			start := time.Now()
			err := fetch(v.hash, endpoints[randIndex], v.digest, ruid)
			fetchTime := time.Since(start)
			if err != nil {
				errored = true
				log.Error("error retrieving hash", "hash idx", i, "err", err)
				metrics.GetOrRegisterCounter("sliding-window.single.error", nil).Inc(1)
				networkDepth = i
				break outer
			}

			metrics.GetOrRegisterMeter("sliding-window.single.fetch-time", nil).Mark(int64(fetchTime))
		}
	}

	log.Info("sliding window test finished", "errored?", errored, "networkDepth", networkDepth, "networkDepth(kb)", int(networkDepth*filesize/1000))
	log.Info("stats", "uploadedFiles", len(hashes), "uploadedKb", uploadedBytes/1000, "filesizeKb", filesize/1000, "networkCapacityKb", int(networkCapacity/1000), "networkCapacityMb", int(networkCapacity/1000000))

	metrics.GetOrRegisterMeter("sliding-window.network-depth", nil).Mark(int64(networkDepth))
	return nil
}
