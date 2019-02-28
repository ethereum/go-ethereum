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

		e := fmt.Errorf("timeout after %v sec", timeout)
		// trigger debug functionality on randomBytes
		err := trackChunks(randomBytes[:])
		if err != nil {
			e = fmt.Errorf("%v; triggerChunkDebug failed: %v", e, err)
		}

		return e
	}
}

func trackChunks(testData []byte) error {
	log.Warn("Test timed out; running chunk debug sequence")

	addrs, err := getAllRefs(testData)
	if err != nil {
		return err
	}
	log.Trace("All references retrieved")

	// has-chunks
	for _, host := range hosts {
		httpHost := fmt.Sprintf("ws://%s:%d", host, 8546)
		log.Trace("Calling `Has` on host", "httpHost", httpHost)
		rpcClient, err := rpc.Dial(httpHost)
		if err != nil {
			log.Trace("Error dialing host", "err", err)
			return err
		}
		log.Trace("rpc dial ok")
		var hasInfo []api.HasInfo
		err = rpcClient.Call(&hasInfo, "bzz_has", addrs)
		if err != nil {
			log.Trace("Error calling host", "err", err)
			return err
		}
		log.Trace("rpc call ok")
		count := 0
		for _, info := range hasInfo {
			if !info.Has {
				count++
				log.Error("Host does not have chunk", "host", httpHost, "chunk", info.Addr)
			}
		}
		if count == 0 {
			log.Info("Host reported to have all chunks", "host", httpHost)
		}
	}
	return nil
}

func getAllRefs(testData []byte) (storage.AddressCollection, error) {
	log.Trace("Getting all references for given root hash")
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
