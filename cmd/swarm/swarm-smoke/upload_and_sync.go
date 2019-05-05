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
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/swarm/testutil"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadAndSyncCmd(ctx *cli.Context) error {
	// use input seed if it has been set
	if inputSeed != 0 {
		seed = inputSeed
	}

	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	errc := make(chan error)

	go func() {
		errc <- uploadAndSync(ctx, randomBytes)
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
	e := trackChunks(randomBytes[:], true)
	if e != nil {
		log.Error(e.Error())
	}

	return err
}

func trackChunks(testData []byte, submitMetrics bool) error {
	addrs, err := getAllRefs(testData)
	if err != nil {
		return err
	}

	for i, ref := range addrs {
		log.Debug(fmt.Sprintf("ref %d", i), "ref", ref)
	}

	var globalYes, globalNo int
	var globalMu sync.Mutex
	var hasErr bool

	var wg sync.WaitGroup
	wg.Add(len(hosts))

	var mu sync.Mutex                    // mutex protecting the allHostsChunks and bzzAddrs maps
	allHostChunks := map[string]string{} // host->bitvector of presence for chunks
	bzzAddrs := map[string]string{}      // host->bzzAddr

	for _, host := range hosts {
		host := host
		go func() {
			defer wg.Done()
			httpHost := fmt.Sprintf("ws://%s:%d", host, 8546)

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rpcClient, err := rpc.DialContext(ctx, httpHost)
			if rpcClient != nil {
				defer rpcClient.Close()
			}
			if err != nil {
				log.Error("error dialing host", "err", err, "host", httpHost)
				hasErr = true
				return
			}

			hostChunks, err := getChunksBitVectorFromHost(rpcClient, addrs)
			if err != nil {
				log.Error("error getting chunks bit vector from host", "err", err, "host", httpHost)
				hasErr = true
				return
			}

			bzzAddr, err := getBzzAddrFromHost(rpcClient)
			if err != nil {
				log.Error("error getting bzz addrs from host", "err", err, "host", httpHost)
				hasErr = true
				return
			}

			mu.Lock()
			allHostChunks[host] = hostChunks
			bzzAddrs[host] = bzzAddr
			mu.Unlock()

			yes, no := 0, 0
			for _, val := range hostChunks {
				if val == '1' {
					yes++
				} else {
					no++
				}
			}

			if no == 0 {
				log.Info("host reported to have all chunks", "host", host)
			}

			log.Debug("chunks", "chunks", hostChunks, "yes", yes, "no", no, "host", host)

			if submitMetrics {
				globalMu.Lock()
				globalYes += yes
				globalNo += no
				globalMu.Unlock()
			}
		}()
	}

	wg.Wait()

	checkChunksVsMostProxHosts(addrs, allHostChunks, bzzAddrs)

	if !hasErr && submitMetrics {
		// remove the chunks stored on the uploader node
		globalYes -= len(addrs)

		metrics.GetOrRegisterCounter("deployment.chunks.yes", nil).Inc(int64(globalYes))
		metrics.GetOrRegisterCounter("deployment.chunks.no", nil).Inc(int64(globalNo))
		metrics.GetOrRegisterCounter("deployment.chunks.refs", nil).Inc(int64(len(addrs)))
	}

	return nil
}

// getChunksBitVectorFromHost returns a bit vector of presence for a given slice of chunks from a given host
func getChunksBitVectorFromHost(client *rpc.Client, addrs []storage.Address) (string, error) {
	var hostChunks string

	err := client.Call(&hostChunks, "bzz_has", addrs)
	if err != nil {
		return "", err
	}

	return hostChunks, nil
}

// getBzzAddrFromHost returns the bzzAddr for a given host
func getBzzAddrFromHost(client *rpc.Client) (string, error) {
	var hive string

	err := client.Call(&hive, "bzz_hive")
	if err != nil {
		return "", err
	}

	// we make an ugly assumption about the output format of the hive.String() method
	// ideally we should replace this with an API call that returns the bzz addr for a given host,
	// but this also works for now (provided we don't change the hive.String() method, which we haven't in some time
	return strings.Split(strings.Split(hive, "\n")[3], " ")[10], nil
}

