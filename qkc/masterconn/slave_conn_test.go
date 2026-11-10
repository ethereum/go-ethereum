package masterconn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/qkc/comm"
	"github.com/ethereum/go-ethereum/qkc/conn"
	"github.com/ethereum/go-ethereum/qkc/slaveconn"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/qkc/wire"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

type slaveHandlerFunc struct {
	addShardBlockHeader func(*big.Int, *types.Header, *uint256.Int) error
}

func (handler slaveHandlerFunc) AddShardBlockHeader(source *big.Int, header *types.Header, amount *uint256.Int) error {
	if handler.addShardBlockHeader == nil {
		return nil
	}
	return handler.addShardBlockHeader(source, header, amount)
}

func noopShutdown() {}

func testRoot(number uint32) *qkctypes.RootBlock {
	return qkctypes.NewRootBlockWithHeader(&qkctypes.RootBlockHeader{Number: number})
}

func testHeader() *types.Header {
	return &types.Header{Number: big.NewInt(7), Difficulty: big.NewInt(1)}
}

func headerRequest(t *testing.T) *wire.AddShardBlockHeaderRequest {
	t.Helper()
	encoded, err := rlp.EncodeToBytes(testHeader())
	if err != nil {
		t.Fatal(err)
	}
	return &wire.AddShardBlockHeaderRequest{ShardBlockHeader: encoded, CoinbaseAmount: [32]byte{9}}
}

func newSlaveTestPair(t *testing.T, chainID *big.Int, handler SlaveRequestHandler, peerHandlers map[byte]conn.TypedHandler) (*SlaveConn, *conn.BaseConn) {
	t.Helper()
	local, remote := net.Pipe()
	if handler == nil {
		handler = slaveHandlerFunc{}
	}
	sc, err := NewSlaveConn(SlaveConnConfig{Conn: local, RemoteChainID: chainID, Handler: handler, ShutdownMaster: noopShutdown})
	if err != nil {
		local.Close()
		remote.Close()
		t.Fatal(err)
	}
	peer := conn.NewBaseConn(conn.Config{
		Transport: conn.NewTCPTransport(remote, func(reader io.Reader) (*wire.ClusterFrame, error) {
			return wire.ReadClusterFrameWithPeerID(reader, wire.MaxFramePayloadLen)
		}, wire.WriteClusterFrameWithPeerID),
		Serializers: map[byte]*conn.OpSerializer{
			byte(wire.ClusterOpPing):                         conn.OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong)),
			byte(wire.ClusterOpConnectToSlavesRequest):       conn.OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](byte(wire.ClusterOpConnectToSlavesResponse)),
			byte(wire.ClusterOpAddRootBlockRequest):          conn.OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](byte(wire.ClusterOpAddRootBlockResponse)),
			byte(wire.ClusterOpGetRootChainStakesRequest):    conn.OpSerializerFor[wire.GetRootChainStakesRequest, wire.GetRootChainStakesResponse](byte(wire.ClusterOpGetRootChainStakesResponse)),
			byte(wire.ClusterOpGetUnconfirmedHeadersRequest): conn.OpSerializerFor[wire.GetUnconfirmedHeadersRequest, wire.GetUnconfirmedHeadersResponse](byte(wire.ClusterOpGetUnconfirmedHeadersResponse)),
			byte(wire.ClusterOpAddShardBlockHeaderRequest):   conn.OpSerializerFor[wire.AddShardBlockHeaderRequest, wire.AddShardBlockHeaderResponse](byte(wire.ClusterOpAddShardBlockHeaderResponse)),
		},
		Handlers: peerHandlers,
	})
	t.Cleanup(func() { sc.Close(); peer.Close() })
	sc.Start()
	peer.Start()
	return sc, peer
}

func slaveTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestSlaveConnConfig(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*SlaveConnConfig)
	}{
		{"nil connection", func(cfg *SlaveConnConfig) { cfg.Conn = nil }},
		{"nil handler", func(cfg *SlaveConnConfig) { cfg.Handler = nil }},
		{"nil shutdown", func(cfg *SlaveConnConfig) { cfg.ShutdownMaster = nil }},
		{"nil chain ID", func(cfg *SlaveConnConfig) { cfg.RemoteChainID = nil }},
		{"zero chain ID", func(cfg *SlaveConnConfig) { cfg.RemoteChainID = new(big.Int) }},
		{"negative chain ID", func(cfg *SlaveConnConfig) { cfg.RemoteChainID = big.NewInt(-1) }},
		{"oversized chain ID", func(cfg *SlaveConnConfig) { cfg.RemoteChainID = new(big.Int).Lsh(big.NewInt(1), 256) }},
		{"oversized payload", func(cfg *SlaveConnConfig) { cfg.MaxPayloadSize = wire.MaxFramePayloadLen + 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			local, remote := net.Pipe()
			defer local.Close()
			defer remote.Close()
			cfg := SlaveConnConfig{
				Conn: local, RemoteChainID: big.NewInt(1),
				Handler: slaveHandlerFunc{}, ShutdownMaster: noopShutdown,
			}
			test.change(&cfg)
			if connection, err := NewSlaveConn(cfg); err == nil {
				connection.Close()
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

// The peer is a real slave-side MasterConn so both sides' codecs are exercised.
type businessHandler struct {
	roots     chan *qkctypes.RootBlock
	address   common.Address
	blockHash common.Hash
}

func (h *businessHandler) CreateShards(root *qkctypes.RootBlock) error { h.roots <- root; return nil }
func (h *businessHandler) AddRootBlock(root *qkctypes.RootBlock) error { h.roots <- root; return nil }
func (h *businessHandler) GetRootChainStakes(address common.Address, hash common.Hash) (*comm.RootChainStakesResult, error) {
	if address != h.address || hash != h.blockHash {
		return nil, errors.New("unexpected stakes query")
	}
	return &comm.RootChainStakesResult{Stakes: uint256.NewInt(0x10203), Signer: common.Address{7}}, nil
}
func (h *businessHandler) GetUnconfirmedHeaders() ([]*types.Header, error) {
	return []*types.Header{testHeader()}, nil
}

type slaveDialerFunc func(context.Context, wire.SlaveInfo) error

func (f slaveDialerFunc) DialToSlave(ctx context.Context, info wire.SlaveInfo) error {
	return f(ctx, info)
}

func TestSlaveConnRPC(t *testing.T) {
	chainID := new(big.Int).Lsh(big.NewInt(1), 100)
	wantID := new(big.Int).Set(chainID)
	local, remote := net.Pipe()
	h := &businessHandler{roots: make(chan *qkctypes.RootBlock, 2), address: common.Address{5}, blockHash: common.Hash{6}}
	receivedHeader := make(chan *types.Header, 1)
	sc, err := NewSlaveConn(SlaveConnConfig{Conn: local, RemoteChainID: chainID, ShutdownMaster: noopShutdown,
		Handler: slaveHandlerFunc{addShardBlockHeader: func(source *big.Int, header *types.Header, amount *uint256.Int) error {
			if source.Cmp(wantID) != 0 || amount.Cmp(uint256.NewInt(0x10203)) != 0 {
				t.Error("incorrect decoded header notification")
			}
			source.SetInt64(0)
			receivedHeader <- header
			return &comm.RemoteError{Code: 9}
		}}})
	if err != nil {
		t.Fatal(err)
	}
	targets := make(chan wire.SlaveInfo, 1)
	mc, err := slaveconn.NewMasterConn(slaveconn.MasterConnConfig{Conn: remote, LocalChainID: wantID, Handler: h, ShutdownSlave: noopShutdown,
		SlaveDialer: slaveDialerFunc(func(_ context.Context, info wire.SlaveInfo) error { targets <- info; return nil })})
	if err != nil {
		sc.Close()
		remote.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { sc.Close(); mc.Close() })
	sc.Start()
	mc.Start()
	chainID.SetInt64(0)
	sc.RemoteChainID().SetInt64(0)
	ctx := slaveTestContext(t)
	if err := sc.SendPing(ctx, nil); err != nil {
		t.Fatal(err)
	}
	root := testRoot(4)
	if err := sc.SendPing(ctx, root); err != nil {
		t.Fatal(err)
	}
	if got := <-h.roots; got.Hash() != root.Hash() {
		t.Fatal("root tip changed")
	}
	if err := sc.SendAddRootBlock(ctx, root); err != nil {
		t.Fatal(err)
	}
	if got := <-h.roots; got.Hash() != root.Hash() {
		t.Fatal("root block changed")
	}
	if err := sc.SendConnectToSlaves(ctx, []comm.SlaveInfo{{Host: "127.0.0.1", Port: 1234, ChainID: wantID}}); err != nil {
		t.Fatal(err)
	}
	if got := <-targets; string(got.Host) != "127.0.0.1" || got.Port != 1234 || wire.ChainIDFromSSZ(got.ChainID).Cmp(wantID) != 0 {
		t.Fatal("slave endpoint changed")
	}
	stakes, err := sc.SendGetRootChainStakes(ctx, h.address, h.blockHash)
	if err != nil {
		t.Fatal(err)
	}
	if stakes.Stakes.Cmp(uint256.NewInt(0x10203)) != 0 || stakes.Signer != (common.Address{7}) {
		t.Fatalf("incorrect stakes: %+v", stakes)
	}
	headers, err := sc.SendGetUnconfirmedHeaders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(headers) != 1 || headers[0].Hash() != testHeader().Hash() {
		t.Fatal("incorrect unconfirmed headers")
	}
	err = mc.SendAddShardBlockHeader(ctx, testHeader(), uint256.NewInt(0x10203))
	var remoteErr *comm.RemoteError
	if !errors.As(err, &remoteErr) || remoteErr.Code != 9 {
		t.Fatalf("expected rejection 9, got %v", err)
	}
	if got := <-receivedHeader; got.Hash() != testHeader().Hash() {
		t.Fatal("shard header changed")
	}
	if sc.RemoteChainID().Cmp(wantID) != 0 {
		t.Fatal("remote chain ID mutated")
	}
}

func TestSlaveConnRejectsWrongIdentity(t *testing.T) {
	for _, chainID := range []*big.Int{new(big.Int), big.NewInt(2), new(big.Int).Lsh(big.NewInt(1), 100)} {
		t.Run(chainID.String(), func(t *testing.T) {
			encoded, err := wire.ChainIDToSSZ(chainID)
			if err != nil {
				t.Fatal(err)
			}
			sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, map[byte]conn.TypedHandler{
				byte(wire.ClusterOpPing): func(wire.SSZMessage) (wire.SSZMessage, error) {
					return &wire.PongResponse{ChainID: encoded}, nil
				},
			})
			if err := sc.SendPing(slaveTestContext(t), nil); err == nil || !strings.Contains(err.Error(), "chain ID mismatch") {
				t.Fatalf("expected identity mismatch, got %v", err)
			}
			if !sc.IsClosed() {
				t.Fatal("identity mismatch left connection open")
			}
		})
	}
}

func TestSlaveConnConnectResults(t *testing.T) {
	for _, test := range []struct {
		name    string
		results [][]byte
		wantErr string
	}{
		{"success", [][]byte{nil, nil}, ""},
		{"too few", [][]byte{nil}, "returned 1 connection results, want 2"},
		{"too many", [][]byte{nil, nil, nil}, "returned 3 connection results, want 2"},
		{"first failure", [][]byte{[]byte("dial failed"), nil}, "failed to connect to slave 2"},
		{"last failure", [][]byte{nil, []byte("ping failed")}, "failed to connect to slave 3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, map[byte]conn.TypedHandler{
				byte(wire.ClusterOpConnectToSlavesRequest): func(wire.SSZMessage) (wire.SSZMessage, error) {
					return &wire.ConnectToSlavesResponse{ResultList: test.results}, nil
				},
			})
			err := sc.SendConnectToSlaves(slaveTestContext(t), []comm.SlaveInfo{{Host: "127.0.0.1", Port: 1, ChainID: big.NewInt(2)}, {Host: "127.0.0.1", Port: 2, ChainID: big.NewInt(3)}})
			if test.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}
}

func TestSlaveConnRemoteErrorsKeepConnection(t *testing.T) {
	for _, operation := range []string{"root block", "stakes"} {
		t.Run(operation, func(t *testing.T) {
			code := uint32(17)
			sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, map[byte]conn.TypedHandler{
				byte(wire.ClusterOpAddRootBlockRequest): func(req wire.SSZMessage) (wire.SSZMessage, error) {
					var result uint32
					var root qkctypes.RootBlock
					if err := root.UnmarshalSSZ(req.(*wire.AddRootBlockRequest).RootBlock); err != nil {
						return nil, err
					}
					if root.Header().Number != 0 {
						result = code
					}
					return &wire.AddRootBlockResponse{ErrorCode: result}, nil
				},
				byte(wire.ClusterOpGetRootChainStakesRequest): func(req wire.SSZMessage) (wire.SSZMessage, error) {
					var result uint32
					if req.(*wire.GetRootChainStakesRequest).Address != (common.Address{}) {
						result = code
					}
					return &wire.GetRootChainStakesResponse{ErrorCode: result}, nil
				},
			})
			ctx := slaveTestContext(t)
			var err error
			if operation == "root block" {
				err = sc.SendAddRootBlock(ctx, testRoot(1))
			} else {
				var response *comm.RootChainStakesResult
				response, err = sc.SendGetRootChainStakes(ctx, common.Address{1}, common.Hash{})
				if response != nil {
					t.Fatal("business failure returned a success response")
				}
			}
			var remoteErr *comm.RemoteError
			if !errors.As(err, &remoteErr) || remoteErr.Code != code {
				t.Fatalf("expected remote error code %d, got %v", code, err)
			}
			if err := sc.SendAddRootBlock(ctx, testRoot(0)); err != nil {
				t.Fatalf("connection failed after business rejection: %v", err)
			}
		})
	}
}

func TestSlaveConnHandlerFailureClosesConnection(t *testing.T) {
	sc, peer := newSlaveTestPair(t, big.NewInt(1), slaveHandlerFunc{addShardBlockHeader: func(*big.Int, *types.Header, *uint256.Int) error {
		return errors.New("backend unavailable")
	}}, nil)
	_, err := peer.SendRPC(slaveTestContext(t), byte(wire.ClusterOpAddShardBlockHeaderRequest), headerRequest(t))
	if !errors.Is(err, conn.ErrConnectionClosed) {
		t.Fatalf("expected connection closed, got %v", err)
	}
	if !sc.IsClosed() {
		t.Fatal("handler failure left master connection open")
	}
}

func TestSlaveConnContextErrorsKeepConnection(t *testing.T) {
	for _, test := range []struct {
		name string
		want error
	}{
		{"cancel", context.Canceled},
		{"timeout", context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			entered := make(chan struct{})
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, map[byte]conn.TypedHandler{
				byte(wire.ClusterOpAddRootBlockRequest): func(req wire.SSZMessage) (wire.SSZMessage, error) {
					var root qkctypes.RootBlock
					if err := root.UnmarshalSSZ(req.(*wire.AddRootBlockRequest).RootBlock); err != nil {
						return nil, err
					}
					if root.Header().Number != 0 {
						close(entered)
						<-release
					}
					return &wire.AddRootBlockResponse{}, nil
				},
			})
			timeout := 5 * time.Second
			if test.want == context.DeadlineExceeded {
				timeout = 200 * time.Millisecond
			}
			ctx, cancel := context.WithTimeout(t.Context(), timeout)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- sc.SendAddRootBlock(ctx, testRoot(1)) }()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("request did not reach slave")
			}
			if test.want == context.Canceled {
				cancel()
			}
			if err := <-result; !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
			if err := sc.SendAddRootBlock(slaveTestContext(t), testRoot(0)); err != nil {
				t.Fatalf("request after context error: %v", err)
			}
		})
	}
}

