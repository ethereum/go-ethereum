// Copyright 2026 The go-ethereum Authors
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

package engineapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	sszt "github.com/ethereum/go-ethereum/beacon/engine/ssz"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params/forks"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

// stubBackend captures requests to verify the router routes correctly.
type stubBackend struct {
	fcuStatus engine.PayloadStatusV1
	fcuID     *engine.PayloadID
	npStatus  engine.PayloadStatusV1
	npErr     error

	lastFCUState  engine.ForkchoiceStateV1
	lastFCUAttrs  *engine.PayloadAttributes
	lastNPData    engine.ExecutableData
	lastNPHashes  []common.Hash
	lastNPRequest [][]byte

	envelope *engine.ExecutionPayloadEnvelope
	getErr   error

	bodies     []*types.Body
	bodyTimes  []uint64
	v1Blobs    []*engine.BlobAndProofV1
	v2Blobs    []*engine.BlobAndProofV2
	forkAtTime func(uint64) forks.Fork
}

func (s *stubBackend) ForkchoiceUpdated(_ context.Context, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes, _ engine.PayloadVersion) (engine.ForkChoiceResponse, error) {
	s.lastFCUState = state
	s.lastFCUAttrs = attrs
	return engine.ForkChoiceResponse{PayloadStatus: s.fcuStatus, PayloadID: s.fcuID}, nil
}

func (s *stubBackend) NewPayload(_ context.Context, data engine.ExecutableData, hashes []common.Hash, _ *common.Hash, requests [][]byte) (engine.PayloadStatusV1, error) {
	s.lastNPData = data
	s.lastNPHashes = hashes
	s.lastNPRequest = requests
	return s.npStatus, s.npErr
}

func (s *stubBackend) GetPayload(engine.PayloadID, []forks.Fork) (*engine.ExecutionPayloadEnvelope, error) {
	return s.envelope, s.getErr
}

func (s *stubBackend) GetBlobs([]common.Hash, bool) ([]*engine.BlobAndProofV2, []*engine.BlobAndProofV1, error) {
	return s.v2Blobs, s.v1Blobs, nil
}

func (s *stubBackend) BodiesByHash(hashes []common.Hash) ([]*types.Body, []uint64) {
	return s.bodies, s.bodyTimes
}

func (s *stubBackend) BodiesByRange(from, count uint64) ([]*types.Body, []uint64) {
	return s.bodies, s.bodyTimes
}

func (s *stubBackend) ForkFromTimestamp(ts uint64) forks.Fork {
	if s.forkAtTime != nil {
		return s.forkAtTime(ts)
	}
	return forks.Amsterdam
}

func (s *stubBackend) ClientVersion() engine.ClientVersionV1 {
	return engine.ClientVersionV1{Code: "GE", Name: "geth", Version: "test", Commit: "0xdeadbeef"}
}

func newTestServer(t *testing.T, b Backend) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.StripPrefix(BasePath, NewRouter(b)))
}

func sszPost(t *testing.T, srv *httptest.Server, path string, body ssz.Object, fork ssz.Fork) *http.Response {
	t.Helper()
	buf := make([]byte, ssz.SizeOnFork(body, fork))
	if err := ssz.EncodeToBytesOnFork(buf, body, fork); err != nil {
		t.Fatalf("encode: %v", err)
	}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+BasePath+path, bytes.NewReader(buf))
	req.Header.Set("Content-Type", sszContentType)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("post %s: %v", path, err)
	}
	return resp
}

func decodeSSZ[T ssz.Object](t *testing.T, resp *http.Response, obj T, fork ssz.Fork) {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := ssz.DecodeFromBytesOnFork(body, obj, fork); err != nil {
		t.Fatalf("decode response: %v\nbody: %x", err, body)
	}
}

func TestRouterNewPayload(t *testing.T) {
	id := engine.PayloadID{1, 2, 3}
	b := &stubBackend{
		npStatus: engine.PayloadStatusV1{Status: engine.VALID, LatestValidHash: &common.Hash{0xaa}},
		fcuID:    &id,
	}
	srv := newTestServer(t, b)
	defer srv.Close()

	amsterdam, _ := sszt.ForkFor(forks.Amsterdam)
	blob, excess, slot := uint64(0), uint64(0), uint64(0)
	env := &sszt.ExecutionPayloadEnvelopeAmsterdam{
		Payload: &sszt.ExecutionPayload{
			BaseFeePerGas: uint256.NewInt(7e9),
			LogsBloom:     [256]byte{},
			BlobGasUsed:   &blob,
			ExcessBlobGas: &excess,
			SlotNumber:    &slot,
		},
		ParentBeaconBlockRoot: common.Hash{0x55},
	}
	resp := sszPost(t, srv, "/amsterdam/payloads", env, amsterdam)
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	got := new(sszt.PayloadStatus)
	decodeSSZ(t, resp, got, amsterdam)
	if got.Status != sszt.StatusValid {
		t.Errorf("status=%d want VALID(%d)", got.Status, sszt.StatusValid)
	}
	if len(got.LatestValidHash) != 1 || got.LatestValidHash[0] != (common.Hash{0xaa}) {
		t.Errorf("LatestValidHash=%v", got.LatestValidHash)
	}
}

