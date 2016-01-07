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
Package rpc provides access to the exported methods of an object across a network
or other I/O connection. After creating a server instance objects can be registered,
making it visible from the outside. Exported methods that follow specific
conventions can be called remotely. It also has support for the publish/subscribe
pattern.

Methods that satisfy the following criteria are made available for remote access:
 - object must be exported
 - method must be exported
 - method returns 0, 1 (response or error) or 2 (response and error) values
 - method argument(s) must be exported or builtin types
 - method returned value(s) must be exported or builtin types

An example method:
 func (s *CalcService) Div(a, b int) (int, error)

When the returned error isn't nil the returned integer is ignored and the error is
send back to the client. Otherwise the returned integer is send back to the client.

The server offers the ServeCodec method which accepts a ServerCodec instance. It will
read requests from the codec, process the request and sends the response back to the
client using the codec. The server can execute requests concurrently. Responses
can be send back to the client out of order.

An example server which uses the JSON codec:
 type CalculatorService struct {}

 func (s *CalculatorService) Add(a, b int) int {
	return a + b
 }

 func (s *CalculatorService Div(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a/b, nil
 }

 calculator := new(CalculatorService)
 server := NewServer()
 server.RegisterName("calculator", calculator")

 l, _ := net.ListenUnix("unix", &net.UnixAddr{Net: "unix", Name: "/tmp/calculator.sock"})
 for {
	c, _ := l.AcceptUnix()
	codec := v2.NewJSONCodec(c)
	go server.ServeCodec(codec)
 }

The package also supports the publish subscribe pattern through the use of subscriptions.
A method that is considered eligible for notifications must satisfy the following criteria:
 - object must be exported
 - method must be exported
 - method argument(s) must be exported or builtin types
 - method must return the tuple Subscription, error


An example method:
 func (s *BlockChainService) Head() (Subscription, error) {
 	sub := s.bc.eventMux.Subscribe(ChainHeadEvent{})
	return v2.NewSubscription(sub), nil
 }

This method will push all raised ChainHeadEvents to subscribed clients. If the client is only
interested in every N'th block it is possible to add a criteria.

 func (s *BlockChainService) HeadFiltered(nth uint64) (Subscription, error) {
 	sub := s.bc.eventMux.Subscribe(ChainHeadEvent{})

	criteria := func(event interface{}) bool {
		chainHeadEvent := event.(ChainHeadEvent)
		if chainHeadEvent.Block.NumberU64() % nth == 0 {
			return true
		}
		return false
	}

	return v2.NewSubscriptionFiltered(sub, criteria), nil
 }

Subscriptions are deleted when:
 - the user sends an unsubscribe request
 - the connection which was used to create the subscription is closed
*/
package v2
