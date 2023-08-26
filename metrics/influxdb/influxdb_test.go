// Copyright 2023 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package influxdb

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/metrics"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func TestMain(m *testing.M) {
	metrics.Enabled = true
	os.Exit(m.Run())
}

func setupSampleRegistry(t *testing.T) metrics.Registry {
	t.Helper()
	r := metrics.NewOrderedRegistry()
	metrics.NewRegisteredGaugeInfo("info", r).Update(metrics.GaugeInfoValue{
		"version":           "1.10.18-unstable",
		"arch":              "amd64",
		"os":                "linux",
		"commit":            "7caa2d8163ae3132c1c2d6978c76610caee2d949",
		"protocol_versions": "64 65 66",
	})
	metrics.NewRegisteredGaugeFloat64("pi", r).Update(3.14)
	metrics.NewRegisteredCounter("months", r).Inc(12)
	metrics.NewRegisteredCounterFloat64("tau", r).Inc(1.57)
	metrics.NewRegisteredMeter("elite", r).Mark(1337)
	metrics.NewRegisteredTimer("second", r).Update(time.Second)
	metrics.NewRegisteredCounterFloat64("tau", r).Inc(1.57)
	metrics.NewRegisteredCounterFloat64("tau", r).Inc(1.57)
	return r
}

func TestExampleV1(t *testing.T) {
	r := setupSampleRegistry(t)
	var have, want string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		haveB, _ := io.ReadAll(r.Body)
		have = string(haveB)
		r.Body.Close()
	}))
	defer ts.Close()
	u, _ := url.Parse(ts.URL)
	rep := &reporter{
		reg:       r,
		url:       *u,
		namespace: "goth.",
	}
	if err := rep.makeClient(); err != nil {
		t.Fatal(err)
	}
	if err := rep.send(978307200); err != nil {
		t.Fatal(err)
	}
	if wantB, err := os.ReadFile("./testdata/influxdbv1.want"); err != nil {
		t.Fatal(err)
	} else {
		want = string(wantB)
	}
	if have != want {
		t.Errorf("\nhave:\n%v\nwant:\n%v\n", have, want)
		t.Logf("have vs want:\n %v", findFirstDiffPos(have, want))
	}
}

func TestExampleV2(t *testing.T) {
	r := setupSampleRegistry(t)
	var have, want string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		haveB, _ := io.ReadAll(r.Body)
		have = string(haveB)
		r.Body.Close()
	}))
	defer ts.Close()

	rep := &v2Reporter{
		reg:       r,
		endpoint:  ts.URL,
		namespace: "goth.",
	}
	rep.client = influxdb2.NewClient(rep.endpoint, rep.token)
	defer rep.client.Close()
	rep.write = rep.client.WriteAPI(rep.organization, rep.bucket)

	rep.send(978307200)

	if wantB, err := os.ReadFile("./testdata/influxdbv2.want"); err != nil {
		t.Fatal(err)
	} else {
		want = string(wantB)
	}
	if have != want {
		t.Errorf("\nhave:\n%v\nwant:\n%v\n", have, want)
		t.Logf("have vs want:\n %v", findFirstDiffPos(have, want))
	}
}

func findFirstDiffPos(a, b string) string {
	x, y := []byte(a), []byte(b)
	var res []byte
	for i, ch := range x {
		if i > len(y) {
			res = append(res, ch)
			res = append(res, fmt.Sprintf("<-- diff: %#x vs EOF", ch)...)
			break
		}
		if ch != y[i] {
			res = append(res, fmt.Sprintf("<-- diff: %#x (%c) vs %#x (%c)", ch, ch, y[i], y[i])...)
			break
		}
		res = append(res, ch)
	}
	if len(res) > 100 {
		res = res[len(res)-100:]
	}
	return string(res)
}
