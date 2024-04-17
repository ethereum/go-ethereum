package main

import (
	"crypto/ecdsa"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"path"
	"slices"
	"strings"

	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/discover/portalwire"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/beacon"
	"github.com/ethereum/go-ethereum/portalnetwork/history"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	"github.com/ethereum/go-ethereum/portalnetwork/storage/sqlite"
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

var app = flags.NewApp("the go-portal-network command line interface")

var (
	portalProtocolFlags = []cli.Flag{
		utils.PortalUDPListenAddrFlag,
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

	addr, err := net.ResolveUDPAddr("udp", config.Protocol.ListenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	return startPortalRpcServer(*config, conn, config.RpcAddr)
}

func setDefaultLogger(config Config) {
	glogger := log.NewGlogHandler(log.NewTerminalHandler(os.Stderr, true))
	slogVerbosity := log.FromLegacyLevel(config.LogLevel)
	glogger.Verbosity(slogVerbosity)
	defaultLogger := log.NewLogger(glogger)
	log.SetDefault(defaultLogger)
}

func startPortalRpcServer(config Config, conn discover.UDPConn, addr string) error {
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

	if slices.Contains(config.Networks, portalwire.HistoryNetworkName) {
		err = initHistory(config, server, conn, localNode, discV5)
		if err != nil {
			return err
		}
	}

	if slices.Contains(config.Networks, portalwire.BeaconNetworkName) {
		err = initBeacon(config, server, conn, localNode, discV5)
		if err != nil {
			return err
		}
	}

	httpServer := &http.Server{
		Addr:    addr,
		Handler: server,
	}
	httpServer.ListenAndServe()
	return nil
}

func initDiscV5(config Config, conn discover.UDPConn) (*discover.UDPv5, *enode.LocalNode, error) {
	discCfg := discover.Config{
		PrivateKey:  config.PrivateKey,
		NetRestrict: config.Protocol.NetRestrict,
		Bootnodes:   config.Protocol.BootstrapNodes,
		Log:         log.New("discV5"),
	}

	nodeDB, err := enode.OpenDB(config.Protocol.NodeDBPath)
	if err != nil {
		return nil, nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, config.PrivateKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localNode.Set(discover.Tag)

	var addrs []net.Addr
	if config.Protocol.NodeIP != nil {
		localNode.SetStaticIP(config.Protocol.NodeIP)
	} else {
		addrs, err = net.InterfaceAddrs()

		if err != nil {
			return nil, nil, err
		}

		for _, address := range addrs {
			// check ip addr is loopback addr
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					localNode.SetStaticIP(ipnet.IP)
					break
				}
			}
		}
	}

	discV5, err := discover.ListenV5(conn, localNode, discCfg)
	if err != nil {
		return nil, nil, err
	}
	return discV5, localNode, nil
}

func initHistory(config Config, server *rpc.Server, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5) error {
	contentStorage, err := sqlite.NewContentStorage(config.DataCapacity, localNode.ID(), config.DataDir)
	if err != nil {
		return err
	}
	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config.Protocol, string(portalwire.HistoryNetwork), config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return err
	}
	historyAPI := discover.NewPortalAPI(protocol)
	historyNetworkAPI := history.NewHistoryNetworkAPI(historyAPI)
	err = server.RegisterName("portal", historyNetworkAPI)
	if err != nil {
		return err
	}
	accumulator, err := history.NewMasterAccumulator()
	if err != nil {
		return err
	}
	historyNetwork := history.NewHistoryNetwork(protocol, &accumulator)
	return historyNetwork.Start()
}

func initBeacon(config Config, server *rpc.Server, conn discover.UDPConn, localNode *enode.LocalNode, discV5 *discover.UDPv5) error {
	dbPath := path.Join(config.DataDir, "beacon")
	err := os.MkdirAll(dbPath, 0755)
	if err != nil {
		return err
	}
	sqlDb, err := sql.Open("sqlite3", path.Join(dbPath, "beacon.sqlite"))
	if err != nil {
		return err
	}

	contentStorage, err := beacon.NewBeaconStorage(storage.PortalStorageConfig{
		StorageCapacityMB: config.DataCapacity,
		DB:                sqlDb,
		NodeId:            localNode.ID(),
		Spec:              configs.Mainnet,
	})
	if err != nil {
		return err
	}
	contentQueue := make(chan *discover.ContentElement, 50)

	protocol, err := discover.NewPortalProtocol(config.Protocol, string(portalwire.BeaconLightClientNetwork), config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return err
	}
	portalApi := discover.NewPortalAPI(protocol)

	beaconAPI := beacon.NewBeaconNetworkAPI(portalApi)
	err = server.RegisterName("portal", beaconAPI)
	if err != nil {
		return err
	}

	beaconNetwork := beacon.NewBeaconNetwork(protocol)
	return beaconNetwork.Start()
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

	udpAddr := ctx.String(utils.PortalUDPListenAddrFlag.Name)
	if udpAddr != "" {
		ip := udpAddr
		netIp := net.ParseIP(ip)
		if netIp == nil {
			return config, fmt.Errorf("invalid ip addr: %s", ip)
		}
		config.Protocol.NodeIP = netIp
	}

	bootNodes := ctx.StringSlice(utils.PortalBootNodesFlag.Name)
	if len(bootNodes) > 0 {
		for _, node := range bootNodes {
			bootNode := new(enode.Node)
			err = bootNode.UnmarshalText([]byte(node))
			if err != nil {
				return config, err
			}
			config.Protocol.BootstrapNodes = append(config.Protocol.BootstrapNodes, bootNode)
		}
	}
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
