package main

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"path"
	"slices"
	"strings"
	"syscall"

	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/portalnetwork/beacon"
	"github.com/ethereum/go-ethereum/portalnetwork/ethapi"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
	"github.com/ethereum/go-ethereum/portalnetwork/state"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/portalnetwork/web3"
	"github.com/ethereum/go-ethereum/rpc"
	_ "github.com/mattn/go-sqlite3"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/urfave/cli/v2"
)

type Config struct {
	Protocol     *discover.PortalProtocolConfig
	PrivateKey   *ecdsa.PrivateKey
	RpcAddr      string
	DataDir      string
	DataCapacity uint64
	LogLevel     int
	Networks     []string
}

type Client struct {
	DiscV5API      *discover.DiscV5API
	HistoryNetwork *history.HistoryNetwork
	BeaconNetwork  *beacon.BeaconNetwork
	StateNetwork   *state.StateNetwork
	Server         *http.Server
}

var app = flags.NewApp("the go-portal-network command line interface")

var (
	portalProtocolFlags = []cli.Flag{
		utils.PortalNATFlag,
		utils.PortalUDPPortFlag,
		utils.PortalBootNodesFlag,
		utils.PortalPrivateKeyFlag,
		utils.PortalNetworksFlag,
	}
	historyRpcFlags = []cli.Flag{
		utils.PortalRPCListenAddrFlag,
		utils.PortalRPCPortFlag,
		utils.PortalDataDirFlag,
		utils.PortalDataCapacityFlag,
		utils.PortalLogLevelFlag,
	}
)

func init() {
	app.Action = shisui
	app.Flags = flags.Merge(portalProtocolFlags, historyRpcFlags)
	flags.AutoEnvVars(app.Flags, "SHISUI")
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func shisui(ctx *cli.Context) error {
	config, err := getPortalConfig(ctx)
	if err != nil {
		return nil
	}

	setDefaultLogger(*config)

	clientChan := make(chan *Client, 1)
	go sigInterrupt(clientChan)

	addr, err := net.ResolveUDPAddr("udp", config.Protocol.ListenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	return startPortalRpcServer(*config, conn, config.RpcAddr, clientChan)
}

func setDefaultLogger(config Config) {
	glogger := log.NewGlogHandler(log.NewTerminalHandler(os.Stderr, true))
	slogVerbosity := log.FromLegacyLevel(config.LogLevel)
	glogger.Verbosity(slogVerbosity)
	defaultLogger := log.NewLogger(glogger)
	log.SetDefault(defaultLogger)
}

func sigInterrupt(clientChan <-chan *Client) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	<-interrupt
	log.Warn("Closing Shisui gracefully (type CTRL-C again to force quit)")

	go func() {
		if len(clientChan) == 0 {
			log.Warn("Waiting for the client to start...")
		}
		c := <-clientChan
		closePortalRpcServer(c)
	}()

	<-interrupt
	os.Exit(1)
}

func closePortalRpcServer(client *Client) {
	log.Info("Closing Database...")
	client.DiscV5API.DiscV5.LocalNode().Database().Close()
	log.Info("Closing UDPv5 protocol...")
	client.DiscV5API.DiscV5.Close()
	if client.HistoryNetwork != nil {
		log.Info("Closing history network...")
		client.HistoryNetwork.Stop()
	}
	if client.BeaconNetwork != nil {
		log.Info("Closing beacon network...")
		client.BeaconNetwork.Stop()
	}
	if client.StateNetwork != nil {
		log.Info("Closing state network...")
		client.StateNetwork.Stop()
	}
	log.Info("Closing servers...")
	client.Server.Close()

	os.Exit(1)
}

func startPortalRpcServer(config Config, conn discover.UDPConn, addr string, clientChan chan<- *Client) error {
	client := &Client{}

	discV5, localNode, err := initDiscV5(config, conn)
	if err != nil {
		return err
	}

	server := rpc.NewServer()
	discV5API := discover.NewDiscV5API(discV5)
	err = server.RegisterName("discv5", discV5API)
	if err != nil {
		return err
	}
	client.DiscV5API = discV5API

	api := &web3.API{}
	err = server.RegisterName("web3", api)
	if err != nil {
		return err
	}

	var historyNetwork *history.HistoryNetwork
	if slices.Contains(config.Networks, portalwire.History.Name()) {
		historyNetwork, err = initHistory(config, server, conn, localNode, discV5)
		if err != nil {
			return err
		}
		client.HistoryNetwork = historyNetwork
	}

	var beaconNetwork *beacon.BeaconNetwork
	if slices.Contains(config.Networks, portalwire.Beacon.Name()) {
		beaconNetwork, err = initBeacon(config, server, conn, localNode, discV5)
		if err != nil {
			return err
		}
		client.BeaconNetwork = beaconNetwork
	}

	var stateNetwork *state.StateNetwork
	if slices.Contains(config.Networks, portalwire.State.Name()) {
		stateNetwork, err = initState(config, server, conn, localNode, discV5)
		if err != nil {
			return err
		}
		client.StateNetwork = stateNetwork
	}

	ethapi := &ethapi.API{
		History: historyNetwork,
		//static configuration of ChainId, currently only mainnet implemented
		ChainID: core.DefaultGenesisBlock().Config.ChainID,
	}
	err = server.RegisterName("eth", ethapi)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    addr,
		Handler: server,
	}
	client.Server = httpServer

	clientChan <- client
	return httpServer.ListenAndServe()
}

