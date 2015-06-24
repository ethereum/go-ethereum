package main

import (
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/gizak/termui"
)

// monitor starts a terminal UI based monitoring tool for the requested metrics.
func monitor(ctx *cli.Context) {
	var (
		client comms.EthereumClient
		args   []string
		err    error
	)
	// Attach to an Ethereum node over IPC or RPC
	if ctx.Args().Present() {
		// Try to interpret the first parameter as an endpoint
		client, err = comms.ClientFromEndpoint(ctx.Args().First(), codec.JSON)
		if err == nil {
			args = ctx.Args().Tail()
		}
	}
	if !ctx.Args().Present() || err != nil {
		// Either no args were given, or not endpoint, use defaults
		cfg := comms.IpcConfig{
			Endpoint: ctx.GlobalString(utils.IPCPathFlag.Name),
		}
		args = ctx.Args()
		client, err = comms.NewIpcClient(cfg, codec.JSON)
	}
	if err != nil {
		utils.Fatalf("Unable to attach to geth node - %v", err)
	}
	defer client.Close()

	xeth := rpc.NewXeth(client)

	// Retrieve all the available metrics and resolve the user pattens
	metrics, err := xeth.Call("debug_metrics", []interface{}{true})
	if err != nil {
		utils.Fatalf("Failed to retrieve system metrics: %v", err)
	}
	monitored := resolveMetrics(metrics, args)
	sort.Strings(monitored)

	// Create the access function and check that the metric exists
	value := func(metrics map[string]interface{}, metric string) float64 {
		parts, found := strings.Split(metric, "/"), true
		for _, part := range parts[:len(parts)-1] {
			metrics, found = metrics[part].(map[string]interface{})
			if !found {
				utils.Fatalf("Metric not found: %s", metric)
			}
		}
		if v, ok := metrics[parts[len(parts)-1]].(float64); ok {
			return v
		}
		utils.Fatalf("Metric not float64: %s", metric)
		return 0
	}
	// Assemble the terminal UI
	if err := termui.Init(); err != nil {
		utils.Fatalf("Unable to initialize terminal UI: %v", err)
	}
	defer termui.Close()

	termui.UseTheme("helloworld")

	charts := make([]*termui.LineChart, len(monitored))
	for i, metric := range monitored {
		charts[i] = termui.NewLineChart()
		charts[i].Border.Label = metric
		charts[i].Data = make([]float64, 512)
		charts[i].DataLabels = []string{""}
		charts[i].Height = termui.TermHeight() / len(monitored)
		charts[i].AxesColor = termui.ColorWhite
		charts[i].LineColor = termui.ColorGreen

		termui.Body.AddRows(termui.NewRow(termui.NewCol(12, 0, charts[i])))
	}
	termui.Body.Align()
	termui.Render(termui.Body)

	refresh := time.Tick(time.Second)
	for {
		select {
		case event := <-termui.EventCh():
			if event.Type == termui.EventKey && event.Ch == 'q' {
				return
			}
			if event.Type == termui.EventResize {
				termui.Body.Width = termui.TermWidth()
				for _, chart := range charts {
					chart.Height = termui.TermHeight() / len(monitored)
				}
				termui.Body.Align()
				termui.Render(termui.Body)
			}
		case <-refresh:
			metrics, err := xeth.Call("debug_metrics", []interface{}{true})
			if err != nil {
				utils.Fatalf("Failed to retrieve system metrics: %v", err)
			}
			for i, metric := range monitored {
				charts[i].Data = append([]float64{value(metrics, metric)}, charts[i].Data[:len(charts[i].Data)-1]...)
			}
			termui.Render(termui.Body)
		}
	}
}

// resolveMetrics takes a list of input metric patterns, and resolves each to one
// or more canonical metric names.
func resolveMetrics(metrics map[string]interface{}, patterns []string) []string {
	res := []string{}
	for _, pattern := range patterns {
		res = append(res, resolveMetric(metrics, pattern, "")...)
	}
	return res
}

// resolveMetrics takes a single of input metric pattern, and resolves it to one
// or more canonical metric names.
func resolveMetric(metrics map[string]interface{}, pattern string, path string) []string {
	var ok bool

	// Build up the canonical metric path
	parts := strings.Split(pattern, "/")
	for len(parts) > 1 {
		if metrics, ok = metrics[parts[0]].(map[string]interface{}); !ok {
			utils.Fatalf("Failed to retrieve system metrics: %s", path+parts[0])
		}
		path += parts[0] + "/"
		parts = parts[1:]
	}
	// Depending what the last link is, return or expand
	switch metric := metrics[parts[0]].(type) {
	case float64:
		// Final metric value found, return as singleton
		return []string{path + parts[0]}

	case map[string]interface{}:
		return expandMetrics(metric, path+parts[0]+"/")

	default:
		utils.Fatalf("Metric pattern resolved to unexpected type: %v", reflect.TypeOf(metric))
		return nil
	}
}

// expandMetrics expands the entire tree of metrics into a flat list of paths.
func expandMetrics(metrics map[string]interface{}, path string) []string {
	// Iterate over all fields and expand individually
	list := []string{}
	for name, metric := range metrics {
		switch metric := metric.(type) {
		case float64:
			// Final metric value found, append to list
			list = append(list, path+name)

		case map[string]interface{}:
			// Tree of metrics found, expand recursively
			list = append(list, expandMetrics(metric, path+name+"/")...)

		default:
			utils.Fatalf("Metric pattern %s resolved to unexpected type: %v", path+name, reflect.TypeOf(metric))
			return nil
		}
	}
	return list
}
