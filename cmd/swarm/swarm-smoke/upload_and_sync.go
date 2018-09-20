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
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

func generateEndpoints(scheme string, cluster string, from int, to int) {
	if cluster == "prod" {
		cluster = ""
	} else if cluster == "local" {
		for port := from; port <= to; port++ {
			endpoints = append(endpoints, fmt.Sprintf("%s://localhost:%v", scheme, port))
		}
		return
	} else {
		cluster = cluster + "."
	}

	for port := from; port <= to; port++ {
		endpoints = append(endpoints, fmt.Sprintf("%s://%v.%sswarm-gateways.net", scheme, port, cluster))
	}

	if includeLocalhost {
		endpoints = append(endpoints, "http://localhost:8500")
	}
}

func cliUploadAndSync(c *cli.Context) error {
	defer func(now time.Time) { log.Info("total time", "time", time.Since(now), "size (kb)", filesize) }(time.Now())

	generateEndpoints(scheme, cluster, from, to)

	log.Info("uploading to " + endpoints[0] + " and syncing")

	f, cleanup := generateRandomFile(filesize * 1000)
	defer cleanup()

	hash, err := upload(f, endpoints[0])
	if err != nil {
		log.Error(err.Error())
		return err
	}

	fhash, err := digest(f)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))

	time.Sleep(3 * time.Second)

	wg := sync.WaitGroup{}
	for _, endpoint := range endpoints {
		endpoint := endpoint
		ruid := uuid.New()[:8]
		wg.Add(1)
		go func(endpoint string, ruid string) {
			for {
				err := fetch(hash, endpoint, fhash, ruid)
				if err != nil {
					continue
				}

				wg.Done()
				return
			}
		}(endpoint, ruid)
	}
	wg.Wait()
	log.Info("all endpoints synced random file successfully")

	return nil
}

// fetch is getting the requested `hash` from the `endpoint` and compares it with the `original` file
func fetch(hash string, endpoint string, original []byte, ruid string) error {
	log.Trace("sleeping", "ruid", ruid)
	time.Sleep(3 * time.Second)

	log.Trace("http get request", "ruid", ruid, "api", endpoint, "hash", hash)
	res, err := http.Get(endpoint + "/bzz:/" + hash + "/")
	if err != nil {
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}
	log.Trace("http get response", "ruid", ruid, "api", endpoint, "hash", hash, "code", res.StatusCode, "len", res.ContentLength)

	if res.StatusCode != 200 {
		err := fmt.Errorf("expected status code %d, got %v", 200, res.StatusCode)
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}

	defer res.Body.Close()

	rdigest, err := digest(res.Body)
	if err != nil {
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}

	if !bytes.Equal(rdigest, original) {
		err := fmt.Errorf("downloaded imported file md5=%x is not the same as the generated one=%x", rdigest, original)
		log.Warn(err.Error(), "ruid", ruid)
		return err
	}

	log.Trace("downloaded file matches random file", "ruid", ruid, "len", res.ContentLength)

	return nil
}

// upload is uploading a file `f` to `endpoint` via the `swarm up` cmd
func upload(f *os.File, endpoint string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("swarm", "--bzzapi", endpoint, "up", f.Name())
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	hash := strings.TrimRight(out.String(), "\r\n")
	return hash, nil
}

func digest(r io.Reader) ([]byte, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// generateRandomFile is creating a temporary file with the requested byte size
func generateRandomFile(size int) (f *os.File, teardown func()) {
	// create a tmp file
	tmp, err := ioutil.TempFile("", "swarm-test")
	if err != nil {
		panic(err)
	}

	// callback for tmp file cleanup
	teardown = func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}

	buf := make([]byte, size)
	_, err = rand.Read(buf)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(tmp.Name(), buf, 0755)

	return tmp, teardown
}
