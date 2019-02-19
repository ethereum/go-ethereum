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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/testutil"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadSpeedCmd(ctx *cli.Context, tuid string) error {
	log.Info("uploading to "+hosts[0], "tuid", tuid, "seed", seed)
	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	errc := make(chan error)

	go func() {
		errc <- uploadSpeed(ctx, tuid, randomBytes)
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

func uploadSpeed(c *cli.Context, tuid string, data []byte) error {
	t1 := time.Now()
	hash, err := upload(data, hosts[0])
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterCounter("upload-speed.upload-time", nil).Inc(int64(time.Since(t1)))

	fhash, err := digest(bytes.NewReader(data))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))
	return nil
}
