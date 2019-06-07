// Copyright 2019 The go-ethereum Authors
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
This work is adapted from work by Charles Crain at https://github.com/jpmorganchase/quorum/blob/9b7fd9af8082795eeeb6863d9746f12b82dd5078/statediff/statediff.go

Package statediff provides an auxiliary service that processes state diff objects from incoming chain events,
relaying the objects to any rpc subscriptions.

The service is spun up using the below CLI flags
--statediff: boolean flag, turns on the service
--statediff.streamblock: boolean flag, configures the service to associate and stream out the rest of the block data with the state diffs.
--statediff.intermediatenodes: boolean flag, tells service to include intermediate (branch and extension) nodes; default (false) processes leaf nodes only.
--statediff.pathsandproofs: boolean flag, tells service to generate paths and proofs for the diffed storage and state trie leaf nodes.
--statediff.watchedaddresses: string slice flag, used to limit the state diffing process to the given addresses. Usage: --statediff.watchedaddresses=addr1 --statediff.watchedaddresses=addr2 --statediff.watchedaddresses=addr3

If you wish to use the websocket endpoint to subscribe to the statediff service, be sure to open up the Websocket RPC server with the `--ws` flag.

Rpc subscriptions to the service can be created using the rpc.Client.Subscribe() method,
with the "statediff" namespace, a statediff.Payload channel, and the name of the statediff api's rpc method- "stream".

e.g.

cli, _ := rpc.Dial("ipcPathOrWsURL")
stateDiffPayloadChan := make(chan statediff.Payload, 20000)
rpcSub, err := cli.Subscribe(context.Background(), "statediff", stateDiffPayloadChan, "stream"})
for {
	select {
	case stateDiffPayload := <- stateDiffPayloadChan:
		processPayload(stateDiffPayload)
	case err := <= rpcSub.Err():
		log.Error(err)
	}
}
*/
package statediff
