// Copyright 2017 The go-ethereum Authors
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
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

//matches hex swarm hashes
// TODO: this is bad, it should not be hardcoded how long is a hash
var hashMatcher = regexp.MustCompile("^([0-9A-Fa-f]{64})([0-9A-Fa-f]{64})?$")

// URI is a reference to content stored in swarm.
type URI struct {
	// Scheme has one of the following values:
	//
	// * bzz           - an entry in a swarm manifest
	// * bzz-raw       - raw swarm content
	// * bzz-immutable - immutable URI of an entry in a swarm manifest
	//                   (address is not resolved)
	// * bzz-list      -  list of all files contained in a swarm manifest
	//
	Scheme string

	// Addr is either a hexadecimal storage address or it an address which
	// resolves to a storage address
	Addr string

	// addr stores the parsed storage address
	addr storage.Address

	// Path is the path to the content within a swarm manifest
	Path string
}

func (u *URI) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + u.String() + `"`), nil
}

func (u *URI) UnmarshalJSON(value []byte) error {
	uri, err := Parse(string(value))
	if err != nil {
		return err
	}
	*u = *uri
	return nil
}

// Parse parses rawuri into a URI struct, where rawuri is expected to have one
// of the following formats:
//
// * <scheme>:/
// * <scheme>:/<addr>
// * <scheme>:/<addr>/<path>
// * <scheme>://
// * <scheme>://<addr>
// * <scheme>://<addr>/<path>
//
// with scheme one of bzz, bzz-raw, bzz-immutable, bzz-list or bzz-hash
func Parse(rawuri string) (*URI, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, err
	}
	uri := &URI{Scheme: u.Scheme}

	// check the scheme is valid
	switch uri.Scheme {
	case "bzz", "bzz-raw", "bzz-immutable", "bzz-list", "bzz-hash", "bzz-resource":
	default:
		return nil, fmt.Errorf("unknown scheme %q", u.Scheme)
	}

	// handle URIs like bzz://<addr>/<path> where the addr and path
	// have already been split by url.Parse
	if u.Host != "" {
		uri.Addr = u.Host
		uri.Path = strings.TrimLeft(u.Path, "/")
		return uri, nil
	}

	// URI is like bzz:/<addr>/<path> so split the addr and path from
	// the raw path (which will be /<addr>/<path>)
	parts := strings.SplitN(strings.TrimLeft(u.Path, "/"), "/", 2)
	uri.Addr = parts[0]
	if len(parts) == 2 {
		uri.Path = parts[1]
	}
	return uri, nil
}
func (u *URI) Resource() bool {
	return u.Scheme == "bzz-resource"
}

func (u *URI) Raw() bool {
	return u.Scheme == "bzz-raw"
}

func (u *URI) Immutable() bool {
	return u.Scheme == "bzz-immutable"
}

func (u *URI) List() bool {
	return u.Scheme == "bzz-list"
}

func (u *URI) Hash() bool {
	return u.Scheme == "bzz-hash"
}

func (u *URI) String() string {
	return u.Scheme + ":/" + u.Addr + "/" + u.Path
}

func (u *URI) Address() storage.Address {
	if u.addr != nil {
		return u.addr
	}
	if hashMatcher.MatchString(u.Addr) {
		u.addr = common.Hex2Bytes(u.Addr)
		return u.addr
	}
	return nil
}
