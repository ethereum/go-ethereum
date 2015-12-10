package eth

import (
	"math/big"
	"errors"

	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/common"
	rpc "github.com/ethereum/go-ethereum/rpc/v2"
)

type EthService struct {
	e   *Ethereum
	gpo *GasPriceOracle
}

func NewEthService(e *Ethereum) *EthService {
	return &EthService{e, NewGasPriceOracle(e)}
}

// GasPrice returns a suggestion for a gas price.
func (s *EthService) GasPrice() *big.Int {
	return s.gpo.SuggestPrice()
}

// GetCompilers returns the collection of available smart contract compilers
func (s *EthService) GetCompilers() ([]string, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc != nil {
		return []string{"Solidity"}, nil
	}

	return nil, nil
}

// CompileSolidity compiles the given solidity source
func (s *EthService) CompileSolidity(source string) (map[string]*compiler.Contract, error) {
	solc, err := s.e.Solc()
	if err != nil {
		return nil, err
	}

	if solc == nil {
		return nil, errors.New("solc (solidity compiler) not found")
	}

	return solc.Compile(source)
}

func (s *EthService) Etherbase() (common.Address, error) {
	return s.e.Etherbase()
}

func (s *EthService) Coinbase() (common.Address, error) {
	return s.Etherbase()
}

func (s *EthService) ProtocolVersion() *rpc.HexNumber {
	return rpc.NewHexNumber(s.e.EthVersion())
}

func (s *EthService) Syncing() (interface{}, error) {
	origin, current, height := s.e.Downloader().Progress()
	if current < height {
		return map[string]interface{}{
			"startingBlock": rpc.NewHexNumber(origin),
			"currentBlock":  rpc.NewHexNumber(current),
			"highestBlock":  rpc.NewHexNumber(height),
		}, nil
	}
	return false, nil
}