func TestRouterForkchoice(t *testing.T) {
	id := engine.PayloadID{9, 9, 9}
	b := &stubBackend{
		fcuStatus: engine.PayloadStatusV1{Status: engine.VALID},
		fcuID:     &id,
	}
	srv := newTestServer(t, b)
	defer srv.Close()

	amsterdam, _ := sszt.ForkFor(forks.Amsterdam)
	fcu := &sszt.ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &sszt.ForkchoiceState{HeadBlockHash: common.Hash{0xaa}},
	}
	resp := sszPost(t, srv, "/amsterdam/forkchoice", fcu, amsterdam)
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	got := new(sszt.ForkchoiceUpdateResponseAmsterdam)
	decodeSSZ(t, resp, got, amsterdam)
	if got.PayloadStatus == nil || got.PayloadStatus.Status != sszt.StatusValid {
		t.Errorf("status: %#v", got.PayloadStatus)
	}
	if len(got.PayloadID) != 1 || got.PayloadID[0] != [8]byte(id) {
		t.Errorf("PayloadID=%v want %v", got.PayloadID, id)
	}
}

func TestRouterUnsupportedFork(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()

	// Bogus fork in URL: the router does not recognise the segment at all.
	resp, err := srv.Client().Post(srv.URL+BasePath+"/bogus/payloads", sszContentType, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("want 400, got %d", resp.StatusCode)
	}
}

