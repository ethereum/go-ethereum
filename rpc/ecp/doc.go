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

/*
Package ecp implements the Ethereum Client Protocol (ECP) and contains a server to
create a RPC server offering a public API using ECP.

The purpose of this package is to provide access to exported methods an I/O connection.
A client can instantiate a server and register objects. The server will expose the
object as a service. After registration exported methods can be called remotely.

Methods that satisfy the following criteria will be exposed:
	- the object is exported
	- the method is exported
	- arguments must be builtin types or *big.Int
	- when the method returns an error its must be the last returned values

The REdis Serialization Protocol (http://redis.io/topics/protocol) is used as
serialization format. It is a fast and easy to parse protocol which is binary safe.
Currently the following types are supported:

	- string, send as simple string
	- []byte, send as bulk/binary string
	- big.Int, send as bulk/binary string in big endian format
	- int64, send as integer
	- error, as the default error type starting with an error code following the error message
	- struct, fields are send in an array in the definition order
	- array, as an array

Nil values are send as a null array in case the type is an array, otherwise as a null
binary string.

See the official REdis Serialization Protocol specification page for a description and
examples of how data is serialized. The ecp server uses the same convention as the Redis
server for messages. Requests and responses are represented in an array. Services who return
no data return with the +OK\r\n message as an indication that the request was processed.

Example server:

	type ExampleService struct {
	}

	func (s *ExampleService) Echo(str string, i int) (string, int) {
		return str, i
	}

	func main() {
		srvr := ecp.NewServer()
		srvr.RegisterName("example", new(ExampleService))

		l, _ := net.ListenUnix("unix", &net.UnixAddr{Name: "/tmp/example.sock", Net: "unix"})
		go srvr.Serve(l)
		...
		srvr.Stop()
	}

A client can now send the following request:
	*3\r\n+example.Echo\r\n+some string\r\n:42\r\n
And receive the following response:
	*2\r\n+some string\r\n:42\r\n
*/
package ecp
