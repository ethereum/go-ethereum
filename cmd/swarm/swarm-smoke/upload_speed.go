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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"

	cli "gopkg.in/urfave/cli.v1"
)

func uploadSpeed(c *cli.Context) error {
	endpoint := generateEndpoint(scheme, cluster, appName, from)
	seed := int(time.Now().UnixNano() / 1e6)
	log.Info("uploading to "+endpoint, "seed", seed)

	h := md5.New()
	r := io.TeeReader(io.LimitReader(crand.Reader, int64(filesize*1000)), h)

	t1 := time.Now()
	hash, err := upload(r, filesize*1000, endpoint)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterCounter("upload-speed.upload-time", nil).Inc(int64(time.Since(t1)))

	fhash := h.Sum(nil)

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))
	return nil
}
