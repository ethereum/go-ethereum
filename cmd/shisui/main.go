package main

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"net/http"
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
	"github.com/ethereum/go-ethereum/portalnetwork/storage/sqlite"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

type Config struct {
	Protocol     *discover.PortalProtocolConfig
	PrivateKey   *ecdsa.PrivateKey
	RpcAddr      string
	DataDir      string
	DataCapacity uint64
	LogLevel     int
}

var app = flags.NewApp("the go-portal-network command line interface")

var (
	portalProtocolFlags = []cli.Flag{
		utils.PortalUDPListenAddrFlag,
		utils.PortalUDPPortFlag,
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

	glogger := log.NewGlogHandler(log.NewTerminalHandler(os.Stderr, true))
	slogVerbosity := log.FromLegacyLevel(config.LogLevel)
	glogger.Verbosity(slogVerbosity)
	defaultLogger := log.NewLogger(glogger)
	log.SetDefault(defaultLogger)

	nodeId := enode.PubkeyToIDV4(&config.PrivateKey.PublicKey)
	contentStorage, err := sqlite.NewContentStorage(config.DataCapacity, nodeId, config.DataDir)
	if err != nil {
		return err
	}

	addr, err := net.ResolveUDPAddr("udp", config.Protocol.ListenAddr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	discCfg := discover.Config{
		PrivateKey:  config.PrivateKey,
		NetRestrict: config.Protocol.NetRestrict,
		Bootnodes:   config.Protocol.BootstrapNodes,
		Log:         defaultLogger,
	}

	nodeDB, err := enode.OpenDB(config.Protocol.NodeDBPath)
	if err != nil {
		return err
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
			return err
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
		return err
	}

	contentQueue := make(chan *discover.ContentElement, 50)

	historyProtocol, err := discover.NewPortalProtocol(config.Protocol, string(portalwire.HistoryNetwork), config.PrivateKey, conn, localNode, discV5, contentStorage, contentQueue)

	if err != nil {
		return err
	}

	accumulator, err := history.NewMasterAccumulator()
	if err != nil {
		return err
	}

	historyNetwork := history.NewHistoryNetwork(historyProtocol, &accumulator)
	err = historyNetwork.Start()
	if err != nil {
		return err
	}
	defer historyNetwork.Stop()

	startPortalRpcServer(discover.NewDiscV5API(discV5), discover.NewPortalAPI(historyProtocol), nil, config.RpcAddr)
	return nil
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

	if ctx.IsSet(utils.PortalUDPListenAddrFlag.Name) {
		ip := ctx.String(utils.PortalUDPListenAddrFlag.Name)
		netIp := net.ParseIP(ip)
		if netIp == nil {
			return config, fmt.Errorf("invalid ip addr: %s", ip)
		}
		config.Protocol.NodeIP = netIp
	}

	if ctx.IsSet(utils.PortalBootNodesFlag.Name) {
		for _, node := range ctx.StringSlice(utils.PortalBootNodesFlag.Name) {
			bootNode := new(enode.Node)
			err = bootNode.UnmarshalText([]byte(node))
			if err != nil {
				return config, err
			}
			config.Protocol.BootstrapNodes = append(config.Protocol.BootstrapNodes, bootNode)
		}
	}
	return config, nil
}

func setPrivateKey(ctx *cli.Context, config *Config) error {
	var privateKey *ecdsa.PrivateKey
	var err error
	if ctx.IsSet(utils.PortalPrivateKeyFlag.Name) {
		keyStr := ctx.String(utils.PortalPrivateKeyFlag.Name)
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

func startPortalRpcServer(discV5API *discover.DiscV5API, historyAPI *discover.PortalProtocolAPI, beaconAPI *discover.PortalProtocolAPI, addr string) error {
	disv5 := discV5API

	server := rpc.NewServer()
	err := server.RegisterName("discv5", disv5)
	if err != nil {
		return err
	}

	var historyNetworkAPI *history.API
	if historyAPI != nil {
		historyNetworkAPI = history.NewHistoryNetworkAPI(historyAPI)
		err = server.RegisterName("portal", historyNetworkAPI)

		if err != nil {
			return err
		}
	}

	var beaconNetworkAPI *beacon.API
	if beaconAPI != nil {
		beaconNetworkAPI = beacon.NewBeaconNetworkAPI(beaconAPI)
		err = server.RegisterName("portal", beaconNetworkAPI)

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
