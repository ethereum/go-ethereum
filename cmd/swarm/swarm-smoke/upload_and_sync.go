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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadAndSync(c *cli.Context) error {
	seed := int(time.Now().UnixNano() / 1e6)

	// test uuid
	tuid := uuid.New()[:8]

	log.Info("uploading to "+httpEndpoint(hosts[0])+" and syncing", "tuid", tuid, "seed", seed)

	h := md5.New()
	r := io.TeeReader(io.LimitReader(crand.Reader, int64(filesize*1000)), h)

	t1 := time.Now()
	hash, err := upload(r, filesize*1000, httpEndpoint(hosts[0]))
	if err != nil {
		log.Error(err.Error())
		return err
	}
	t2 := time.Since(t1)
	metrics.GetOrRegisterResettingTimer("upload-and-sync.upload-time", nil).Update(t2)

	fhash := h.Sum(nil)

	log.Info("uploaded successfully", "tuid", tuid, "hash", hash, "took", t2, "digest", fmt.Sprintf("%x", fhash))

	time.Sleep(time.Duration(syncDelay) * time.Second)

	wg := sync.WaitGroup{}
	if single {
		rand.Seed(time.Now().UTC().UnixNano())
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
