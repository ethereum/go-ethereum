package state

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

type TestCase struct {
	BlockHeader           string `yaml:"block_header"`
	ContentKey            string `yaml:"content_key"`
	ContentValueOffer     string `yaml:"content_value_offer"`
	ContentValueRetrieval string `yaml:"content_value_retrieval"`
}

func getTestCases(filename string) ([]TestCase, error) {
	file, err := os.ReadFile(fmt.Sprintf("./testdata/%s", filename))
	if err != nil {
		return nil, err
	}
	res := make([]TestCase, 0)
	err = yaml.Unmarshal(file, &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

type MockAPI struct {
	header string
}

func (p *MockAPI) HistoryGetContent(contentKeyHex string) (*portalwire.ContentInfo, error) {
	headerWithProof := &history.BlockHeaderWithProof{
		Header: hexutil.MustDecode(p.header),
		Proof: &history.BlockHeaderProof{
			Selector: 0,
			Proof:    [][]byte{},
		},
	}
	data, err := headerWithProof.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return &portalwire.ContentInfo{
		Content:     hexutil.Encode(data),
		UtpTransfer: false,
	}, nil
}

func TestValidateAccountTrieNode(t *testing.T) {
	cases, err := getTestCases("account_trie_node.yaml")
	require.NoError(t, err)

	for _, tt := range cases {
		server := rpc.NewServer()
		api := &MockAPI{
			header: tt.BlockHeader,
		}
		server.RegisterName("portal", api)
		client := rpc.DialInProc(server)
		bn := NewStateNetwork(nil, client)
		err = bn.validateContent(hexutil.MustDecode(tt.ContentKey), hexutil.MustDecode(tt.ContentValueOffer))
		require.NoError(t, err)
	}
}

func TestValidateContractStorage(t *testing.T) {
	cases, err := getTestCases("contract_storage_trie_node.yaml")
	require.NoError(t, err)

	for _, tt := range cases {
		server := rpc.NewServer()
		api := &MockAPI{
			header: tt.BlockHeader,
		}
		server.RegisterName("portal", api)
		client := rpc.DialInProc(server)
		bn := NewStateNetwork(nil, client)
		err = bn.validateContent(hexutil.MustDecode(tt.ContentKey), hexutil.MustDecode(tt.ContentValueOffer))
		require.NoError(t, err)
	}
}

func TestValidateContractByte(t *testing.T) {
	cases, err := getTestCases("contract_bytecode.yaml")
	require.NoError(t, err)

	for _, tt := range cases {
		server := rpc.NewServer()
		api := &MockAPI{
			header: tt.BlockHeader,
		}
		server.RegisterName("portal", api)
		client := rpc.DialInProc(server)
		bn := NewStateNetwork(nil, client)
		err = bn.validateContent(hexutil.MustDecode(tt.ContentKey), hexutil.MustDecode(tt.ContentValueOffer))
		require.NoError(t, err)
	}
}
