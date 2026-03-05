// Copyright 2025 The go-ethereum Authors
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

package catalyst

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/node"
	"github.com/golang-jwt/jwt/v4"
)

func makeJWTToken(secret []byte) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(),
	})
	ss, _ := token.SignedString(secret)
	return ss
}

type testResponseWriter struct {
	code    int
	headers http.Header
	body    bytes.Buffer
}

func (w *testResponseWriter) Header() http.Header         { return w.headers }
func (w *testResponseWriter) Write(b []byte) (int, error) { return w.body.Write(b) }
func (w *testResponseWriter) WriteHeader(code int)        { w.code = code }

// TestSszRestJWTAuth tests that JWT auth is enforced.
func TestSszRestJWTAuth(t *testing.T) {
	secret := make([]byte, 32)
	rand.Read(secret)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := node.NewJWTHandler(secret, mux)

	// Test with valid token
	token := makeJWTToken(secret)
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := &testResponseWriter{headers: make(http.Header)}
	handler.ServeHTTP(rr, req)
	if rr.code != http.StatusOK {
		t.Errorf("expected 200 with valid JWT, got %d", rr.code)
	}

	// Test without token
	req2, _ := http.NewRequest("POST", "/test", nil)
	rr2 := &testResponseWriter{headers: make(http.Header)}
	handler.ServeHTTP(rr2, req2)
	if rr2.code != http.StatusUnauthorized {
		t.Errorf("expected 401 without JWT, got %d", rr2.code)
	}
}

// TestSszRestCapabilities tests the exchange_capabilities SSZ encoding.
func TestSszRestCapabilities(t *testing.T) {
	caps := []string{"engine_newPayloadV4", "engine_forkchoiceUpdatedV3"}
	encoded := engine.EncodeCapabilitiesSSZ(caps)
	decoded, err := engine.DecodeCapabilitiesSSZ(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 caps, got %d", len(decoded))
	}
	if decoded[0] != "engine_newPayloadV4" {
		t.Errorf("unexpected cap: %s", decoded[0])
	}
}

// TestSszRestErrorFormat tests that error responses are JSON per EIP-8161.
func TestSszRestErrorFormat(t *testing.T) {
	rr := &testResponseWriter{headers: make(http.Header), code: 200}
	sszErrorResponse(rr, http.StatusBadRequest, -32602, "test error")

	if rr.code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.code)
	}
	if rr.headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected application/json, got %s", rr.headers.Get("Content-Type"))
	}

	var errResp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	if errResp.Code != -32602 {
		t.Errorf("expected -32602, got %d", errResp.Code)
	}
	if errResp.Message != "test error" {
		t.Errorf("expected 'test error', got '%s'", errResp.Message)
	}
}

// TestSszRestSuccessFormat tests that success responses are SSZ per EIP-8161.
func TestSszRestSuccessFormat(t *testing.T) {
	rr := &testResponseWriter{headers: make(http.Header), code: 200}
	data := []byte{0x01, 0x02, 0x03}
	sszResponse(rr, data)

	if rr.code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.code)
	}
	if rr.headers.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("expected application/octet-stream, got %s", rr.headers.Get("Content-Type"))
	}
	if !bytes.Equal(rr.body.Bytes(), data) {
		t.Errorf("body mismatch")
	}
}