// checkChunksVsMostProxHosts is checking:
// 1. whether a chunk has been found at less than 2 hosts. Considering our NN size, this should not happen.
// 2. if a chunk is not found at its closest node. This should also not happen.
// Together with the --only-upload flag, we could run this smoke test and make sure that our syncing
// functionality is correct (without even trying to retrieve the content).
//
// addrs - a slice with all uploaded chunk refs
// allHostChunks - host->bit vector, showing what chunks are present on what hosts
// bzzAddrs - host->bzz address, used when determining the most proximate host for a given chunk
func checkChunksVsMostProxHosts(addrs []storage.Address, allHostChunks map[string]string, bzzAddrs map[string]string) {
	for k, v := range bzzAddrs {
		log.Trace("bzzAddr", "bzz", v, "host", k)
	}

	for i := range addrs {
		var foundAt int
		maxProx := -1
		var maxProxHost string
		for host := range allHostChunks {
			if allHostChunks[host][i] == '1' {
				foundAt++
			}

			ba, err := hex.DecodeString(bzzAddrs[host])
			if err != nil {
				panic(err)
			}

			// calculate the host closest to any chunk
			prox := chunk.Proximity(addrs[i], ba)
			if prox > maxProx {
				maxProx = prox
				maxProxHost = host
			}
		}

		if allHostChunks[maxProxHost][i] == '0' {
			log.Error("chunk not found at max prox host", "ref", addrs[i], "host", maxProxHost, "bzzAddr", bzzAddrs[maxProxHost])
		} else {
			log.Trace("chunk present at max prox host", "ref", addrs[i], "host", maxProxHost, "bzzAddr", bzzAddrs[maxProxHost])
		}

		// if chunk found at less than 2 hosts
		if foundAt < 2 {
			log.Error("chunk found at less than two hosts", "foundAt", foundAt, "ref", addrs[i])
		}
	}
}

func getAllRefs(testData []byte) (storage.AddressCollection, error) {
	datadir, err := ioutil.TempDir("", "chunk-debug")
	if err != nil {
		return nil, fmt.Errorf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(datadir)
	fileStore, err := storage.NewLocalFileStore(datadir, make([]byte, 32), chunk.NewTags())
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(testData)
	return fileStore.GetAllReferences(context.Background(), reader, false)
}

func uploadAndSync(c *cli.Context, randomBytes []byte) error {
	log.Info("uploading to "+httpEndpoint(hosts[0])+" and syncing", "seed", seed)

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

	log.Info("uploaded successfully", "hash", hash, "took", t2, "digest", fmt.Sprintf("%x", fhash))

	waitToSync()

	log.Debug("chunks before fetch attempt", "hash", hash)

	err = trackChunks(randomBytes, false)
	if err != nil {
		log.Error(err.Error())
	}

	if onlyUpload {
		log.Debug("only-upload is true, stoppping test", "hash", hash)
		return nil
	}

	randIndex := 1 + rand.Intn(len(hosts)-1)

	for {
		start := time.Now()
		err := fetch(hash, httpEndpoint(hosts[randIndex]), fhash, "")
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		ended := time.Since(start)

		metrics.GetOrRegisterResettingTimer("upload-and-sync.single.fetch-time", nil).Update(ended)
		log.Info("fetch successful", "took", ended, "endpoint", httpEndpoint(hosts[randIndex]))
		break
	}

	return nil
}

func isSyncing(wsHost string) (bool, error) {
	rpcClient, err := rpc.Dial(wsHost)
	if rpcClient != nil {
		defer rpcClient.Close()
	}

	if err != nil {
		log.Error("error dialing host", "err", err)
		return false, err
	}

	var isSyncing bool
	err = rpcClient.Call(&isSyncing, "bzz_isSyncing")
	if err != nil {
		log.Error("error calling host for isSyncing", "err", err)
		return false, err
	}

	log.Debug("isSyncing result", "host", wsHost, "isSyncing", isSyncing)

	return isSyncing, nil
}

func waitToSync() {
	t1 := time.Now()

	ns := uint64(1)

	for ns > 0 {
		time.Sleep(3 * time.Second)

		notSynced := uint64(0)
		var wg sync.WaitGroup
		wg.Add(len(hosts))
		for i := 0; i < len(hosts); i++ {
			i := i
			go func(idx int) {
				stillSyncing, err := isSyncing(wsEndpoint(hosts[idx]))

				if stillSyncing || err != nil {
					atomic.AddUint64(&notSynced, 1)
				}
				wg.Done()
			}(i)
		}
		wg.Wait()

		ns = atomic.LoadUint64(&notSynced)
	}

	t2 := time.Since(t1)
	metrics.GetOrRegisterResettingTimer("upload-and-sync.single.wait-for-sync.deployment", nil).Update(t2)
}
