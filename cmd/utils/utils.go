package utils

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/XDCx"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending"
	"github.com/XinFinOrg/XDPoSChain/eth"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/ethconfig"
	"github.com/XinFinOrg/XDPoSChain/ethstats"
	"github.com/XinFinOrg/XDPoSChain/les"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/node"
)

// RegisterEthService adds an Ethereum client to the stack.
func RegisterEthService(stack *node.Node, cfg *ethconfig.Config, version string) {
	var err error
	if cfg.SyncMode == downloader.LightSync {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, cfg)
		})
	} else {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			var XDCXServ *XDCx.XDCX
			ctx.Service(&XDCXServ)
			var lendingServ *XDCxlending.Lending
			ctx.Service(&lendingServ)
			fullNode, err := eth.New(ctx, cfg, XDCXServ, lendingServ)
			if fullNode != nil && cfg.LightServ > 0 {
				ls, _ := les.NewLesServer(fullNode, cfg)
				fullNode.AddLesServer(ls)
			}

			// TODO: move the following code to function makeFullNode
			// Ref: #21105, #22641, #23761, #24877
			// Create gauge with geth system and build information
			var protos []string
			for _, p := range fullNode.Protocols() {
				protos = append(protos, fmt.Sprintf("%v/%d", p.Name, p.Version))
			}
			metrics.NewRegisteredGaugeInfo("xdc/info", nil).Update(metrics.GaugeInfoValue{
				"arch":          runtime.GOARCH,
				"os":            runtime.GOOS,
				"version":       version, // cfg.Node.Version
				"eth_protocols": strings.Join(protos, ","),
			})

			return fullNode, err
		})
	}
	if err != nil {
		Fatalf("Failed to register the Ethereum service: %v", err)
	}
}

// RegisterEthStatsService configures the Ethereum Stats daemon and adds it to the node.
func RegisterEthStatsService(stack *node.Node, url string) {
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		// Retrieve both eth and les services
		var ethServ *eth.Ethereum
		ctx.Service(&ethServ)

		var lesServ *les.LightEthereum
		ctx.Service(&lesServ)

		return ethstats.New(url, ethServ, lesServ)
	}); err != nil {
		Fatalf("Failed to register the Ethereum Stats service: %v", err)
	}
}

func RegisterXDCXService(stack *node.Node, cfg *XDCx.Config) {
	XDCX := XDCx.New(cfg)
	if err := stack.Register(func(n *node.ServiceContext) (node.Service, error) {
		return XDCX, nil
	}); err != nil {
		Fatalf("Failed to register the XDCX service: %v", err)
	}

	// register XDCxlending service
	if err := stack.Register(func(n *node.ServiceContext) (node.Service, error) {
		return XDCxlending.New(XDCX), nil
	}); err != nil {
		Fatalf("Failed to register the XDCXLending service: %v", err)
	}
}
