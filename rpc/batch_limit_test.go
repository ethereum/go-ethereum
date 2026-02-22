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

package rpc

import (
	"strings"
	"sync/atomic"
	"testing"
)

const bigDataErrorCode = 555

type bigDataError struct {
	data string
}

func (e bigDataError) Error() string          { return "big data error" }
func (e bigDataError) ErrorCode() int         { return bigDataErrorCode }
func (e bigDataError) ErrorData() interface{} { return e.data }

type bigDataErrorService struct {
	calls int32
	data  string
}

func (s *bigDataErrorService) Fail() error {
	atomic.AddInt32(&s.calls, 1)
	return bigDataError{data: s.data}
}

func TestBatchResponseSizeLimitCountsErrorResponses(t *testing.T) {
	t.Parallel()

	server := newTestServer()
	defer server.Stop()
	server.SetBatchLimits(100, 50)

	svc := &bigDataErrorService{data: strings.Repeat("a", 200)}
	if err := server.RegisterName("big", svc); err != nil {
		t.Fatalf("RegisterName: %v", err)
	}

	client := DialInProc(server)
	defer client.Close()

	batch := make([]BatchElem, 5)
	for i := range batch {
		batch[i] = BatchElem{Method: "big_fail"}
	}
	if err := client.BatchCall(batch); err != nil {
		t.Fatalf("BatchCall: %v", err)
	}

	if got := atomic.LoadInt32(&svc.calls); got != 1 {
		t.Fatalf("expected 1 processed call, got %d", got)
	}

	// The first item was processed and returned the original error.
	if batch[0].Error == nil {
		t.Fatal("batch elem 0 missing error")
	}
	if re, ok := batch[0].Error.(Error); !ok {
		t.Fatalf("batch elem 0 wrong error type: %T", batch[0].Error)
	} else if re.ErrorCode() != bigDataErrorCode {
		t.Fatalf("batch elem 0 wrong error code: have %d want %d", re.ErrorCode(), bigDataErrorCode)
	}
	if de, ok := batch[0].Error.(DataError); !ok {
		t.Fatalf("batch elem 0 missing error data: %T", batch[0].Error)
	} else if data, ok := de.ErrorData().(string); !ok || len(data) != len(svc.data) {
		t.Fatalf("batch elem 0 wrong error data size: have %v want %d", de.ErrorData(), len(svc.data))
	}

	// Remaining items should return "response too large" without being processed.
	for i := 1; i < len(batch); i++ {
		if batch[i].Error == nil {
			t.Fatalf("batch elem %d missing error", i)
		}
		re, ok := batch[i].Error.(Error)
		if !ok {
			t.Fatalf("batch elem %d wrong error type: %T", i, batch[i].Error)
		}
		if re.ErrorCode() != errcodeResponseTooLarge {
			t.Fatalf("batch elem %d wrong error code: have %d want %d", i, re.ErrorCode(), errcodeResponseTooLarge)
		}
	}
}