func TestSlaveConnPeerFramesDoNotReachLocalDispatch(t *testing.T) {
	received := make(chan *types.Header, 2)
	sc, peer := newSlaveTestPair(t, big.NewInt(1), slaveHandlerFunc{addShardBlockHeader: func(_ *big.Int, header *types.Header, _ *uint256.Int) error {
		received <- header
		return nil
	}}, nil)
	forwarded := &wire.AddShardBlockHeaderRequest{ShardBlockHeader: wire.Data{1}}
	payload, err := forwarded.MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	for _, frame := range []*wire.ClusterFrame{
		{ClusterPeerID: 42, Opcode: byte(wire.ClusterOpAddShardBlockHeaderRequest), RPCID: 0, Payload: payload},
		{ClusterPeerID: 42, Opcode: byte(wire.ClusterOpPong), RPCID: 999},
		{ClusterPeerID: 42, Opcode: 0xff, RPCID: 999},
	} {
		if err := peer.WriteFrame(frame); err != nil {
			t.Fatal(err)
		}
	}
	local := headerRequest(t)
	if _, err := peer.SendRPC(slaveTestContext(t), byte(wire.ClusterOpAddShardBlockHeaderRequest), local); err != nil {
		t.Fatalf("local request after peer frames: %v", err)
	}
	if got := <-received; got.Hash() != testHeader().Hash() {
		t.Fatalf("peer frame reached local handler: %+v", got)
	}
	if sc.IsClosed() {
		t.Fatal("peer frame closed the physical connection")
	}
}

