// Copyright 2016 The go-ethereum Authors
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

package api

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func testApi(t *testing.T, f func(*Api)) {
	datadir, err := ioutil.TempDir("", "bzz-test")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	os.RemoveAll(datadir)
	defer os.RemoveAll(datadir)
	dpa, err := storage.NewLocalDPA(datadir)
	if err != nil {
		return
	}
	api := NewApi(dpa, nil)
	dpa.Start()
	f(api)
	dpa.Stop()
}

type testResponse struct {
	reader storage.LazySectionReader
	*Response
}

func checkResponse(t *testing.T, resp *testResponse, exp *Response) {

	if resp.MimeType != exp.MimeType {
		t.Errorf("incorrect mimeType. expected '%s', got '%s'", exp.MimeType, resp.MimeType)
	}
	if resp.Status != exp.Status {
		t.Errorf("incorrect status. expected '%d', got '%d'", exp.Status, resp.Status)
	}
	if resp.Size != exp.Size {
		t.Errorf("incorrect size. expected '%d', got '%d'", exp.Size, resp.Size)
	}
	if resp.reader != nil {
		content := make([]byte, resp.Size)
		read, _ := resp.reader.Read(content)
		if int64(read) != exp.Size {
			t.Errorf("incorrect content length. expected '%d...', got '%d...'", read, exp.Size)
		}
		resp.Content = string(content)
	}
	if resp.Content != exp.Content {
		// if !bytes.Equal(resp.Content, exp.Content)
		t.Errorf("incorrect content. expected '%s...', got '%s...'", string(exp.Content), string(resp.Content))
	}
}

// func expResponse(content []byte, mimeType string, status int) *Response {
func expResponse(content string, mimeType string, status int) *Response {
	log.Trace(fmt.Sprintf("expected content (%v): %v ", len(content), content))
	return &Response{mimeType, status, int64(len(content)), content}
}

