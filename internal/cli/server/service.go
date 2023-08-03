package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

	grpc_net_conn "github.com/JekaMas/go-grpc-net-conn"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/internal/cli/server/pprof"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

const chunkSize = 1024 * 1024 * 1024

var ErrUnavailable = errors.New("bor service is currently unavailable, try again later")
var ErrUnavailable2 = errors.New("bor service unavailable even after waiting for 10 seconds, make sure bor is running")

func sendStreamDebugFile(stream proto.Bor_DebugPprofServer, headers map[string]string, data []byte) error {
	// open the stream and send the headers
	err := stream.Send(&proto.DebugFileResponse{
		Event: &proto.DebugFileResponse_Open_{
			Open: &proto.DebugFileResponse_Open{
				Headers: headers,
			},
		},
	})
	if err != nil {
		return err
	}

	// Wrap our conn around the response.
	encoder := grpc_net_conn.SimpleEncoder(func(msg *proto.DebugFileResponse_Input) *[]byte {
		return &msg.Data
	})

	conn := &grpc_net_conn.Conn[*proto.DebugFileResponse_Input, *proto.DebugFileResponse_Input]{
		Stream:  stream,
		Request: &proto.DebugFileResponse_Input{},
		Encode:  grpc_net_conn.ChunkedEncoder(encoder, chunkSize),
	}

	if _, err := conn.Write(data); err != nil {
		return err
	}

	// send the eof
	err = stream.Send(&proto.DebugFileResponse{
		Event: &proto.DebugFileResponse_Eof{},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) DebugPprof(req *proto.DebugPprofRequest, stream proto.Bor_DebugPprofServer) error {
	var (
		payload []byte
		headers map[string]string
		err     error
	)

	ctx := context.Background()

	switch req.Type {
	case proto.DebugPprofRequest_CPU:
		payload, headers, err = pprof.CPUProfile(ctx, int(req.Seconds))
	case proto.DebugPprofRequest_TRACE:
		payload, headers, err = pprof.Trace(ctx, int(req.Seconds))
	case proto.DebugPprofRequest_LOOKUP:
		payload, headers, err = pprof.Profile(req.Profile, 0, 0)
	}

	if err != nil {
		return err
	}

	// send the file on a grpc stream
	if err := sendStreamDebugFile(stream, headers, payload); err != nil {
		return err
	}

	return nil
}

func (s *Server) PeersAdd(ctx context.Context, req *proto.PeersAddRequest) (*proto.PeersAddResponse, error) {
	node, err := enode.Parse(enode.ValidSchemes, req.Enode)
	if err != nil {
		return nil, fmt.Errorf("invalid enode: %v", err)
	}
	srv := s.node.Server()
	if req.Trusted {
		srv.AddTrustedPeer(node)
	} else {
		srv.AddPeer(node)
	}
	return &proto.PeersAddResponse{}, nil
}

func (s *Server) PeersRemove(ctx context.Context, req *proto.PeersRemoveRequest) (*proto.PeersRemoveResponse, error) {
	node, err := enode.Parse(enode.ValidSchemes, req.Enode)
	if err != nil {
		return nil, fmt.Errorf("invalid enode: %v", err)
	}
	srv := s.node.Server()
	if req.Trusted {
		srv.RemoveTrustedPeer(node)
	} else {
		srv.RemovePeer(node)
	}
	return &proto.PeersRemoveResponse{}, nil
}

func (s *Server) PeersList(ctx context.Context, req *proto.PeersListRequest) (*proto.PeersListResponse, error) {
	resp := &proto.PeersListResponse{}

	peers := s.node.Server().PeersInfo()
	for _, p := range peers {
		resp.Peers = append(resp.Peers, peerInfoToPeer(p))
	}
	return resp, nil
}

func (s *Server) PeersStatus(ctx context.Context, req *proto.PeersStatusRequest) (*proto.PeersStatusResponse, error) {
	var peerInfo *p2p.PeerInfo
	for _, p := range s.node.Server().PeersInfo() {
		if strings.HasPrefix(p.ID, req.Enode) {
			if peerInfo != nil {
				return nil, fmt.Errorf("more than one peer with the same prefix")
			}
			peerInfo = p
		}
	}
	resp := &proto.PeersStatusResponse{}
	if peerInfo != nil {
		resp.Peer = peerInfoToPeer(peerInfo)
	}
	return resp, nil
}

func peerInfoToPeer(info *p2p.PeerInfo) *proto.Peer {
	return &proto.Peer{
		Id:      info.ID,
		Enode:   info.Enode,
		Enr:     info.ENR,
		Caps:    info.Caps,
		Name:    info.Name,
		Trusted: info.Network.Trusted,
		Static:  info.Network.Static,
	}
}

func (s *Server) ChainSetHead(ctx context.Context, req *proto.ChainSetHeadRequest) (*proto.ChainSetHeadResponse, error) {
	s.backend.APIBackend.SetHead(req.Number)
	return &proto.ChainSetHeadResponse{}, nil
}

func (s *Server) Status(ctx context.Context, in *proto.StatusRequest) (*proto.StatusResponse, error) {
	if s.backend == nil && !in.Wait {
		return nil, ErrUnavailable
	}

	// check for s.backend at an interval of 2 seconds
	// wait for a maximum of 10 seconds (5 iterations)
	if s.backend == nil && in.Wait {
		i := 1

		for {
			time.Sleep(2 * time.Second)

			if s.backend == nil {
				if i == 5 {
					return nil, ErrUnavailable2
				}
			} else {
				break
			}
			i++
		}
	}

	apiBackend := s.backend.APIBackend
	syncProgress := apiBackend.SyncProgress()

	resp := &proto.StatusResponse{
		CurrentHeader: headerToProtoHeader(apiBackend.CurrentHeader()),
		CurrentBlock:  headerToProtoHeader(apiBackend.CurrentBlock().Header()),
		NumPeers:      int64(len(s.node.Server().PeersInfo())),
		SyncMode:      s.config.SyncMode,
		Syncing: &proto.StatusResponse_Syncing{
			StartingBlock: int64(syncProgress.StartingBlock),
			HighestBlock:  int64(syncProgress.HighestBlock),
			CurrentBlock:  int64(syncProgress.CurrentBlock),
		},
		Forks: gatherForks(s.config.chain.Genesis.Config, s.config.chain.Genesis.Config.Bor),
	}
	return resp, nil
}

func headerToProtoHeader(h *types.Header) *proto.Header {
	return &proto.Header{
		Hash:   h.Hash().String(),
		Number: h.Number.Uint64(),
	}
}

func (s *Server) DebugBlock(req *proto.DebugBlockRequest, stream proto.Bor_DebugBlockServer) error {
	traceReq := &tracers.TraceBlockRequest{
		Number: req.Number,
		Config: &tracers.TraceConfig{
			Config: &logger.Config{
				EnableMemory: true,
			},
		},
	}

	res, err := s.tracerAPI.TraceBorBlock(traceReq)
	if err != nil {
		return err
	}

	// this is memory heavy
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}

	if err := sendStreamDebugFile(stream, map[string]string{}, data); err != nil {
		return err
	}

	return nil
}

var bigIntT = reflect.TypeOf(new(big.Int)).Kind()

// gatherForks gathers all the fork numbers via reflection
func gatherForks(configList ...interface{}) []*proto.StatusResponse_Fork {
	var forks []*proto.StatusResponse_Fork

	for _, config := range configList {
		kind := reflect.TypeOf(config)
		for kind.Kind() == reflect.Ptr {
			kind = kind.Elem()
		}

		skip := "DAOForkBlock"

		conf := reflect.ValueOf(config).Elem()
		for i := 0; i < kind.NumField(); i++ {
			// Fetch the next field and skip non-fork rules
			field := kind.Field(i)
			if strings.Contains(field.Name, skip) {
				continue
			}
			if !strings.HasSuffix(field.Name, "Block") {
				continue
			}

			fork := &proto.StatusResponse_Fork{
				Name: strings.TrimSuffix(field.Name, "Block"),
			}

			val := conf.Field(i)
			switch field.Type.Kind() {
			case bigIntT:
				rule := val.Interface().(*big.Int)
				if rule != nil {
					fork.Block = rule.Int64()
				} else {
					fork.Disabled = true
				}
			case reflect.Uint64:
				fork.Block = int64(val.Uint())

			default:
				continue
			}

			forks = append(forks, fork)
		}
	}
	return forks
}

func convertBlockToBlockStub(blocks []*types.Block) []*proto.BlockStub {

	var blockStubs []*proto.BlockStub

	for _, block := range blocks {
		blockStub := &proto.BlockStub{
			Hash:   block.Hash().String(),
			Number: block.NumberU64(),
		}
		blockStubs = append(blockStubs, blockStub)
	}

	return blockStubs
}

func (s *Server) ChainWatch(req *proto.ChainWatchRequest, reply proto.Bor_ChainWatchServer) error {

	chain2HeadChanSize := 10

	chain2HeadCh := make(chan core.Chain2HeadEvent, chain2HeadChanSize)
	headSub := s.backend.APIBackend.SubscribeChain2HeadEvent(chain2HeadCh)
	defer headSub.Unsubscribe()

	for {
		msg := <-chain2HeadCh

		err := reply.Send(&proto.ChainWatchResponse{Type: msg.Type,
			Newchain: convertBlockToBlockStub(msg.NewChain),
			Oldchain: convertBlockToBlockStub(msg.OldChain),
		})
		if err != nil {
			return err
		}
	}
}