func TestSlaveConnCloseShutsDownMaster(t *testing.T) {
	for _, remoteClose := range []bool{false, true} {
		t.Run(fmt.Sprint(remoteClose), func(t *testing.T) {
			local, remote := net.Pipe()
			defer remote.Close()
			shutdown := make(chan struct{}, 2)
			sc, err := NewSlaveConn(SlaveConnConfig{Conn: local, RemoteChainID: big.NewInt(1), Handler: slaveHandlerFunc{}, ShutdownMaster: func() { shutdown <- struct{}{} }})
			if err != nil {
				local.Close()
				t.Fatal(err)
			}
			defer sc.Close()
			sc.Start()
			if remoteClose {
				remote.Close()
			} else {
				sc.Close()
			}
			select {
			case <-shutdown:
			case <-time.After(5 * time.Second):
				t.Fatal("master shutdown callback was not called")
			}
			sc.Close()
			select {
			case <-shutdown:
				t.Fatal("master shutdown callback called more than once")
			case <-time.After(10 * time.Millisecond):
			}
		})
	}
}

func TestSlaveConnUnconfirmedHeadersResponse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *wire.GetUnconfirmedHeadersResponse
		wantErr  string
	}{
		{"empty", &wire.GetUnconfirmedHeadersResponse{}, ""},
		{"malformed header", &wire.GetUnconfirmedHeadersResponse{HeaderList: []wire.Data{{1}}}, "decode unconfirmed header 0"},
		{"rejection skips decoding", &wire.GetUnconfirmedHeadersResponse{ErrorCode: 17, HeaderList: []wire.Data{{1}}}, "remote error code"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, map[byte]conn.TypedHandler{
				byte(wire.ClusterOpGetUnconfirmedHeadersRequest): func(wire.SSZMessage) (wire.SSZMessage, error) { return tc.response, nil },
			})
			headers, err := sc.SendGetUnconfirmedHeaders(slaveTestContext(t))
			if tc.wantErr == "" {
				if err != nil || len(headers) != 0 {
					t.Fatalf("headers=%v, err=%v", headers, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) || headers != nil {
				t.Fatalf("headers=%v, err=%v, want %q", headers, err, tc.wantErr)
			}
			if sc.IsClosed() {
				t.Fatal("response error closed connection")
			}
		})
	}
}

