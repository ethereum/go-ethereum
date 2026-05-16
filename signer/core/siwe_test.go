// Copyright 2024 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// siwePositiveCase mirrors the structure of siwe_parsing_positive.json.
type siwePositiveCase struct {
	Message string             `json:"message"`
	Fields  siweExpectedFields `json:"fields"`
}

// siweExpectedFields holds the expected parsed values from the test fixture.
// Time fields are kept as strings because the fixture preserves the raw wire
// format; we parse them via parseSIWEDateTime for comparison.
type siweExpectedFields struct {
	Scheme         *string  `json:"scheme"`
	Domain         string   `json:"domain"`
	Address        string   `json:"address"`
	Statement      string   `json:"statement"`
	URI            string   `json:"uri"`
	Version        string   `json:"version"`
	ChainID        uint64   `json:"chainId"`
	Nonce          string   `json:"nonce"`
	IssuedAt       string   `json:"issuedAt"`
	ExpirationTime *string  `json:"expirationTime"`
	NotBefore      *string  `json:"notBefore"`
	RequestID      *string  `json:"requestId"`
	Resources      []string `json:"resources"`
}

// minimalSIWE is a valid EIP-4361 message used across domain-check tests.
const minimalSIWE = "example.com wants you to sign in with your Ethereum account:\n" +
	"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2\n" +
	"\n" +
	"\n" +
	"URI: https://example.com/login\n" +
	"Version: 1\n" +
	"Chain ID: 1\n" +
	"Nonce: 32891757\n" +
	"Issued At: 2021-09-30T16:25:24Z"

func TestParseSIWEMessage_Positive(t *testing.T) {
	data, err := os.ReadFile("testdata/siwe/parsing_positive.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases map[string]siwePositiveCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseSIWEMessage(tc.Message)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkSIWEFields(t, got, tc.Fields)
		})
	}
}

func TestParseSIWEMessage_Negative(t *testing.T) {
	data, err := os.ReadFile("testdata/siwe/parsing_negative.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases map[string]string
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}

	for name, message := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parseSIWEMessage(message)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestValidateSIWEMessage(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		meta         Metadata
		wantMessages bool
		wantCRIT     bool
		wantWARN     bool
	}{
		{
			name:         "non-SIWE text returns nil",
			text:         "hello world",
			meta:         Metadata{Scheme: "http"},
			wantMessages: false,
		},
		{
			name:         "valid SIWE over IPC skips domain check",
			text:         minimalSIWE,
			meta:         Metadata{Scheme: "ipc"},
			wantMessages: true,
		},
		{
			name:         "valid SIWE over HTTP with matching origin",
			text:         minimalSIWE,
			meta:         Metadata{Scheme: "http", Origin: "https://example.com"},
			wantMessages: true,
		},
		{
			name:         "valid SIWE over HTTP with mismatched origin",
			text:         minimalSIWE,
			meta:         Metadata{Scheme: "http", Origin: "https://evil.com"},
			wantMessages: true,
			wantCRIT:     true,
		},
		{
			name:         "valid SIWE over HTTP with no origin header",
			text:         minimalSIWE,
			meta:         Metadata{Scheme: "http", Origin: ""},
			wantMessages: true,
			wantWARN:     true,
		},
		{
			name:         "malformed SIWE returns nil messages and a WARN",
			text:         "example.com wants you to sign in with your Ethereum account:\nnot-an-address",
			meta:         Metadata{Scheme: "http"},
			wantMessages: false,
			wantWARN:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, callInfo := validateSIWEMessage(tt.text, tt.meta)
			if tt.wantMessages && messages == nil {
				t.Error("expected structured messages, got nil")
			}
			if !tt.wantMessages && messages != nil {
				t.Errorf("expected nil messages, got %d entries", len(messages))
			}
			checkSIWECallInfo(t, callInfo, tt.wantCRIT, tt.wantWARN)
		})
	}
}

func checkSIWECallInfo(t *testing.T, callInfo []apitypes.ValidationInfo, wantCRIT, wantWARN bool) {
	t.Helper()
	var hasCRIT, hasWARN bool
	for _, info := range callInfo {
		hasCRIT = hasCRIT || info.Typ == apitypes.CRIT
		hasWARN = hasWARN || info.Typ == apitypes.WARN
	}
	if wantCRIT != hasCRIT {
		t.Errorf("CRIT callInfo: want %v, got entries %v", wantCRIT, callInfo)
	}
	if wantWARN != hasWARN {
		t.Errorf("WARN callInfo: want %v, got entries %v", wantWARN, callInfo)
	}
}

func checkSIWEFields(t *testing.T, got *SIWEMessage, want siweExpectedFields) {
	t.Helper()

	wantScheme := ""
	if want.Scheme != nil {
		wantScheme = *want.Scheme
	}
	if got.Scheme != wantScheme {
		t.Errorf("Scheme: got %q, want %q", got.Scheme, wantScheme)
	}
	if got.Domain != want.Domain {
		t.Errorf("Domain: got %q, want %q", got.Domain, want.Domain)
	}
	if got.Address != want.Address {
		t.Errorf("Address: got %q, want %q", got.Address, want.Address)
	}
	if got.Statement != want.Statement {
		t.Errorf("Statement: got %q, want %q", got.Statement, want.Statement)
	}
	if got.URI != want.URI {
		t.Errorf("URI: got %q, want %q", got.URI, want.URI)
	}
	if got.Version != want.Version {
		t.Errorf("Version: got %q, want %q", got.Version, want.Version)
	}
	if got.ChainID != want.ChainID {
		t.Errorf("ChainID: got %d, want %d", got.ChainID, want.ChainID)
	}
	if got.Nonce != want.Nonce {
		t.Errorf("Nonce: got %q, want %q", got.Nonce, want.Nonce)
	}

	wantIssuedAt, err := parseSIWEDateTime(want.IssuedAt)
	if err != nil {
		t.Fatalf("test data has invalid IssuedAt %q: %v", want.IssuedAt, err)
	}
	if !got.IssuedAt.Equal(wantIssuedAt) {
		t.Errorf("IssuedAt: got %v, want %v", got.IssuedAt, wantIssuedAt)
	}

	checkSIWEOptionalTime(t, "ExpirationTime", got.ExpirationTime, want.ExpirationTime)
	checkSIWEOptionalTime(t, "NotBefore", got.NotBefore, want.NotBefore)

	wantRequestID := ""
	if want.RequestID != nil {
		wantRequestID = *want.RequestID
	}
	if got.RequestID != wantRequestID {
		t.Errorf("RequestID: got %q, want %q", got.RequestID, wantRequestID)
	}

	checkSIWEResources(t, got.Resources, want.Resources)
}

func checkSIWEResources(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("Resources: got %d items, want %d", len(got), len(want))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Resources[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func checkSIWEOptionalTime(t *testing.T, field string, got *time.Time, wantStr *string) {
	t.Helper()
	if wantStr == nil {
		if got != nil {
			t.Errorf("%s: got %v, want nil", field, *got)
		}
		return
	}
	if got == nil {
		t.Errorf("%s: got nil, want %q", field, *wantStr)
		return
	}
	want, err := parseSIWEDateTime(*wantStr)
	if err != nil {
		t.Fatalf("test data has invalid %s %q: %v", field, *wantStr, err)
	}
	if !got.Equal(want) {
		t.Errorf("%s: got %v, want %v", field, *got, want)
	}
}
