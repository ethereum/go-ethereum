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
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadAndSyncCmd(ctx *cli.Context, tuid string) error {
	// use input seed if it has been set
	if inputSeed != 0 {
		seed = inputSeed
	}

	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	errc := make(chan error)

	go func() {
		errc <- uploadAndSync(ctx, randomBytes, tuid)
	}()

	var err error
	select {
	case err = <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", commandName), nil).Inc(1)
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter(fmt.Sprintf("%s.timeout", commandName), nil).Inc(1)

		err = fmt.Errorf("timeout after %v sec", timeout)
	}

	// trigger debug functionality on randomBytes
	e := trackChunks(randomBytes[:])
	if e != nil {
		log.Error(e.Error())
	}

	return err
}

func trackChunks(testData []byte) error {
	addrs, err := getAllRefs(testData)
	if err != nil {
		return err
	}

	for i, ref := range addrs {
		log.Trace(fmt.Sprintf("ref %d", i), "ref", ref)
	}

	for _, host := range hosts {
		httpHost := fmt.Sprintf("ws://%s:%d", host, 8546)

		hostChunks := []string{}

		rpcClient, err := rpc.Dial(httpHost)
		if err != nil {
			log.Error("error dialing host", "err", err, "host", httpHost)
			continue
		}

		var hasInfo []api.HasInfo
		err = rpcClient.Call(&hasInfo, "bzz_has", addrs)
		if err != nil {
			log.Error("error calling rpc client", "err", err, "host", httpHost)
			continue
		}

		count := 0
		for _, info := range hasInfo {
			if info.Has {
				hostChunks = append(hostChunks, "1")
			} else {
				hostChunks = append(hostChunks, "0")
				count++
			}
		}

		if count == 0 {
			log.Info("host reported to have all chunks", "host", host)
		}

		log.Trace("chunks", "chunks", strings.Join(hostChunks, ""), "host", host)
	}
	return nil
}

func getAllRefs(testData []byte) (storage.AddressCollection, error) {
	datadir, err := ioutil.TempDir("", "chunk-debug")
	if err != nil {
		return nil, fmt.Errorf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(datadir)
	fileStore, err := storage.NewLocalFileStore(datadir, make([]byte, 32))
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(trackTimeout)*time.Second)
	defer cancel()

	reader := bytes.NewReader(testData)
	return fileStore.GetAllReferences(ctx, reader, false)
}

func uploadAndSync(c *cli.Context, randomBytes []byte, tuid string) error {
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
