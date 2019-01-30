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
	"crypto/md5"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

var seed = time.Now().UTC().UnixNano()

func init() {
	rand.Seed(seed)
}

type uploadResult struct {
	hash   string
	digest []byte
}

func slidingWindow(c *cli.Context) error {
	generateEndpoints(scheme, cluster, appName, from, to)
	hashes := []uploadResult{} //swarm hashes of the uploads
	nodes := to - from
	const iterationTimeout = 30 * time.Second
	log.Info("sliding window test started", "nodes", nodes, "filesize(kb)", filesize, "timeout", timeout)
	uploadedBytes := 0
	networkDepth := 0
	errored := false

outer:
	for {
		log.Info("uploading to "+endpoints[0]+" and syncing", "seed", seed)

		h := md5.New()
		r := io.TeeReader(io.LimitReader(crand.Reader, int64(filesize*1000)), h)
		t1 := time.Now()

		hash, err := upload(r, filesize*1000, endpoints[0])
		if err != nil {
			log.Error(err.Error())
			return err
		}

		metrics.GetOrRegisterResettingTimer("sliding-window.upload-time", nil).UpdateSince(t1)

		fhash := h.Sum(nil)

		log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash), "sleeping", syncDelay)
		hashes = append(hashes, uploadResult{hash: hash, digest: fhash})
		time.Sleep(time.Duration(syncDelay) * time.Second)
		uploadedBytes += filesize * 1000

		for i, v := range hashes {
			timeout := time.After(time.Duration(timeout) * time.Second)
			errored = false

		inner:
			for {
				select {
				case <-timeout:
					errored = true
					log.Error("error retrieving hash. timeout", "hash idx", i, "err", err)
					metrics.GetOrRegisterCounter("sliding-window.single.error", nil).Inc(1)
					break inner
				default:
					randIndex := 1 + rand.Intn(len(endpoints)-1)
					ruid := uuid.New()[:8]
					start := time.Now()
					err := fetch(v.hash, endpoints[randIndex], v.digest, ruid)
					if err != nil {
						continue inner
					}
					metrics.GetOrRegisterResettingTimer("sliding-window.single.fetch-time", nil).UpdateSince(start)
					break inner
				}
			}

			if errored {
				break outer
			}
			networkDepth = i
			metrics.GetOrRegisterGauge("sliding-window.network-depth", nil).Update(int64(networkDepth))
		}
	}

	log.Info("sliding window test finished", "errored?", errored, "networkDepth", networkDepth, "networkDepth(kb)", networkDepth*filesize)
	log.Info("stats", "uploadedFiles", len(hashes), "uploadedKb", uploadedBytes/1000, "filesizeKb", filesize)

	return nil
}