// TestRouterAdvertisedForkRoutable asserts that a fork the capabilities
// endpoint advertises (here osaka) is actually routable end-to-end: forkchoice
// with attributes, newPayload, and getPayload all succeed. This guards against
// advertising a fork the handlers reject (the failure mode where capabilities
// outran the per-handler fork gating).
func TestRouterAdvertisedForkRoutable(t *testing.T) {
	id := engine.PayloadID{4, 4, 4}
	env := &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: &engine.ExecutableData{LogsBloom: make([]byte, 256)},
		BlockValue:       uint256.NewInt(0).ToBig(),
	}
	b := &stubBackend{
		fcuStatus: engine.PayloadStatusV1{Status: engine.VALID},
		fcuID:     &id,
		npStatus:  engine.PayloadStatusV1{Status: engine.VALID},
		envelope:  env,
		// Osaka chain sitting in a BPO era: ForkFromTimestamp returns a BPO
		// fork, which must collapse onto the osaka URL fork.
		forkAtTime: func(uint64) forks.Fork { return forks.BPO1 },
	}
	srv := newTestServer(t, b)
	defer srv.Close()

	osaka, _ := sszt.ForkFor(forks.Osaka)
	blob, excess := uint64(0), uint64(0)

	// forkchoice with payload attributes (proposal path).
	fcu := &sszt.ForkchoiceUpdateAmsterdam{
		ForkchoiceState: &sszt.ForkchoiceState{HeadBlockHash: common.Hash{0xaa}},
		PayloadAttributes: []*sszt.PayloadAttributes{{
			Timestamp:             1,
			ParentBeaconBlockRoot: &common.Hash{0x55},
		}},
	}
	resp := sszPost(t, srv, "/osaka/forkchoice", fcu, osaka)
	if resp.StatusCode != 200 {
		t.Fatalf("forkchoice: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()
	if b.lastFCUAttrs == nil {
		t.Fatal("forkchoice: attributes not forwarded to backend")
	}

	// newPayload.
	penv := &sszt.ExecutionPayloadEnvelopeAmsterdam{
		Payload: &sszt.ExecutionPayload{
			BaseFeePerGas: uint256.NewInt(7e9),
			BlobGasUsed:   &blob,
			ExcessBlobGas: &excess,
		},
		ParentBeaconBlockRoot: common.Hash{0x55},
	}
	resp = sszPost(t, srv, "/osaka/payloads", penv, osaka)
	if resp.StatusCode != 200 {
		t.Fatalf("newPayload: want 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// getPayload.
	resp, err := srv.Client().Get(srv.URL + BasePath + "/osaka/payloads/0x0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("getPayload: want 200, got %d", resp.StatusCode)
	}
}

func TestRouterUnsupportedMediaType(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()
	resp, err := srv.Client().Post(srv.URL+BasePath+"/amsterdam/payloads", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 415 {
		t.Fatalf("want 415, got %d", resp.StatusCode)
	}
}

func TestRouterCapabilities(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + BasePath + "/capabilities")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var c capabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		t.Fatal(err)
	}
	// Capabilities must advertise every fork the router routes, so a consensus
	// client negotiates SSZ for all of them rather than falling back to JSON-RPC.
	// amsterdam is routed but not yet fully implemented, so it is intentionally
	// not advertised for now.
	want := map[string]bool{"paris": true, "shanghai": true, "cancun": true, "prague": true, "osaka": true}
	if len(c.SupportedForks) != len(want) {
		t.Errorf("supported forks: got %v, want all of %v", c.SupportedForks, want)
	}
	for _, f := range c.SupportedForks {
		if !want[f] {
			t.Errorf("unexpected fork advertised: %q", f)
		}
	}
	if got := c.Limits["blobs.max_versioned_hashes"]; got != sszt.MaxBlobsRequest {
		t.Errorf("blobs limit: %d", got)
	}
}

func TestRouterIdentity(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + BasePath + "/identity")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	var body struct {
		Versions []engine.ClientVersionV1 `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Versions) != 1 || body.Versions[0].Code != "GE" {
		t.Errorf("identity: %#v", body)
	}
}

func TestRouterTrailingSlash404(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + BasePath + "/capabilities/")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("want 404 for trailing slash, got %d", resp.StatusCode)
	}
}

func TestRouterSSZDecodeError(t *testing.T) {
	srv := newTestServer(t, &stubBackend{})
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+BasePath+"/amsterdam/payloads", bytes.NewReader([]byte{0xff}))
	req.Header.Set("Content-Type", sszContentType)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("want 400 ssz-decode-error, got %d", resp.StatusCode)
	}
	var p problem
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.Type != ErrSSZDecode {
		t.Errorf("type=%s want %s", p.Type, ErrSSZDecode)
	}
}

func TestRouterBlobsV2AllOrNothing(t *testing.T) {
	// One requested hash, but backend returns a nil entry -> /v2 should 204.
	b := &stubBackend{v2Blobs: []*engine.BlobAndProofV2{nil}}
	srv := newTestServer(t, b)
	defer srv.Close()
	req := &sszt.BlobsVersionedHashesRequest{VersionedHashes: []common.Hash{{0x01}}}
	resp := sszPost(t, srv, "/blobs/v2", req, ssz.ForkUnknown)
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatalf("want 204, got %d", resp.StatusCode)
	}
}

func TestRouterBlobsV3PartialOK(t *testing.T) {
	b := &stubBackend{v2Blobs: []*engine.BlobAndProofV2{nil, {Blob: make([]byte, sszt.BytesPerBlob)}}}
	srv := newTestServer(t, b)
	defer srv.Close()
	req := &sszt.BlobsVersionedHashesRequest{VersionedHashes: []common.Hash{{0x01}, {0x02}}}
	resp := sszPost(t, srv, "/blobs/v3", req, ssz.ForkUnknown)
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	got := new(sszt.BlobsV2Response)
	decodeSSZ(t, resp, got, ssz.ForkUnknown)
	if len(got.Entries) != 2 {
		t.Fatalf("entries=%d", len(got.Entries))
	}
	if got.Entries[0].Available {
		t.Error("entry 0 should be unavailable")
	}
	if !got.Entries[1].Available {
		t.Error("entry 1 should be available")
	}
}

func TestRouterGetPayloadCacheControl(t *testing.T) {
	env := &engine.ExecutionPayloadEnvelope{
		ExecutionPayload: &engine.ExecutableData{LogsBloom: make([]byte, 256)},
		BlockValue:       uint256.NewInt(0).ToBig(),
	}
	srv := newTestServer(t, &stubBackend{envelope: env})
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + BasePath + "/amsterdam/payloads/0x0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("want 200, got %d", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control=%q want no-store", cc)
	}
}

func TestRouterGetPayloadUnknown(t *testing.T) {
	srv := newTestServer(t, &stubBackend{getErr: engine.UnknownPayload})
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL + BasePath + "/amsterdam/payloads/0x0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("want 404, got %d", resp.StatusCode)
	}
}
