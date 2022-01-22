package ethtest

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"math/rand"
)

func (s *Suite) TestSnapStatus(t *utesting.T) {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err := conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
}

type accRangeTest struct {
	nBytes uint64
	root   common.Hash
	origin common.Hash
	limit  common.Hash

	expAccounts int
	expFirst    common.Hash
	expLast     common.Hash
}

// TestSnapGetAccountRange various forms of GetAccountRange requests.
func (s *Suite) TestSnapGetAccountRange(t *utesting.T) {
	root := s.chain.RootAt(999)
	ffHash := common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	for i, tc := range []accRangeTest{
		// Tests decreasing the number of bytes
		{4000, root, common.Hash{}, ffHash, 76, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0xd2669dcf3858e7f1eecb8b5fedbf22fbea3e9433848a75035f79d68422c2dcda")},
		{3000, root, common.Hash{}, ffHash, 57, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x9b63fa753ece5cb90657d02ecb15df4dc1508d8c1d187af1bf7f1a05e747d3c7")},
		{2000, root, common.Hash{}, ffHash, 38, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x5e6140ecae4354a9e8f47559a8c6209c1e0e69cb077b067b528556c11698b91f")},
		{1, root, common.Hash{}, ffHash, 1, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a")},

		// Tests variations of the range
		{4000, root, common.HexToHash("0x00bf000000000000000000000000000000000000000000000000000000000000"), common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2b"), 2, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x09e47cd5056a689e708f22fe1f932709a320518e444f5f7d8d46a3da523d6606")},
		{4000, root, common.HexToHash("0x00b0000000000000000000000000000000000000000000000000000000000000"), common.HexToHash("0x00bf100000000000000000000000000000000000000000000000000000000000"), 1, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a")},
		{4000, root, common.HexToHash("0x00"), common.HexToHash("0x00"), 1, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a")},
		{4000, root, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), ffHash, 76, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2a"), common.HexToHash("0xd2669dcf3858e7f1eecb8b5fedbf22fbea3e9433848a75035f79d68422c2dcda")},
		{4000, root, common.HexToHash("0x00bf49f440a1cd0527e4d06e2765654c0f56452257516d793a9b8d604dcfdf2b"), ffHash, 76, common.HexToHash("0x09e47cd5056a689e708f22fe1f932709a320518e444f5f7d8d46a3da523d6606"), common.HexToHash("0xd28f55d3b994f16389f36944ad685b48e0fc3f8fbe86c3ca92ebecadf16a783f")},
	} {
		if err := s.snapGetAccountRange(t, &tc); err != nil {
			t.Errorf("test %d \n root: %x\n range: %#x - %#x\n bytes: %d\nfailed: %v", i, tc.root, tc.origin, tc.limit, tc.nBytes, err)
		}
	}
}

func (s *Suite) snapGetAccountRange(t *utesting.T, tc *accRangeTest) error {
	conn, err := s.dialSnap()
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()
	if err = conn.peer(s.chain, nil); err != nil {
		t.Fatalf("peering failed: %v", err)
	}
	// write request
	req := &GetAccountRange{
		ID:     uint64(rand.Int63()),
		Root:   tc.root,
		Origin: tc.origin,
		Limit:  tc.limit,
		Bytes:  tc.nBytes,
	}
	resp, err := conn.snapRequest(req, req.ID, s.chain)
	if err != nil {
		return fmt.Errorf("account range request failed: %v", err)
	}
	accRange, ok := resp.(*AccountRange)
	if !ok {
		return fmt.Errorf("account range response wrong: %T %v", resp, resp)
	}
	if exp, got := tc.expAccounts, len(accRange.Accounts); exp != got {
		return fmt.Errorf("expected %d accounts, got %d", exp, got)
	}
	if exp, got := tc.expFirst, accRange.Accounts[0].Hash; exp != got {
		return fmt.Errorf("expected first account 0x%x, got 0x%x", exp, got)
	}
	if exp, got := tc.expLast, accRange.Accounts[len(accRange.Accounts)-1].Hash; exp != got {
		return fmt.Errorf("expected last account 0x%x, got 0x%x", exp, got)
	}
	return nil
}
