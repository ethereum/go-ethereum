package node

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// createAndStartServerTB is will create a server instance with the given config and returns it.
func createAndStartServerTB(tb testing.TB, httpConfig *httpConfig, disableHTTP2 bool, wsConfig *wsConfig, apis []rpc.API) *httpServer {
	tb.Helper()
	srv := &httpServer{
		log:          log.New("rpc-bench"),
		timeouts:     rpc.DefaultHTTPTimeouts,
		disableHTTP2: disableHTTP2,
	}
	if err := srv.setListenAddr("127.0.0.1", 0); err != nil {
		tb.Fatal("failed to set listen address:", err)
	}
	if err := srv.enableRPC(apis, *httpConfig); err != nil {
		tb.Fatal("failed to enable RPC:", err)
	}
	if err := srv.enableWS(apis, *wsConfig); err != nil {
		tb.Fatal("failed to enable WS:", err)
	}
	if err := srv.start(); err != nil {
		tb.Fatal("failed to start HTTP server:", err)
	}
	return srv
}

// newHTTP1Client returns a vanilla HTTP/1.1 client.
func newHTTP1Client() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        2048,
		MaxIdleConnsPerHost: 2048,
		IdleConnTimeout:     90 * time.Second,
	}
	tr.Protocols = new(http.Protocols)
	tr.Protocols.SetHTTP1(true)
	return &http.Client{Transport: tr}
}

// newH2CClient returns a client that speaks unencrypted HTTP/2 (H2C).
func newH2CClient() *http.Client {
	tr := &http.Transport{}
	tr.Protocols = new(http.Protocols)
	tr.Protocols.SetUnencryptedHTTP2(true)
	return &http.Client{Transport: tr}
}

// rpcPayload is a minimal valid JSON-RPC 2.0 request.
const rpcPayload = `{"jsonrpc":"2.0","id":1,"method":"rpc_modules","params":[]}`

// doRequest sends one JSON-RPC POST and discards the body. It returns the
// round-trip latency in nanoseconds and any error.
func doRequest(client *http.Client, url string) (int64, error) {
	start := time.Now()
	resp, err := client.Post(url, "application/json",
		strings.NewReader(rpcPayload))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection is reusable
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return time.Since(start).Nanoseconds(), nil
}

// latencyHistogram is a lock-free accumulator for percentile reporting.
type latencyHistogram struct {
	mu      sync.Mutex
	samples []int64
}

func (h *latencyHistogram) record(ns int64) {
	h.mu.Lock()
	h.samples = append(h.samples, ns)
	h.mu.Unlock()
}

func (h *latencyHistogram) report(b *testing.B) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.samples) == 0 {
		return
	}
	sort.Slice(h.samples, func(i, j int) bool { return h.samples[i] < h.samples[j] })
	n := len(h.samples)
	p := func(pct float64) float64 {
		idx := int(math.Ceil(pct/100.0*float64(n))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return float64(h.samples[idx]) / 1e6 // ns → ms
	}
	b.ReportMetric(p(50), "p50_ms")
	b.ReportMetric(p(95), "p95_ms")
	b.ReportMetric(p(99), "p99_ms")
}

// benchConcurrent is the shared driver used by every sub-benchmark.
//
//	client      – pre-configured HTTP client (HTTP/1 or H2C)
//	url         – full RPC endpoint URL
//	concurrency – number of parallel goroutines hammering the server
func benchConcurrent(b *testing.B, client *http.Client, url string, concurrency int) {
	b.Helper()

	hist := &latencyHistogram{}
	var errors atomic.Int64

	b.SetParallelism(concurrency)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ns, err := doRequest(client, url)
			if err != nil {
				errors.Add(1)
				continue
			}
			hist.record(ns)
		}
	})

	b.StopTimer()
	hist.report(b)

	if e := errors.Load(); e > 0 {
		b.Logf("WARN: %d requests failed", e)
	}
}

var concurrencyLevels = []int{1, 10, 50, 100, 500}

// BenchmarkRPCHTTP1 measures the HTTP/1.1 baseline.
func BenchmarkRPCHTTP1(b *testing.B) {
	srv := createAndStartServerTB(b, &httpConfig{}, true /*disableHTTP2*/, &wsConfig{}, nil)
	defer srv.stop()

	url := "http://" + srv.listenAddr()
	client := newHTTP1Client()

	for _, c := range concurrencyLevels {
		c := c
		b.Run(fmt.Sprintf("concurrency=%d", c), func(b *testing.B) {
			benchConcurrent(b, client, url, c)
		})
	}
}

// BenchmarkRPCH2C measures HTTP/2 cleartext (H2C).
func BenchmarkRPCH2C(b *testing.B) {
	srv := createAndStartServerTB(b, &httpConfig{}, false /*disableHTTP2*/, &wsConfig{}, nil)
	defer srv.stop()

	url := "http://" + srv.listenAddr()
	client := newH2CClient()

	for _, c := range concurrencyLevels {
		c := c
		b.Run(fmt.Sprintf("concurrency=%d", c), func(b *testing.B) {
			benchConcurrent(b, client, url, c)
		})
	}
}

// BenchmarkRPCH2CvsHTTP1SameServer exercises both protocols against the same
// server instance (H2C enabled) to isolate protocol overhead from any
// server-startup variance.
func BenchmarkRPCH2CvsHTTP1SameServer(b *testing.B) {
	srv := createAndStartServerTB(b, &httpConfig{}, false /*disableHTTP2*/, &wsConfig{}, nil)
	defer srv.stop()

	url := "http://" + srv.listenAddr()

	protocols := []struct {
		name   string
		client *http.Client
	}{
		{"HTTP1", newHTTP1Client()},
		{"H2C", newH2CClient()},
	}

	for _, proto := range protocols {
		proto := proto
		for _, c := range concurrencyLevels {
			c := c
			b.Run(fmt.Sprintf("%s/concurrency=%d", proto.name, c), func(b *testing.B) {
				benchConcurrent(b, proto.client, url, c)
			})
		}
	}
}
