package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func ContentId(contentKey []byte) []byte {
	digest := sha256.Sum256(contentKey)
	return digest[:]
}

// testcases from https://github.com/ethereum/portal-network-specs/blob/master/content-keys-test-vectors.md
func TestContentKey(t *testing.T) {
	testCases := []struct {
		name          string
		hash          string
		contentKey    string
		contentIdHex  string
		contentIdU256 string
		selector      ContentType
	}{
		{
			name:          "block header key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "00d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "3e86b3767b57402ea72e369ae0496ce47cc15be685bec3b4726b9f316e3895fe",
			contentIdU256: "28281392725701906550238743427348001871342819822834514257505083923073246729726",
			selector:      BlockHeaderType,
		},
		{
			name:          "block body key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "01d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "ebe414854629d60c58ddd5bf60fd72e41760a5f7a463fdcb169f13ee4a26786b",
			contentIdU256: "106696502175825986237944249828698290888857178633945273402044845898673345165419",
			selector:      BlockBodyType,
		},
		{
			name:          "receipt key",
			hash:          "d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentKey:    "02d1c390624d3bd4e409a61a858e5dcc5517729a9170d014a6c96530d64dd8621d",
			contentIdHex:  "a888f4aafe9109d495ac4d4774a6277c1ada42035e3da5e10a04cc93247c04a4",
			contentIdU256: "76230538398907151249589044529104962263309222250374376758768131420767496438948",
			selector:      ReceiptsType,
		},
		{
			name:          "epoch accumelator key",
			hash:          "e242814b90ed3950e13aac7e56ce116540c71b41d1516605aada26c6c07cc491",
			contentKey:    "03e242814b90ed3950e13aac7e56ce116540c71b41d1516605aada26c6c07cc491",
			contentIdHex:  "9fb2175e76c6989e0fdac3ee10c40d2a81eb176af32e1c16193e3904fe56896e",
			contentIdU256: "72232402989179419196382321898161638871438419016077939952896528930608027961710",
			selector:      EpochAccumulatorType,
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			hashByte, err := hex.DecodeString(c.hash)
			require.NoError(t, err)

			contentKey := newContentKey(c.selector, hashByte).encode()
			hexKey := hex.EncodeToString(contentKey)
			require.Equal(t, hexKey, c.contentKey)
			contentId := ContentId(contentKey)
			require.Equal(t, c.contentIdHex, hex.EncodeToString(contentId))

			bigNum := big.NewInt(0).SetBytes(contentId)
			u256Format, isOverflow := uint256.FromBig(bigNum)
			require.False(t, isOverflow)
			u256Str := fmt.Sprint(u256Format)
			require.Equal(t, u256Str, c.contentIdU256)
		})
	}
}

func TestValidateHeader(t *testing.T) {
	entrys, err := parseBlockHeaderKeyContent()
	require.NoError(t, err)
	historyNetwork, err := genHistoryNetwork(":7891", nil)
	require.NoError(t, err)
	for _, entry := range entrys {
		err = historyNetwork.validateContent(entry.key, entry.value)
		require.NoError(t, err)

		headerWithProof, err := DecodeBlockHeaderWithProof(entry.value)
		require.NoError(t, err)
		// invalid blockhash
		_, err = ValidateBlockHeaderBytes(headerWithProof.Header, entry.key)
		require.Equal(t, ErrInvalidBlockHash, err)
		header, err := ValidateBlockHeaderBytes(headerWithProof.Header, entry.key[1:])
		require.NoError(t, err)
		// wrong header number
		header.Number = big.NewInt(0).Add(header.Number, big.NewInt(122))
		valid, err := historyNetwork.verifyHeader(header, *headerWithProof.Proof)
		require.False(t, valid)
		require.NoError(t, err)
	}
}

func TestGetContentByKey(t *testing.T) {
	historyNetwork1, err := genHistoryNetwork(":7895", nil)
	require.NoError(t, err)
	historyNetwork2, err := genHistoryNetwork(":7896", []*enode.Node{historyNetwork1.portalProtocol.Self()})
	require.NoError(t, err)
	// wait node start
	time.Sleep(10 * time.Second)

	entryMap, err := parseDataForBlock14764013()
	require.NoError(t, err)

	headerEntry := entryMap["header"]

	// test GetBlockHeader
	// no content
	header, err := historyNetwork2.GetBlockHeader(headerEntry.key[1:])
	require.Error(t, err)
	require.Nil(t, header)

	contentId := historyNetwork1.portalProtocol.ToContentId(headerEntry.key)
	err = historyNetwork1.portalProtocol.Put(contentId, headerEntry.value)
	require.NoError(t, err)
	// get content from historyNetwork1
	header, err = historyNetwork2.GetBlockHeader(headerEntry.key[1:])
	require.NoError(t, err)
	require.NotNil(t, header)
	// get content from local
	header, err = historyNetwork2.GetBlockHeader(headerEntry.key[1:])
	require.NoError(t, err)
	require.NotNil(t, header)
}

type contentEntry struct {
	key   []byte
	value []byte
}

func parseBlockHeaderKeyContent() ([]contentEntry, error) {
	headWithProofBytes, err := os.ReadFile("./testdata/header_with_proofs.json")
	if err != nil {
		return nil, err
	}
	headerMap := make(map[string]map[string]string)

	err = json.Unmarshal(headWithProofBytes, &headerMap)
	if err != nil {
		return nil, err
	}
	res := make([]contentEntry, 0)
	for _, v := range headerMap {
		entry := contentEntry{}
		val := v["value"]
		bytes, err := hexutil.Decode(val)
		if err != nil {
			return nil, err
		}
		entry.value = bytes
		key := v["content_key"]
		keyBytes, err := hexutil.Decode(key)
		if err != nil {
			return nil, err
		}
		entry.key = keyBytes
		res = append(res, entry)
	}
	return res, nil
}

type MockStorage struct {
	db map[string][]byte
}

func (m *MockStorage) Get(contentId []byte) ([]byte, error) {
	if content, ok := m.db[string(contentId)]; ok {
		return content, nil
	}
	return nil, storage.ErrContentNotFound
}

func (m *MockStorage) Put(contentId []byte, content []byte) error {
	m.db[string(contentId)] = content
	return nil
}

func genHistoryNetwork(addr string, bootNodes []*enode.Node) (*HistoryNetwork, error) {
	conf := discover.DefaultPortalProtocolConfig()
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}

	contentQueue := make(chan *discover.ContentElement, 50)

	key, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}

	portalProtocol, err := discover.NewPortalProtocol(conf, portalwire.HistoryNetwork, key, &MockStorage{db: make(map[string][]byte)}, contentQueue)
	if err != nil {
		return nil, err
	}

	accu, err := NewMasterAccumulator()
	if err != nil {
		return nil, err
	}

	err = portalProtocol.Start()
	if err != nil {
		return nil, err
	}

	return NewHistoryNetwork(portalProtocol, &accu), nil
}

func parseDataForBlock14764013() (map[string]contentEntry, error) {
	content, err := os.ReadFile("./testdata/block_14764013.json")
	if err != nil {
		return nil, err
	}

	contentMap := make(map[string]map[string]string)
	json.Unmarshal(content, &contentMap)
	res := make(map[string]contentEntry)
	for key, val := range contentMap {
		entry := contentEntry{}
		contentKey := val["content_key"]
		entry.key, err = hexutil.Decode(contentKey)
		if err != nil {
			return nil, err
		}
		entry.value, err = hexutil.Decode(val["content_value"])
		if err != nil {
			return nil, err
		}
		res[key] = entry
	}
	return res, nil
}
