package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

type ethClientStub struct {
	peersResult   uint64
	peersError    error
	blockResult   *types.Block
	blockError    error
	syncingResult *ethereum.SyncProgress
	syncingError  error
}

func (e *ethClientStub) PeerCount(_ context.Context) (uint64, error) {
	return e.peersResult, e.peersError
}

func (e *ethClientStub) BlockByNumber(_ context.Context, _ *big.Int) (*types.Block, error) {
	return e.blockResult, e.blockError
}

func (e *ethClientStub) SyncProgress(_ context.Context) (*ethereum.SyncProgress, error) {
	return e.syncingResult, e.syncingError
}

func TestProcessFromHeaders(t *testing.T) {
	cases := []struct {
		headers             []string
		clientPeerResult    uint64
		clientPeerError     error
		clientBlockResult   *types.Block
		clientBlockError    error
		clientSyncingResult *ethereum.SyncProgress
		clientSyncingError  error
		expectedStatusCode  int
		expectedBody        map[string]string
	}{
		// 0 - sync check enabled - syncing
		{
			headers:             []string{"synced"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				synced:           "OK",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 1 - sync check enabled - not syncing
		{
			headers:             []string{"synced"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: &ethereum.SyncProgress{},
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "ERROR: not synced",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 2 - sync check enabled - error checking sync
		{
			headers:             []string{"synced"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: &ethereum.SyncProgress{},
			clientSyncingError:  errors.New("problem checking sync"),
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "ERROR: problem checking sync",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 3 - peer count enabled - good request
		{
			headers:             []string{"min_peer_count1"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "OK",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 4 - peer count enabled - not enough peers
		{
			headers:             []string{"min_peer_count10"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "ERROR: not enough peers: 1 (minimum 10)",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 5 - peer count enabled - error checking peers
		{
			headers:             []string{"min_peer_count10"},
			clientPeerResult:    uint64(1),
			clientPeerError:     errors.New("problem checking peers"),
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "ERROR: problem checking peers",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 6 - peer count enabled - badly formed request
		{
			headers:             []string{"min_peer_countABC"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "ERROR: strconv.Atoi: parsing \"abc\": invalid syntax",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 7 - block check - all ok
		{
			headers:             []string{"check_block10"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "OK",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 8 - block check - no block found
		{
			headers:             []string{"check_block10"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    errors.New("not found"),
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "ERROR: not found",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 9 - block check - error checking block
		{
			headers:             []string{"check_block10"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    errors.New("problem checking block"),
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "ERROR: problem checking block",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 10 - block check - badly formed request
		{
			headers:             []string{"check_blockABC"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "ERROR: strconv.Atoi: parsing \"abc\": invalid syntax",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 11 - seconds check - all ok
		{
			headers:          []string{"max_seconds_behind60"},
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(-10 * time.Second).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "OK",
			},
		},
		// 12 - seconds check - too old
		{
			headers:          []string{"max_seconds_behind60"},
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(-1 * time.Hour).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "ERROR: timestamp too old: got ts:",
			},
		},
		// 13 - seconds check - less than 0 seconds
		{
			headers:          []string{"max_seconds_behind-1"},
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(1 * time.Hour).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "ERROR: invalid value provided",
			},
		},
		// 14 - seconds check - badly formed request
		{
			headers:             []string{"max_seconds_behindABC"},
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "ERROR: strconv.Atoi: parsing \"abc\": invalid syntax",
			},
		},
		// 15 - all checks - report ok
		{
			headers:          []string{"synced", "check_block10", "min_peer_count1", "max_seconds_behind60"},
			clientPeerResult: uint64(10),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(1 * time.Second).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				synced:           "OK",
				minPeerCount:     "OK",
				checkBlock:       "OK",
				maxSecondsBehind: "OK",
			},
		},
	}

	for idx, c := range cases {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "http://localhost:9090/health", nil)
		if err != nil {
			t.Errorf("%v: creating request: %v", idx, err)
		}

		for _, header := range c.headers {
			r.Header.Add("X-GETH-HEALTHCHECK", header)
		}

		ethClient := &ethClientStub{
			peersResult:   c.clientPeerResult,
			peersError:    c.clientPeerError,
			blockResult:   c.clientBlockResult,
			blockError:    c.clientBlockError,
			syncingResult: c.clientSyncingResult,
			syncingError:  c.clientSyncingError,
		}

		processFromHeaders(ethClient, r.Header.Values(healthHeader), w, r)

		result := w.Result()
		if result.StatusCode != c.expectedStatusCode {
			t.Errorf("%v: expected status code: %v, but got: %v", idx, c.expectedStatusCode, result.StatusCode)
		}

		bodyBytes, err := io.ReadAll(result.Body)
		if err != nil {
			t.Errorf("%v: reading response body: %s", idx, err)
		}

		var body map[string]string
		err = json.Unmarshal(bodyBytes, &body)
		if err != nil {
			t.Errorf("%v: unmarshalling the response body: %s", idx, err)
		}
		result.Body.Close()

		for k, v := range c.expectedBody {
			val, found := body[k]
			if !found {
				t.Errorf("%v: expected the key: %s to be in the response body but it wasn't there", idx, k)
			}
			if !strings.Contains(val, v) {
				t.Errorf("%v: expected the response body key: %s to contain: %s, but it contained: %s", idx, k, v, val)
			}
		}
	}
}

func TestProcessFromBody(t *testing.T) {
	cases := []struct {
		body                string
		clientPeerResult    uint64
		clientPeerError     error
		clientBlockResult   *types.Block
		clientBlockError    error
		clientSyncingResult *ethereum.SyncProgress
		clientSyncingError  error
		expectedStatusCode  int
		expectedBody        map[string]string
	}{
		// 0 - sync check enabled - syncing
		{
			body:                "{\"synced\": true}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "OK",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 1 - sync check enabled - not syncing
		{
			body:                "{\"synced\": true}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: &ethereum.SyncProgress{},
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "ERROR: not synced",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 2 - sync check enabled - error checking sync
		{
			body:                "{\"synced\": true}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: &ethereum.SyncProgress{},
			clientSyncingError:  errors.New("problem checking sync"),
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "ERROR: problem checking sync",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 3 - peer count enabled - good request
		{
			body:                "{\"min_peer_count\": 1}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "OK",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 4 - peer count enabled - not enough peers
		{
			body:                "{\"min_peer_count\": 10}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "ERROR: not enough peers: 1 (minimum 10)",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 5 - peer count enabled - error checking peers
		{
			body:                "{\"min_peer_count\": 10}",
			clientPeerResult:    uint64(1),
			clientPeerError:     errors.New("problem checking peers"),
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "ERROR: problem checking peers",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 6 - peer count enabled - badly formed request
		{
			body:                "{\"min_peer_count\": \"ABC\"}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "ERROR: json: cannot unmarshal string into Go struct field requestBody.min_peer_count of type uint",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 7 - block check - all ok
		{
			body:                "{\"check_block\": 10}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "OK",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 8 - block check - no block found
		{
			body:                "{\"check_block\": 10}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    errors.New("not found"),
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "ERROR: not found",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 9 - block check - error checking block
		{
			body:                "{\"check_block\": 10}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    errors.New("problem checking block"),
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "ERROR: problem checking block",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 10 - block check - badly formed request
		{
			body:                "{\"check_block\": \"ABC\"}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   nil,
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "ERROR: math/big: cannot unmarshal \"\\\"ABC\\\"\" into a *big.Int",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 11 - seconds check - all ok
		{
			body:             "{\"max_seconds_behind\": 60}",
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(-10 * time.Second).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "OK",
			},
		},
		// 12 - seconds check - too old
		{
			body:             "{\"max_seconds_behind\": 60}",
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(-1 * time.Hour).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "ERROR: timestamp too old: got ts:",
			},
		},
		// 13 - seconds check - less than 0 seconds
		{
			body:             "{\"max_seconds_behind\": -1}",
			clientPeerResult: uint64(1),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(1 * time.Hour).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "ERROR: invalid value provided",
			},
		},
		// 14 - seconds check - badly formed request
		{
			body:                "{\"max_seconds_behind\": \"ABC\"}",
			clientPeerResult:    uint64(1),
			clientPeerError:     nil,
			clientBlockResult:   &types.Block{},
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusInternalServerError,
			expectedBody: map[string]string{
				query:            "ERROR: json: cannot unmarshal string into Go struct field requestBody.max_seconds_behind of type int",
				synced:           "DISABLED",
				minPeerCount:     "DISABLED",
				checkBlock:       "DISABLED",
				maxSecondsBehind: "DISABLED",
			},
		},
		// 15 - all checks - report ok
		{
			body:             "{\"synced\": true, \"min_peer_count\": 1, \"check_block\": 10, \"max_seconds_behind\": 60}",
			clientPeerResult: uint64(10),
			clientPeerError:  nil,
			clientBlockResult: types.NewBlockWithHeader(&types.Header{
				Time: uint64(time.Now().Add(1 * time.Second).Unix()),
			}),
			clientBlockError:    nil,
			clientSyncingResult: nil,
			clientSyncingError:  nil,
			expectedStatusCode:  http.StatusOK,
			expectedBody: map[string]string{
				query:            "OK",
				synced:           "OK",
				minPeerCount:     "OK",
				checkBlock:       "OK",
				maxSecondsBehind: "OK",
			},
		},
	}

	for idx, c := range cases {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "http://localhost:9090/health", nil)
		if err != nil {
			t.Errorf("%v: creating request: %v", idx, err)
		}

		r.Body = io.NopCloser(strings.NewReader(c.body))

		ethClient := &ethClientStub{
			peersResult:   c.clientPeerResult,
			peersError:    c.clientPeerError,
			blockResult:   c.clientBlockResult,
			blockError:    c.clientBlockError,
			syncingResult: c.clientSyncingResult,
			syncingError:  c.clientSyncingError,
		}

		processFromBody(ethClient, w, r)

		result := w.Result()
		if result.StatusCode != c.expectedStatusCode {
			t.Errorf("%v: expected status code: %v, but got: %v", idx, c.expectedStatusCode, result.StatusCode)
		}

		bodyBytes, err := io.ReadAll(result.Body)
		if err != nil {
			t.Errorf("%v: reading response body: %s", idx, err)
		}

		var body map[string]string
		err = json.Unmarshal(bodyBytes, &body)
		if err != nil {
			t.Errorf("%v: unmarshalling the response body: %s", idx, err)
		}
		result.Body.Close()

		for k, v := range c.expectedBody {
			val, found := body[k]
			if !found {
				t.Errorf("%v: expected the key: %s to be in the response body but it wasn't there", idx, k)
			}
			if !strings.Contains(val, v) {
				t.Errorf("%v: expected the response body key: %s to contain: %s, but it contained: %s", idx, k, v, val)
			}
		}
	}
}
