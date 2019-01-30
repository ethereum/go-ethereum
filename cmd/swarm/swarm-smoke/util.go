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
	"net/http"
	"net/http/httptrace"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/api/client"
	"github.com/ethereum/go-ethereum/swarm/spancontext"
	opentracing "github.com/opentracing/opentracing-go"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	commandName = ""
)

func wrapCliCommand(name string, killOnTimeout bool, command func(*cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		log.PrintOrigins(true)
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		defer func(now time.Time) {
			totalTime := time.Since(now)

			log.Info("total time", "time", totalTime)
			metrics.GetOrRegisterCounter(name+".total-time", nil).Inc(int64(totalTime))
		}(time.Now())

		log.Info("smoke test starting", "task", name, "timeout", timeout)
		commandName = name
		metrics.GetOrRegisterCounter(name, nil).Inc(1)

		errc := make(chan error)
		done := make(chan struct{})

		if killOnTimeout {
			go func() {
				<-time.After(time.Duration(timeout) * time.Second)
				close(done)
			}()
		}

		go func() {
			errc <- command(ctx)
		}()

		select {
		case err := <-errc:
			if err != nil {
				metrics.GetOrRegisterCounter(fmt.Sprintf("%s.fail", name), nil).Inc(1)
			}
			return err
		case <-done:
			metrics.GetOrRegisterCounter(fmt.Sprintf("%s.timeout", name), nil).Inc(1)
			return fmt.Errorf("timeout after %v sec", timeout)
		}
	}
}

func generateEndpoints(scheme string, cluster string, app string, from int, to int) {
	if cluster == "prod" {
		for port := from; port < to; port++ {
			endpoints = append(endpoints, fmt.Sprintf("%s://%v.swarm-gateways.net", scheme, port))
		}
	} else if cluster == "private-internal" {
		for port := from; port < to; port++ {
			endpoints = append(endpoints, fmt.Sprintf("%s://swarm-private-internal-%v:8500", scheme, port))
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

//just use the first endpoint
func generateEndpoint(scheme string, cluster string, app string, from int) string {
	if cluster == "prod" {
		return fmt.Sprintf("%s://%v.swarm-gateways.net", scheme, from)
	} else if cluster == "private-internal" {
		return fmt.Sprintf("%s://swarm-private-internal-%v:8500", scheme, from)
	} else {
		return fmt.Sprintf("%s://%s-%v-%s.stg.swarm-gateways.net", scheme, app, from, cluster)
	}
}

func fetchFeed(topic string, user string, endpoint string, original []byte, ruid string) error {
	ctx, sp := spancontext.StartSpan(context.Background(), "feed-and-sync.fetch")
	defer sp.Finish()

	log.Trace("sleeping", "ruid", ruid)
	time.Sleep(3 * time.Second)

	log.Trace("http get request (feed)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user)

	var tn time.Time
	reqUri := endpoint + "/bzz-feed:/?topic=" + topic + "&user=" + user
	req, _ := http.NewRequest("GET", reqUri, nil)

	opentracing.GlobalTracer().Inject(
		sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	trace := client.GetClientTrace("feed-and-sync - http get", "feed-and-sync", ruid, &tn)

	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))
	transport := http.DefaultTransport

	//transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	tn = time.Now()
	res, err := transport.RoundTrip(req)
	if err != nil {
		log.Error(err.Error(), "ruid", ruid)
		return err
	}

	log.Trace("http get response (feed)", "ruid", ruid, "api", endpoint, "topic", topic, "user", user, "code", res.StatusCode, "len", res.ContentLength)

	if res.StatusCode != 200 {
		return fmt.Errorf("expected status code %d, got %v (ruid %v)", 200, res.StatusCode, ruid)
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

// fetch is getting the requested `hash` from the `endpoint` and compares it with the `original` file
func fetch(hash string, endpoint string, original []byte, ruid string) error {
	ctx, sp := spancontext.StartSpan(context.Background(), "upload-and-sync.fetch")
	defer sp.Finish()

	log.Trace("http get request", "ruid", ruid, "api", endpoint, "hash", hash)

	var tn time.Time
	reqUri := endpoint + "/bzz:/" + hash + "/"
	req, _ := http.NewRequest("GET", reqUri, nil)

	opentracing.GlobalTracer().Inject(
		sp.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header))

	trace := client.GetClientTrace(commandName+" - http get", commandName, ruid, &tn)

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

// upload an arbitrary byte as a plaintext file  to `endpoint` using the api client
func upload(r io.Reader, size int, endpoint string) (string, error) {
	swarm := client.NewClient(endpoint)
	f := &client.File{
		ReadCloser: ioutil.NopCloser(r),
		ManifestEntry: api.ManifestEntry{
			ContentType: "text/plain",
			Mode:        0660,
			Size:        int64(size),
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