func TestSlaveConnMalformedHeaderClosesConnection(t *testing.T) {
	called := make(chan struct{}, 1)
	sc, peer := newSlaveTestPair(t, big.NewInt(1), slaveHandlerFunc{addShardBlockHeader: func(*big.Int, *types.Header, *uint256.Int) error { called <- struct{}{}; return nil }}, nil)
	_, err := peer.SendRPC(slaveTestContext(t), byte(wire.ClusterOpAddShardBlockHeaderRequest), &wire.AddShardBlockHeaderRequest{ShardBlockHeader: wire.Data{1}})
	if !errors.Is(err, conn.ErrConnectionClosed) || !sc.IsClosed() {
		t.Fatalf("expected connection closure, got %v", err)
	}
	select {
	case <-called:
		t.Fatal("malformed header reached business handler")
	default:
	}
}

func TestSlaveConnRejectsInvalidBusinessInput(t *testing.T) {
	sc, _ := newSlaveTestPair(t, big.NewInt(1), nil, nil)
	if err := sc.SendAddRootBlock(slaveTestContext(t), nil); err == nil {
		t.Fatal("nil root block accepted")
	}
	for _, id := range []*big.Int{nil, new(big.Int), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 256)} {
		if err := sc.SendConnectToSlaves(slaveTestContext(t), []comm.SlaveInfo{{Host: "127.0.0.1", Port: 1, ChainID: id}}); err == nil {
			t.Fatalf("invalid chain ID accepted: %v", id)
		}
	}
	for _, slave := range []comm.SlaveInfo{{Port: 1, ChainID: big.NewInt(2)}, {Host: "127.0.0.1", ChainID: big.NewInt(2)}} {
		if err := sc.SendConnectToSlaves(slaveTestContext(t), []comm.SlaveInfo{slave}); err == nil {
			t.Fatalf("invalid slave address accepted: %+v", slave)
		}
	}
	if sc.IsClosed() {
		t.Fatal("invalid business input closed connection")
	}
}
