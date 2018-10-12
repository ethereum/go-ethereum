package stream

import (
	"testing"

	p2ptest "github.com/ethereum/go-ethereum/p2p/testing"
)

func TestLigthnodeRetrieveRequestWithRetrieve(t *testing.T) {
	registryOptions := &RegistryOptions{
		DoServeRetrieve: true,
	}
	tester, _, _, teardown, err := newStreamerTester(t, registryOptions)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

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

func TestLigthnodeRetrieveRequestWithoutRetrieve(t *testing.T) {
	registryOptions := &RegistryOptions{
		DoServeRetrieve: false,
	}
	tester, _, _, teardown, err := newStreamerTester(t, registryOptions)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

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

func TestLigthnodeRequestSubscriptionWithSync(t *testing.T) {
	registryOptions := &RegistryOptions{
		DoSync: true,
	}
	tester, _, _, teardown, err := newStreamerTester(t, registryOptions)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

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

func TestLigthnodeRequestSubscriptionWithoutSync(t *testing.T) {
	registryOptions := &RegistryOptions{
		DoSync: false,
	}
	tester, _, _, teardown, err := newStreamerTester(t, registryOptions)
	defer teardown()
	if err != nil {
		t.Fatal(err)
	}

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
