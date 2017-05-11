// p2psim provides a command-line client for a simulation API.
//
// Here is an example of creating a 2 node network with the first node
// connected to the second:
//
//     $ p2psim network create
//     Created network net1
//
//     $ p2psim node create net1
//     Created node01
//
//     $ p2psim node start net1 node01
//     Started node01
//
//     $ p2psim node create net1
//     Created node02
//
//     $ p2psim node start net1 node02
//     Started node02
//
//     $ p2psim node connect net1 node01 node02
//     Connected node01 to node02
//
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/simulations"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

var client *simulations.Client

func main() {
	app := cli.NewApp()
	app.Usage = "devp2p simulation command-line client"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "api",
			Value:  "http://localhost:8888",
			Usage:  "simulation API URL",
			EnvVar: "P2PSIM_API_URL",
		},
	}
	app.Before = func(ctx *cli.Context) error {
		client = simulations.NewClient(ctx.GlobalString("api"))
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:   "network",
			Usage:  "manage simulation networks",
			Action: listNetworks,
			Subcommands: []cli.Command{
				{
					Name:   "list",
					Usage:  "list networks",
					Action: listNetworks,
				},
				{
					Name:   "create",
					Usage:  "create a network",
					Action: createNetwork,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "config",
							Value: "{}",
							Usage: "JSON encoded network config",
						},
					},
				},
				{
					Name:      "show",
					ArgsUsage: "<network>",
					Usage:     "show network information",
					Action:    showNetwork,
				},
				{
					Name:      "events",
					ArgsUsage: "<network>",
					Usage:     "stream network events",
					Action:    streamNetwork,
				},
				{
					Name:      "snapshot",
					ArgsUsage: "<network>",
					Usage:     "create a network snapshot to stdout",
					Action:    createSnapshot,
				},
				{
					Name:      "load",
					ArgsUsage: "<network>",
					Usage:     "load a network snapshot from stdin",
					Action:    loadSnapshot,
				},
			},
		},
		{
			Name:   "node",
			Usage:  "manage simulation nodes",
			Action: listNodes,
			Subcommands: []cli.Command{
				{
					Name:      "list",
					ArgsUsage: "<network>",
					Usage:     "list nodes",
					Action:    listNodes,
				},
				{
					Name:      "create",
					ArgsUsage: "<network>",
					Usage:     "create a node",
					Action:    createNode,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "config",
							Value: "{}",
							Usage: "JSON encoded node config",
						},
					},
				},
				{
					Name:      "show",
					ArgsUsage: "<network> <node>",
					Usage:     "show node information",
					Action:    showNode,
				},
				{
					Name:      "start",
					ArgsUsage: "<network> <node>",
					Usage:     "start a node",
					Action:    startNode,
				},
				{
					Name:      "stop",
					ArgsUsage: "<network> <node>",
					Usage:     "stop a node",
					Action:    stopNode,
				},
				{
					Name:      "connect",
					ArgsUsage: "<network> <node> <peer>",
					Usage:     "connect a node to a peer node",
					Action:    connectNode,
				},
				{
					Name:      "disconnect",
					ArgsUsage: "<network> <node> <peer>",
					Usage:     "disconnect a node from a peer node",
					Action:    disconnectNode,
				},
				{
					Name:      "rpc",
					ArgsUsage: "<network> <node> <method> [<args>]",
					Usage:     "call a node RPC method",
					Action:    rpcNode,
					Flags: []cli.Flag{
						cli.BoolFlag{
							Name:  "subscribe",
							Usage: "method is a subscription",
						},
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

func listNetworks(ctx *cli.Context) error {
	if len(ctx.Args()) != 0 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networks, err := client.GetNetworks()
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.App.Writer, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\tNODES\tCONNS\n")
	for _, network := range networks {
		fmt.Fprintf(w, "%s\t%d\t%d\n", network.Id, len(network.Nodes), len(network.Conns))
	}
	return nil
}

func createNetwork(ctx *cli.Context) error {
	if len(ctx.Args()) != 0 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	config := &simulations.NetworkConfig{}
	if err := json.Unmarshal([]byte(ctx.String("config")), config); err != nil {
		return err
	}
	network, err := client.CreateNetwork(config)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Created network", network.Id)
	return nil
}

func showNetwork(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	network, err := client.GetNetwork(networkID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.App.Writer, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "ID\t%s\n", network.Id)
	fmt.Fprintf(w, "NODES\t%d\n", len(network.Nodes))
	fmt.Fprintf(w, "CONNS\t%d\n", len(network.Conns))
	return nil
}

func streamNetwork(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	events := make(chan *simulations.Event)
	sub, err := client.SubscribeNetwork(networkID, events)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	enc := json.NewEncoder(ctx.App.Writer)
	for {
		select {
		case event := <-events:
			if err := enc.Encode(event); err != nil {
				return err
			}
		case err := <-sub.Err():
			return err
		}
	}
}

func createSnapshot(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	snap, err := client.CreateSnapshot(networkID)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(snap)
}

func loadSnapshot(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	snap := &simulations.Snapshot{}
	if err := json.NewDecoder(os.Stdin).Decode(snap); err != nil {
		return err
	}
	return client.LoadSnapshot(networkID, snap)
}

func listNodes(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodes, err := client.GetNodes(networkID)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.App.Writer, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "NAME\tPROTOCOLS\tID\n")
	for _, node := range nodes {
		fmt.Fprintf(w, "%s\t%s\t%s\n", node.Name, strings.Join(protocolList(node), ","), node.ID)
	}
	return nil
}

func protocolList(node *p2p.NodeInfo) []string {
	protos := make([]string, 0, len(node.Protocols))
	for name := range node.Protocols {
		protos = append(protos, name)
	}
	return protos
}

func createNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	config := &adapters.NodeConfig{}
	if err := json.Unmarshal([]byte(ctx.String("config")), config); err != nil {
		return err
	}
	node, err := client.CreateNode(networkID, config)
	if err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Created", node.Name)
	return nil
}

func showNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	node, err := client.GetNode(networkID, nodeName)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(ctx.App.Writer, 1, 2, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "NAME\t%s\n", node.Name)
	fmt.Fprintf(w, "PROTOCOLS\t%s\n", strings.Join(protocolList(node), ","))
	fmt.Fprintf(w, "ID\t%s\n", node.ID)
	fmt.Fprintf(w, "ENODE\t%s\n", node.Enode)
	for name, proto := range node.Protocols {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "--- PROTOCOL INFO: %s\n", name)
		fmt.Fprintf(w, "%v\n", proto)
		fmt.Fprintf(w, "---\n")
	}
	return nil
}

func startNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	if err := client.StartNode(networkID, nodeName); err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Started", nodeName)
	return nil
}

func stopNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	if err := client.StopNode(networkID, nodeName); err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Stopped", nodeName)
	return nil
}

func connectNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 3 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	peerName := args[2]
	if err := client.ConnectNode(networkID, nodeName, peerName); err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Connected", nodeName, "to", peerName)
	return nil
}

func disconnectNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 3 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	peerName := args[2]
	if err := client.DisconnectNode(networkID, nodeName, peerName); err != nil {
		return err
	}
	fmt.Fprintln(ctx.App.Writer, "Disconnected", nodeName, "from", peerName)
	return nil
}

func rpcNode(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) < 3 {
		return cli.ShowCommandHelp(ctx, ctx.Command.Name)
	}
	networkID := args[0]
	nodeName := args[1]
	method := args[2]
	rpcClient, err := client.RPCClient(context.Background(), networkID, nodeName)
	if err != nil {
		return err
	}
	if ctx.Bool("subscribe") {
		return rpcSubscribe(rpcClient, ctx.App.Writer, method, args[3:]...)
	}
	var result interface{}
	params := make([]interface{}, len(args[3:]))
	for i, v := range args[3:] {
		params[i] = v
	}
	if err := rpcClient.Call(&result, method, params...); err != nil {
		return err
	}
	return json.NewEncoder(ctx.App.Writer).Encode(result)
}

func rpcSubscribe(client *rpc.Client, out io.Writer, method string, args ...string) error {
	parts := strings.SplitN(method, "_", 2)
	namespace := parts[0]
	method = parts[1]
	ch := make(chan interface{})
	subArgs := make([]interface{}, len(args)+1)
	subArgs[0] = method
	for i, v := range args {
		subArgs[i+1] = v
	}
	sub, err := client.Subscribe(context.Background(), namespace, ch, subArgs...)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	enc := json.NewEncoder(out)
	for {
		select {
		case v := <-ch:
			if err := enc.Encode(v); err != nil {
				return err
			}
		case err := <-sub.Err():
			return err
		}
	}
}
