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
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/testutil"

	cli "gopkg.in/urfave/cli.v1"
)

var endpoint string

//just use the first endpoint
func generateEndpoint(scheme string, cluster string, app string, from int) {
	if cluster == "prod" {
		endpoint = fmt.Sprintf("%s://%v.swarm-gateways.net", scheme, from)
	} else {
		endpoint = fmt.Sprintf("%s://%s-%v-%s.stg.swarm-gateways.net", scheme, app, from, cluster)
	}
}

func cliUploadSpeed(c *cli.Context) error {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	metrics.GetOrRegisterCounter("upload-speed", nil).Inc(1)

	errc := make(chan error)
	go func() {
		errc <- uploadSpeed(c)
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter("upload-speed.fail", nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter("upload-speed.timeout", nil).Inc(1)
		return fmt.Errorf("timeout after %v sec", timeout)
	}
}

func uploadSpeed(c *cli.Context) error {
	defer func(now time.Time) {
		totalTime := time.Since(now)

		log.Info("total time", "time", totalTime, "kb", filesize)
		metrics.GetOrRegisterCounter("upload-speed.total-time", nil).Inc(int64(totalTime))
	}(time.Now())

	generateEndpoint(scheme, cluster, appName, from)
	seed := int(time.Now().UnixNano() / 1e6)
	log.Info("uploading to "+endpoint, "seed", seed)

	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	t1 := time.Now()
	hash, err := upload(&randomBytes, endpoint)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterCounter("upload-speed.upload-time", nil).Inc(int64(time.Since(t1)))

	fhash, err := digest(bytes.NewReader(randomBytes))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))
	return nil
}
