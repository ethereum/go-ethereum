package beacon

import (
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func setupBeaconNetwork(addr string, bootNodes []*enode.Node) (*BeaconNetwork, error) {
	conf := discover.DefaultPortalProtocolConfig()
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}

	addr1, err := net.ResolveUDPAddr("udp", conf.ListenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr1)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}

	discCfg := discover.Config{
		PrivateKey:  privKey,
		NetRestrict: conf.NetRestrict,
		Bootnodes:   conf.BootstrapNodes,
	}

	nodeDB, err := enode.OpenDB(conf.NodeDBPath)
	if err != nil {
		return nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, privKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localNode.Set(discover.Tag)

	discV5, err := discover.ListenV5(conn, localNode, discCfg)
	if err != nil {
		return nil, err
	}

	contentQueue := make(chan *discover.ContentElement, 50)

	portalProtocol, err := discover.NewPortalProtocol(conf, portalwire.Beacon, privKey, conn, localNode, discV5, &storage.MockStorage{Db: make(map[string][]byte)}, contentQueue)
	if err != nil {
		return nil, err
	}

	return NewBeaconNetwork(portalProtocol), nil
}

func TestBeaconNetworkContent(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	node1, err := setupBeaconNetwork(":6666", nil)
	assert.NoError(t, err)
	node1.log = logger
	node1.portalProtocol.Log = logger
	err = node1.Start()
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)

	node2, err := setupBeaconNetwork(":6667", []*enode.Node{node1.portalProtocol.Self()})
	assert.NoError(t, err)
	node2.log = logger
	node2.portalProtocol.Log = logger
	err = node2.Start()
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)

	node3, err := setupBeaconNetwork(":6668", []*enode.Node{node1.portalProtocol.Self()})
	assert.NoError(t, err)
	node3.log = logger
	node3.portalProtocol.Log = logger
	err = node3.Start()
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)

	filePath := "testdata/light_client_updates_by_range.json"
	jsonStr, _ := os.ReadFile(filePath)

	var result map[string]interface{}
	err = json.Unmarshal(jsonStr, &result)
	assert.NoError(t, err)

	id := node1.portalProtocol.Self().ID()
	for _, v := range result {
		kBytes, _ := hexutil.Decode(v.(map[string]interface{})["content_key"].(string))
		vBytes, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))
		contentKey := storage.NewContentKey(LightClientUpdate, kBytes).Encode()
		_, err = node1.portalProtocol.Gossip(&id, [][]byte{contentKey}, [][]byte{vBytes})
		assert.NoError(t, err)
		time.Sleep(3 * time.Second)

		contentId := node2.portalProtocol.ToContentId(contentKey)
		get, err := node2.portalProtocol.Get(contentKey, contentId)
		assert.NoError(t, err)
		assert.Equal(t, vBytes, get)
	}

	filePath1 := "testdata/light_client_finality_update.json"
	jsonStr1, _ := os.ReadFile(filePath1)

	var result1 map[string]interface{}
	err = json.Unmarshal(jsonStr1, &result1)
	assert.NoError(t, err)

	for _, v := range result1 {
		kBytes1, _ := hexutil.Decode(v.(map[string]interface{})["content_key"].(string))
		vBytes1, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))

		contentKey1 := storage.NewContentKey(LightClientFinalityUpdate, kBytes1).Encode()

		id = node1.portalProtocol.Self().ID()
		_, err = node1.portalProtocol.Gossip(&id, [][]byte{contentKey1}, [][]byte{vBytes1})
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		contentId1 := node2.portalProtocol.ToContentId(contentKey1)
		get1, err := node2.portalProtocol.Get(contentKey1, contentId1)
		assert.NoError(t, err)
		assert.Equal(t, vBytes1, get1)
	}

	filePath2 := "testdata/light_client_optimistic_update.json"
	jsonStr2, _ := os.ReadFile(filePath2)

	var result2 map[string]interface{}
	err = json.Unmarshal(jsonStr2, &result2)
	assert.NoError(t, err)

	for _, v := range result2 {
		kBytes2, _ := hexutil.Decode(v.(map[string]interface{})["content_key"].(string))
		vBytes2, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))

		contentKey2 := storage.NewContentKey(LightClientOptimisticUpdate, kBytes2).Encode()

		id = node1.portalProtocol.Self().ID()
		_, err = node1.portalProtocol.Gossip(&id, [][]byte{contentKey2}, [][]byte{vBytes2})
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		contentId2 := node2.portalProtocol.ToContentId(contentKey2)
		get2, err := node2.portalProtocol.Get(contentKey2, contentId2)
		assert.NoError(t, err)
		assert.Equal(t, vBytes2, get2)
	}

	filePath3 := "testdata/light_client_bootstrap.json"
	jsonStr3, _ := os.ReadFile(filePath3)

	var result3 map[string]interface{}
	err = json.Unmarshal(jsonStr3, &result3)
	assert.NoError(t, err)

	for _, v := range result3 {
		kBytes3, _ := hexutil.Decode(v.(map[string]interface{})["content_key"].(string))
		vBytes3, _ := hexutil.Decode(v.(map[string]interface{})["content_value"].(string))

		contentKey3 := storage.NewContentKey(LightClientBootstrap, kBytes3).Encode()

		id = node1.portalProtocol.Self().ID()
		_, err = node1.portalProtocol.Gossip(&id, [][]byte{contentKey3}, [][]byte{vBytes3})
		assert.NoError(t, err)

		time.Sleep(3 * time.Second)

		contentId3 := node2.portalProtocol.ToContentId(contentKey3)
		get3, err := node2.portalProtocol.Get(contentKey3, contentId3)
		assert.NoError(t, err)
		assert.Equal(t, vBytes3, get3)
	}
}

type Entry struct {
	ContentKey   string `yaml:"content_key"`
	ContentValue string `yaml:"content_value"`
}

func TestGossipTwoNodes(t *testing.T) {
	file, err := os.ReadFile("./testdata/hive/gossip.yaml")
	require.NoError(t, err)
	entries := make([]Entry, 0)
	err = yaml.Unmarshal(file, &entries)
	require.NoError(t, err)

	keys := make([][]byte, 0)
	values := make([][]byte, 0)

	for _, entry := range entries {
		keys = append(keys, hexutil.MustDecode(entry.ContentKey))
		values = append(values, hexutil.MustDecode(entry.ContentValue))
	}

	logger := testlog.Logger(t, log.LvlTrace)
	node1, err := setupBeaconNetwork(":6998", nil)
	assert.NoError(t, err)
	node1.log = logger
	node1.portalProtocol.Log = logger
	err = node1.Start()
	assert.NoError(t, err)

	node2, err := setupBeaconNetwork(":6999", nil)
	assert.NoError(t, err)
	node2.log = logger
	node2.portalProtocol.Log = logger
	err = node2.Start()
	assert.NoError(t, err)

	node2.portalProtocol.AddEnr(node1.portalProtocol.Self())

	id := node2.portalProtocol.Self().ID()

	num, err := node2.portalProtocol.Gossip(&id, keys, values)
	require.NoError(t, err)
	require.Equal(t, num, 1)

	time.Sleep(time.Second * 10)

	for i, key := range keys {
		val := values[i]
		res, err := node1.portalProtocol.Get(key, node1.portalProtocol.ToContentId(key))
		require.NoError(t, err)
		require.Equal(t, res, val)
	}
}
