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
	"fmt"
	"net/url"
	"strings"
)

// URI is a reference to content stored in swarm.
type URI struct {
	// Scheme has one of the following values:
	//
	// * bzz  - an entry in a swarm manifest
	// * bzzr - raw swarm content
	// * bzzi - immutable URI of an entry in a swarm manifest
	//          (address is not resolved)
	Scheme string

	// Addr is either a hexadecimal storage key or it an address which
	// resolves to a storage key
	Addr string

	// Path is the path to the content within a swarm manifest
	Path string
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
// with scheme one of bzz, bzzr or bzzi
func Parse(rawuri string) (*URI, error) {
	u, err := url.Parse(rawuri)
	if err != nil {
		return nil, err
	}
	uri := &URI{Scheme: u.Scheme}

	// check the scheme is valid
	switch uri.Scheme {
	case "bzz", "bzzi", "bzzr":
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

func (u *URI) Raw() bool {
	return u.Scheme == "bzzr"
}

func (u *URI) Immutable() bool {
	return u.Scheme == "bzzi"
}

func (u *URI) String() string {
	return u.Scheme + ":/" + u.Addr + "/" + u.Path
}