func initDiscV5(config Config, conn discover.UDPConn) (*discover.UDPv5, *enode.LocalNode, error) {
	discCfg := discover.Config{
		PrivateKey:  config.PrivateKey,
		NetRestrict: config.Protocol.NetRestrict,
		Bootnodes:   config.Protocol.BootstrapNodes,
		Log:         log.New("protocol", "discV5"),
	}

	nodeDB, err := enode.OpenDB(config.Protocol.NodeDBPath)
	if err != nil {
		return nil, nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, config.PrivateKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localNode.Set(discover.Tag)

	discV5, err := discover.ListenV5(conn, localNode, discCfg)
	if err != nil {
		return nil, nil, err
	}
	return discV5, localNode, nil
}

func initHistory(config Config, server *rpc.Server, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5) (*history.HistoryNetwork, error) {
	networkName := portalwire.History.Name()
	db, err := history.NewDB(config.DataDir, networkName)
	if err != nil {
		return nil, err
	}
	contentStorage, err := history.NewHistoryStorage(storage.PortalStorageConfig{
		StorageCapacityMB: config.DataCapacity,
		DB:                db,
		NodeId:            localNode.ID(),
		NetworkName:       networkName,
	})
	if err != nil {
		return nil, err
	}
	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config.Protocol, portalwire.History, config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return nil, err
	}
	historyAPI := discover.NewPortalAPI(protocol)
	historyNetworkAPI := history.NewHistoryNetworkAPI(historyAPI)
	err = server.RegisterName("portal", historyNetworkAPI)
	if err != nil {
		return nil, err
	}
	accumulator, err := history.NewMasterAccumulator()
	if err != nil {
		return nil, err
	}
	historyNetwork := history.NewHistoryNetwork(protocol, &accumulator)
	return historyNetwork, historyNetwork.Start()
}

func initBeacon(config Config, server *rpc.Server, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5) (*beacon.BeaconNetwork, error) {
	dbPath := path.Join(config.DataDir, "beacon")
	err := os.MkdirAll(dbPath, 0755)
	if err != nil {
		return nil, err
	}
	sqlDb, err := sql.Open("sqlite3", path.Join(dbPath, "beacon.sqlite"))
	if err != nil {
		return nil, err
	}

	contentStorage, err := beacon.NewBeaconStorage(storage.PortalStorageConfig{
		StorageCapacityMB: config.DataCapacity,
		DB:                sqlDb,
		NodeId:            localNode.ID(),
		Spec:              configs.Mainnet,
	})
	if err != nil {
		return nil, err
	}
	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config.Protocol, portalwire.Beacon, config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return nil, err
	}
	portalApi := discover.NewPortalAPI(protocol)

	beaconAPI := beacon.NewBeaconNetworkAPI(portalApi)
	err = server.RegisterName("portal", beaconAPI)
	if err != nil {
		return nil, err
	}

	beaconNetwork := beacon.NewBeaconNetwork(protocol)
	return beaconNetwork, beaconNetwork.Start()
}

