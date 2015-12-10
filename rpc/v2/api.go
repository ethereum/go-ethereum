// Copyright 2015 The go-ethereum Authors
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

package v2

// Api describes a versioned API schema offered over the RPC interface.
type Api struct {
	Version  int          // Version number under which to advertise this method set
	Handlers []ApiHandler // List of API handlers advertized on this API version
}

// ApiHandler describes the set of methods offered over the RPC interface
type ApiHandler struct {
	Path    string      // Path under which the RPC methods of Handler are to be exposed
	Handler interface{} // Receiver instance which holds the methods to expose
	Public  bool        // Indication if the methods can be considered safe for public use
}