// func testGet(t *testing.T, api *Api, bzzhash string) *testResponse {
func testGet(t *testing.T, api *Api, bzzhash, path string) *testResponse {
	key := storage.Key(common.Hex2Bytes(bzzhash))
	reader, mimeType, status, err := api.Get(key, path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	quitC := make(chan bool)
	size, err := reader.Size(quitC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	log.Trace(fmt.Sprintf("reader size: %v ", size))
	s := make([]byte, size)
	_, err = reader.Read(s)
	if err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
	reader.Seek(0, 0)
	return &testResponse{reader, &Response{mimeType, status, size, string(s)}}
	// return &testResponse{reader, &Response{mimeType, status, reader.Size(), nil}}
}

func TestApiPut(t *testing.T) {
	testApi(t, func(api *Api) {
		content := "hello"
		exp := expResponse(content, "text/plain", 0)
		// exp := expResponse([]byte(content), "text/plain", 0)
		key, err := api.Put(content, exp.MimeType)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resp := testGet(t, api, key.String(), "")
		checkResponse(t, resp, exp)
	})
}

// testResolver implements the Resolver interface and either returns the given
// hash if it is set, or returns a "name not found" error
type testResolver struct {
	hash *common.Hash
}

func newTestResolver(addr string) *testResolver {
	r := &testResolver{}
	if addr != "" {
		hash := common.HexToHash(addr)
		r.hash = &hash
	}
	return r
}

func (t *testResolver) Resolve(addr string) (common.Hash, error) {
	if t.hash == nil {
		return common.Hash{}, fmt.Errorf("DNS name not found: %q", addr)
	}
	return *t.hash, nil
}

// TestAPIResolve tests resolving URIs which can either contain content hashes
// or ENS names
func TestAPIResolve(t *testing.T) {
	ensAddr := "swarm.eth"
	hashAddr := "1111111111111111111111111111111111111111111111111111111111111111"
	resolvedAddr := "2222222222222222222222222222222222222222222222222222222222222222"
	doesResolve := newTestResolver(resolvedAddr)
	doesntResolve := newTestResolver("")

	type test struct {
		desc      string
		dns       Resolver
		addr      string
		immutable bool
		result    string
		expectErr error
	}

	tests := []*test{
		{
			desc:   "DNS not configured, hash address, returns hash address",
			dns:    nil,
			addr:   hashAddr,
			result: hashAddr,
		},
		{
			desc:      "DNS not configured, ENS address, returns error",
			dns:       nil,
			addr:      ensAddr,
			expectErr: errors.New(`no DNS to resolve name: "swarm.eth"`),
		},
		{
			desc:   "DNS configured, hash address, hash resolves, returns resolved address",
			dns:    doesResolve,
			addr:   hashAddr,
			result: resolvedAddr,
		},
		{
			desc:      "DNS configured, immutable hash address, hash resolves, returns hash address",
			dns:       doesResolve,
			addr:      hashAddr,
			immutable: true,
			result:    hashAddr,
		},
		{
			desc:   "DNS configured, hash address, hash doesn't resolve, returns hash address",
			dns:    doesntResolve,
			addr:   hashAddr,
			result: hashAddr,
		},
		{
			desc:   "DNS configured, ENS address, name resolves, returns resolved address",
			dns:    doesResolve,
			addr:   ensAddr,
			result: resolvedAddr,
		},
		{
			desc:      "DNS configured, immutable ENS address, name resolves, returns error",
			dns:       doesResolve,
			addr:      ensAddr,
			immutable: true,
			expectErr: errors.New(`immutable address not a content hash: "swarm.eth"`),
		},
		{
			desc:      "DNS configured, ENS address, name doesn't resolve, returns error",
			dns:       doesntResolve,
			addr:      ensAddr,
			expectErr: errors.New(`DNS name not found: "swarm.eth"`),
		},
	}
	for _, x := range tests {
		t.Run(x.desc, func(t *testing.T) {
			api := &Api{dns: x.dns}
			uri := &URI{Addr: x.addr, Scheme: "bzz"}
			if x.immutable {
				uri.Scheme = "bzzi"
			}
			res, err := api.Resolve(uri)
			if err == nil {
				if x.expectErr != nil {
					t.Fatalf("expected error %q, got result %q", x.expectErr, res)
				}
				if res.String() != x.result {
					t.Fatalf("expected result %q, got %q", x.result, res)
				}
			} else {
				if x.expectErr == nil {
					t.Fatalf("expected no error, got %q", err)
				}
				if err.Error() != x.expectErr.Error() {
					t.Fatalf("expected error %q, got %q", x.expectErr, err)
				}
			}
		})
	}
}

func TestMultiResolver(t *testing.T) {
	doesntResolve := newTestResolver("")

	ethAddr := "swarm.eth"
	ethHash := "0x2222222222222222222222222222222222222222222222222222222222222222"
	ethResolve := newTestResolver(ethHash)

	testAddr := "swarm.test"
	testHash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	testResolve := newTestResolver(testHash)

	tests := []struct {
		desc   string
		r      Resolver
		addr   string
		result string
		err    error
	}{
		{
			desc: "No resolvers, returns error",
			r:    NewMultiResolver(),
			err:  NewNoResolverError(""),
		},
		{
			desc:   "One default resolver, returns resolved address",
			r:      NewMultiResolver(MultiResolverOptionWithResolver(ethResolve, "")),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "Two default resolvers, returns resolved address",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(ethResolve, ""),
				MultiResolverOptionWithResolver(ethResolve, ""),
			),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "Two default resolvers, first doesn't resolve, returns resolved address",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(doesntResolve, ""),
				MultiResolverOptionWithResolver(ethResolve, ""),
			),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "Default resolver doesn't resolve, tld resolver resolve, returns resolved address",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(doesntResolve, ""),
				MultiResolverOptionWithResolver(ethResolve, "eth"),
			),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "Three TLD resolvers, third resolves, returns resolved address",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(doesntResolve, "eth"),
				MultiResolverOptionWithResolver(doesntResolve, "eth"),
				MultiResolverOptionWithResolver(ethResolve, "eth"),
			),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "One TLD resolver doesn't resolve, returns error",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(doesntResolve, ""),
				MultiResolverOptionWithResolver(ethResolve, "eth"),
			),
			addr:   ethAddr,
			result: ethHash,
		},
		{
			desc: "One defautl and one TLD resolver, all doesn't resolve, returns error",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(doesntResolve, ""),
				MultiResolverOptionWithResolver(doesntResolve, "eth"),
			),
			addr:   ethAddr,
			result: ethHash,
			err:    errors.New(`DNS name not found: "swarm.eth"`),
		},
		{
			desc: "Two TLD resolvers, both resolve, returns resolved address",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(ethResolve, "eth"),
				MultiResolverOptionWithResolver(testResolve, "test"),
			),
			addr:   testAddr,
			result: testHash,
		},
		{
			desc: "One TLD resolver, no default resolver, returns error for different TLD",
			r: NewMultiResolver(
				MultiResolverOptionWithResolver(ethResolve, "eth"),
			),
			addr: testAddr,
			err:  NewNoResolverError("test"),
		},
	}
	for _, x := range tests {
		t.Run(x.desc, func(t *testing.T) {
			res, err := x.r.Resolve(x.addr)
			if err == nil {
				if x.err != nil {
					t.Fatalf("expected error %q, got result %q", x.err, res.Hex())
				}
				if res.Hex() != x.result {
					t.Fatalf("expected result %q, got %q", x.result, res.Hex())
				}
			} else {
				if x.err == nil {
					t.Fatalf("expected no error, got %q", err)
				}
				if err.Error() != x.err.Error() {
					t.Fatalf("expected error %q, got %q", x.err, err)
				}
			}
		})
	}
}
