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
	"crypto/md5"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	"github.com/ethereum/go-ethereum/swarm/testutil"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pborman/uuid"

	cli "gopkg.in/urfave/cli.v1"
)

func generateEndpoints(scheme string, cluster string, app string, from int, to int) {
	if cluster == "prod" {
		for port := from; port < to; port++ {
			endpoints = append(endpoints, fmt.Sprintf("%s://%v.swarm-gateways.net", scheme, port))
		}
	} else {
		for port := from; port < to; port++ {
			endpoints = append(endpoints, fmt.Sprintf("%s://%s-%v-%s.stg.swarm-gateways.net", scheme, app, port, cluster))
		}
	}

	if includeLocalhost {
		endpoints = append(endpoints, "http://localhost:8500")
	}
}

func cliUploadAndSync(c *cli.Context) error {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	metrics.GetOrRegisterCounter("upload-and-sync", nil).Inc(1)

	errc := make(chan error)
	go func() {
		errc <- uploadAndSync(c)
	}()

	select {
	case err := <-errc:
		if err != nil {
			metrics.GetOrRegisterCounter("upload-and-sync.fail", nil).Inc(1)
		}
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		metrics.GetOrRegisterCounter("upload-and-sync.timeout", nil).Inc(1)
		return fmt.Errorf("timeout after %v sec", timeout)
	}
}

func uploadAndSync(c *cli.Context) error {
	defer func(now time.Time) {
		totalTime := time.Since(now)

		log.Info("total time", "time", totalTime, "kb", filesize)
		metrics.GetOrRegisterCounter("upload-and-sync.total-time", nil).Inc(int64(totalTime))
	}(time.Now())

	generateEndpoints(scheme, cluster, appName, from, to)
	seed := int(time.Now().UnixNano() / 1e6)
	log.Info("uploading to "+endpoints[0]+" and syncing", "seed", seed)

	randomBytes := testutil.RandomBytes(seed, filesize*1000)

	t1 := time.Now()
	hash, err := upload(&randomBytes, endpoints[0])
	if err != nil {
		log.Error(err.Error())
		return err
	}
	metrics.GetOrRegisterCounter("upload-and-sync.upload-time", nil).Inc(int64(time.Since(t1)))

	fhash, err := digest(bytes.NewReader(randomBytes))
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("uploaded successfully", "hash", hash, "digest", fmt.Sprintf("%x", fhash))

	time.Sleep(time.Duration(syncDelay) * time.Second)

	wg := sync.WaitGroup{}
	if single {
		rand.Seed(time.Now().UTC().UnixNano())
		randIndex := 1 + rand.Intn(len(endpoints)-1)
		ruid := uuid.New()[:8]
		wg.Add(1)
		go func(endpoint string, ruid string) {
			for {
				start := time.Now()
				err := fetch(hash, endpoint, fhash, ruid)
				fetchTime := time.Since(start)
				if err != nil {
					continue
				}

				metrics.GetOrRegisterMeter("upload-and-sync.single.fetch-time", nil).Mark(int64(fetchTime))
				wg.Done()
				return
			}
		}(endpoints[randIndex], ruid)
	} else {
		for _, endpoint := range endpoints {
			ruid := uuid.New()[:8]
			wg.Add(1)
			go func(endpoint string, ruid string) {
				for {
					start := time.Now()
					err := fetch(hash, endpoint, fhash, ruid)
					fetchTime := time.Since(start)
					if err != nil {
						continue
					}

					metrics.GetOrRegisterMeter("upload-and-sync.each.fetch-time", nil).Mark(int64(fetchTime))
					wg.Done()
					return
				}
			}(endpoint, ruid)
		}
	}
	wg.Wait()
	log.Info("all endpoints synced random file successfully")

	return nil
}

// fetch is getting the requested `hash` from the `endpoint` and compares it with the `original` file
func fetch(hash string, endpoint string, original []byte, ruid string) error {
	ctx, sp := spancontext.StartSpan(context.Background(), "upload-and-sync.fetch")
	defer sp.Finish()

	log.Trace("sleeping", "ruid", ruid)
	time.Sleep(3 * time.Second)
	log.Trace("http get request", "ruid", ruid, "api", endpoint, "hash", hash)

	var tn time.Time
	reqUri := endpoint + "/bzz:/" + hash + "/"
	req, _ := http.NewRequest("GET", reqUri, nil)

	opentracing.GlobalTracer().Inject(
		sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	trace := client.GetClientTrace("upload-and-sync - http get", "upload-and-sync", ruid, &tn)

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))
	transport := http.DefaultTransport

	//transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	tn = time.Now()
	res, err := transport.RoundTrip(req)
	if err != nil {
		log.Error(err.Error(), "ruid", ruid)
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
func upload(dataBytes *[]byte, endpoint string) (string, error) {
	swarm := client.NewClient(endpoint)
	f := &client.File{
		ReadCloser: ioutil.NopCloser(bytes.NewReader(*dataBytes)),
		ManifestEntry: api.ManifestEntry{
			ContentType: "text/plain",
			Mode:        0660,
			Size:        int64(len(*dataBytes)),
		},
	}

	// upload data to bzz:// and retrieve the content-addressed manifest hash, hex-encoded.
	return swarm.Upload(f, "", false)
}

func digest(r io.Reader) ([]byte, error) {
	h := md5.New()
	_, err := io.Copy(h, r)
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// generates random data in heap buffer
func generateRandomData(datasize int) ([]byte, error) {
	b := make([]byte, datasize)
	c, err := crand.Read(b)
	if err != nil {
		return nil, err
	} else if c != datasize {
		return nil, errors.New("short read")
	}
	return b, nil
}
