// Copyright 2018 The go-ethereum Authors
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
package stream

import (
	"testing"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

// This test checks the default behavior of the server, that is
// when it is serving Retrieve requests.
func TestLigthnodeRetrieveRequestWithRetrieve(t *testing.T) {
	registryOptions := &RegistryOptions{
		Retrieval: RetrievalClientOnly,
		Syncing:   SyncingDisabled,
	}
	tester, _, _, teardown, err := newStreamerTester(registryOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	stream := NewStream(swarmChunkServerStreamName, "", false)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "SubscribeMsg",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream: stream,
				},
				Peer: node.ID(),
			},
		},
	})
	if err != nil {
		t.Fatalf("Got %v", err)
	}

	err = tester.TestDisconnected(&p2ptest.Disconnect{Peer: node.ID()})
	if err == nil || err.Error() != "timed out waiting for peers to disconnect" {
		t.Fatalf("Expected no disconnect, got %v", err)
	}
}

// This test checks the Lightnode behavior of server, when serving Retrieve
// requests are disabled
func TestLigthnodeRetrieveRequestWithoutRetrieve(t *testing.T) {
	registryOptions := &RegistryOptions{
		Retrieval: RetrievalDisabled,
		Syncing:   SyncingDisabled,
	}
	tester, _, _, teardown, err := newStreamerTester(registryOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	stream := NewStream(swarmChunkServerStreamName, "", false)

	err = tester.TestExchanges(
		p2ptest.Exchange{
			Label: "SubscribeMsg",
			Triggers: []p2ptest.Trigger{
				{
					Code: 4,
					Msg: &SubscribeMsg{
						Stream: stream,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 7,
					Msg: &SubscribeErrorMsg{
						Error: "stream RETRIEVE_REQUEST not registered",
					},
					Peer: node.ID(),
				},
			},
		})
	if err != nil {
		t.Fatalf("Got %v", err)
	}
}

// This test checks the default behavior of the server, that is
// when syncing is enabled.
func TestLigthnodeRequestSubscriptionWithSync(t *testing.T) {
	registryOptions := &RegistryOptions{
		Retrieval: RetrievalDisabled,
		Syncing:   SyncingRegisterOnly,
	}
	tester, _, _, teardown, err := newStreamerTester(registryOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	syncStream := NewStream("SYNC", FormatSyncBinKey(1), false)

	err = tester.TestExchanges(
		p2ptest.Exchange{
			Label: "RequestSubscription",
			Triggers: []p2ptest.Trigger{
				{
					Code: 8,
					Msg: &RequestSubscriptionMsg{
						Stream: syncStream,
					},
					Peer: node.ID(),
				},
			},
			Expects: []p2ptest.Expect{
				{
					Code: 4,
					Msg: &SubscribeMsg{
						Stream: syncStream,
					},
					Peer: node.ID(),
				},
			},
		})

	if err != nil {
		t.Fatalf("Got %v", err)
	}
}

// This test checks the Lightnode behavior of the server, that is
// when syncing is disabled.
func TestLigthnodeRequestSubscriptionWithoutSync(t *testing.T) {
	registryOptions := &RegistryOptions{
		Retrieval: RetrievalDisabled,
		Syncing:   SyncingDisabled,
	}
	tester, _, _, teardown, err := newStreamerTester(registryOptions)
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	node := tester.Nodes[0]

	syncStream := NewStream("SYNC", FormatSyncBinKey(1), false)

	err = tester.TestExchanges(p2ptest.Exchange{
		Label: "RequestSubscription",
		Triggers: []p2ptest.Trigger{
			{
				Code: 8,
				Msg: &RequestSubscriptionMsg{
					Stream: syncStream,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 7,
				Msg: &SubscribeErrorMsg{
					Error: "stream SYNC not registered",
				},
				Peer: node.ID(),
			},
		},
	}, p2ptest.Exchange{
		Label: "RequestSubscription",
		Triggers: []p2ptest.Trigger{
			{
				Code: 4,
				Msg: &SubscribeMsg{
					Stream: syncStream,
				},
				Peer: node.ID(),
			},
		},
		Expects: []p2ptest.Expect{
			{
				Code: 7,
				Msg: &SubscribeErrorMsg{
					Error: "stream SYNC not registered",
				},
				Peer: node.ID(),
			},
		},
	})

	if err != nil {
		t.Fatalf("Got %v", err)
	}
}
