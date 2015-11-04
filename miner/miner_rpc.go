package miner

import (
	"github.com/ethereum/go-ethereum/common"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
	"fmt"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type MinerService struct {
	miner *Miner
	agent *RemoteAgent
}

func NewMinerService(miner *Miner) *MinerService {
	return &MinerService{miner, NewRemoteAgent()}
}

// Mining returns an indication if this node is currently mining.
func (s *MinerService) Mining() bool {
	return s.miner.Mining()
}

// SubmitWork can be used by external miner to submit their POW solution. It returns an indication if the work was
// accepted. Note, this is not an indication if the provided work was valid!
func (s *MinerService) SubmitWork(nonce rpc.HexNumber, solution, digest common.Hash) bool {
	return s.agent.SubmitWork(nonce.Uint64(), digest, solution)
}

// GetWork returns a work package for external miner. The work package consists of 3 strings
// result[0], 32 bytes hex encoded current block header pow-hash
// result[1], 32 bytes hex encoded seed hash used for DAG
// result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (s *MinerService) GetWork() ([]string, error) {
	if !s.Mining() {
		s.miner.Start(s.miner.coinbase, 0)
	}
	if work, err := s.agent.GetWork(); err == nil {
		return work[:], nil
	} else {
		glog.Infof("%v\n", err)
	}
	return nil, fmt.Errorf("mining not ready")
}

// SubmitHashrate can be used for remote miners to submit their hash rate. This enables the node to report the combined
// hash rate of all miners which submit work through this node. It accepts the miner hash rate and an identifier which
// must be unique between nodes.
func (s *MinerService) SubmitHashrate(hashrate rpc.HexNumber, id common.Hash) bool {
	s.agent.SubmitHashrate(id, hashrate.Uint64())
	return true
}