func initState(config Config, server *rpc.Server, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5) (*state.StateNetwork, error) {
	networkName := portalwire.State.Name()
	db, err := history.NewDB(config.DataDir, networkName)
	if err != nil {
		return nil, err
	}
	contentStorage, err := history.NewHistoryStorage(storage.PortalStorageConfig{
		StorageCapacityMB: config.DataCapacity,
		DB:                db,
		NodeId:            localNode.ID(),
		NetworkName:       networkName,
	})
	if err != nil {
		return nil, err
	}
	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config.Protocol, portalwire.State, config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return nil, err
	}
	api := discover.NewPortalAPI(protocol)
	stateNetworkAPI := state.NewStateNetworkAPI(api)
	err = server.RegisterName("portal", stateNetworkAPI)
	if err != nil {
		return nil, err
	}
	client := rpc.DialInProc(server)
	historyNetwork := state.NewStateNetwork(protocol, client)
	return historyNetwork, historyNetwork.Start()
}

func getPortalConfig(ctx *cli.Context) (*Config, error) {
	config := &Config{
		Protocol: discover.DefaultPortalProtocolConfig(),
	}
	err := setPrivateKey(ctx, config)
	if err != nil {
		return config, err
	}

	httpAddr := ctx.String(utils.PortalRPCListenAddrFlag.Name)
	httpPort := ctx.String(utils.PortalRPCPortFlag.Name)
	config.RpcAddr = net.JoinHostPort(httpAddr, httpPort)
	config.DataDir = ctx.String(utils.PortalDataDirFlag.Name)
	config.DataCapacity = ctx.Uint64(utils.PortalDataCapacityFlag.Name)
	config.LogLevel = ctx.Int(utils.PortalLogLevelFlag.Name)
	port := ctx.String(utils.PortalUDPPortFlag.Name)
	if !strings.HasPrefix(port, ":") {
		config.Protocol.ListenAddr = ":" + port
	} else {
		config.Protocol.ListenAddr = port
	}

	natString := ctx.String(utils.PortalNATFlag.Name)
	if natString != "" {
		natInterface, err := nat.Parse(natString)
		if err != nil {
			return config, err
		}
		config.Protocol.NAT = natInterface
	}

	setPortalBootstrapNodes(ctx, config)
	config.Networks = ctx.StringSlice(utils.PortalNetworksFlag.Name)
	return config, nil
}

func setPrivateKey(ctx *cli.Context, config *Config) error {
	var privateKey *ecdsa.PrivateKey
	var err error
	keyStr := ctx.String(utils.PortalPrivateKeyFlag.Name)
	if keyStr != "" {
		keyBytes, err := hexutil.Decode(keyStr)
		if err != nil {
			return err
		}
		privateKey, err = crypto.ToECDSA(keyBytes)
		if err != nil {
			return err
		}
	} else {
		privateKey, err = crypto.GenerateKey()
		if err != nil {
			return err
		}
	}
	config.PrivateKey = privateKey
	return nil
}

// setPortalBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setPortalBootstrapNodes(ctx *cli.Context, config *Config) {
	urls := params.PortalBootnodes
	if ctx.IsSet(utils.PortalBootNodesFlag.Name) {
		flag := ctx.String(utils.PortalBootNodesFlag.Name)
		if flag == "none" {
			return
		}
		urls = utils.SplitAndTrim(flag)
	}

	for _, url := range urls {
		if url != "" {
			node, err := enode.Parse(enode.ValidSchemes, url)
			if err != nil {
				log.Error("Bootstrap URL invalid", "enode", url, "err", err)
				continue
			}
			config.Protocol.BootstrapNodes = append(config.Protocol.BootstrapNodes, node)
		}
	}
}
