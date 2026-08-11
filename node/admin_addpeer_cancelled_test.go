package node

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// TestAdminAddPeerReturnsTrueAfterSchedulerStop is a regression test for
// https://github.com/ethereum/go-ethereum/issues/35484: admin_addPeer (and
// admin_addTrustedPeer) reported success even after the dial scheduler / server
// had been stopped. In that state the request is silently dropped instead of
// being accepted for tracking, so the RPC must surface an error rather than a
// bogus "true".
//
// Stock Node.Close stops the RPC layer before P2P, so this fixture deliberately
// stops the p2p server while keeping the admin API object callable, isolating
// the API/scheduler contract.
func TestAdminAddPeerReturnsTrueAfterSchedulerStop(t *testing.T) {
	n, err := New(&Config{
		Name: "admin-addpeer-cancelled",
		P2P: p2p.Config{
			ListenAddr:  "127.0.0.1:0",
			NoDiscovery: true,
			MaxPeers:    1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer n.Close()
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}

	// Stop the p2p server while leaving the admin service callable. This cancels
	// the dial scheduler, so a subsequent AddPeer / AddTrustedPeer must not
	// report success: the node can no longer be accepted for tracking.
	n.server.Stop()

	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	peerURL := enode.NewV4(&key.PublicKey, nil, 30399, 0).URLv4()

	api := &adminAPI{node: n}
	if result, err := api.AddPeer(peerURL); err == nil && result {
		t.Fatalf("AddPeer reported success after the dial scheduler was stopped: result=%v err=%v", result, err)
	}
	if result, err := api.AddTrustedPeer(peerURL); err == nil && result {
		t.Fatalf("AddTrustedPeer reported success after the server was stopped: result=%v err=%v", result, err)
	}
}
