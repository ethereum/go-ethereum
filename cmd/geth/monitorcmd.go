package main

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/comms"
	"github.com/gizak/termui"
)

var (
	monitorCommandAttachFlag = cli.StringFlag{
		Name:  "attach",
		Value: "ipc:" + common.DefaultIpcPath(),
		Usage: "API endpoint to attach to",
	}
	monitorCommandRowsFlag = cli.IntFlag{
		Name:  "rows",
		Value: 5,
		Usage: "Maximum rows in the chart grid",
	}
	monitorCommandRefreshFlag = cli.IntFlag{
		Name:  "refresh",
		Value: 3,
		Usage: "Refresh interval in seconds",
	}
	monitorCommand = cli.Command{
		Action: monitor,
		Name:   "monitor",
		Usage:  `Geth Monitor: node metrics monitoring and visualization`,
		Description: `
The Geth monitor is a tool to collect and visualize various internal metrics
gathered by the node, supporting different chart types as well as the capacity
to display multiple metrics simultaneously.
`,
		Flags: []cli.Flag{
			monitorCommandAttachFlag,
			monitorCommandRowsFlag,
			monitorCommandRefreshFlag,
		},
	}
)

// monitor starts a terminal UI based monitoring tool for the requested metrics.
func monitor(ctx *cli.Context) {
	var (
		client comms.EthereumClient
		err    error
	)
	// Attach to an Ethereum node over IPC or RPC
	endpoint := ctx.String(monitorCommandAttachFlag.Name)
	if client, err = comms.ClientFromEndpoint(endpoint, codec.JSON); err != nil {
		utils.Fatalf("Unable to attach to geth node: %v", err)
	}
	defer client.Close()

	xeth := rpc.NewXeth(client)

	// Retrieve all the available metrics and resolve the user pattens
	metrics, err := retrieveMetrics(xeth)
	if err != nil {
		utils.Fatalf("Failed to retrieve system metrics: %v", err)
	}
	monitored := resolveMetrics(metrics, ctx.Args())
	if len(monitored) == 0 {
		list := expandMetrics(metrics, "")
		sort.Strings(list)
		utils.Fatalf("No metrics specified.\n\nAvailable:\n - %s", strings.Join(list, "\n - "))
	}
	sort.Strings(monitored)
	if cols := len(monitored) / ctx.Int(monitorCommandRowsFlag.Name); cols > 6 {
		utils.Fatalf("Requested metrics (%d) spans more that 6 columns:\n - %s", len(monitored), strings.Join(monitored, "\n - "))
	}
	// Create and configure the chart UI defaults
	if err := termui.Init(); err != nil {
		utils.Fatalf("Unable to initialize terminal UI: %v", err)
	}
	defer termui.Close()

	termui.UseTheme("helloworld")

	rows := len(monitored)
	if max := ctx.Int(monitorCommandRowsFlag.Name); rows > max {
		rows = max
	}
	cols := (len(monitored) + rows - 1) / rows
	for i := 0; i < rows; i++ {
		termui.Body.AddRows(termui.NewRow())
	}
	// Create each individual data chart
	footer := termui.NewPar("")
	footer.HasBorder = true
	footer.Height = 3

	charts := make([]*termui.LineChart, len(monitored))
	units := make([]int, len(monitored))
	data := make([][]float64, len(monitored))
	for i := 0; i < len(monitored); i++ {
		charts[i] = createChart((termui.TermHeight() - footer.Height) / rows)
		row := termui.Body.Rows[i%rows]
		row.Cols = append(row.Cols, termui.NewCol(12/cols, 0, charts[i]))
	}
	termui.Body.AddRows(termui.NewRow(termui.NewCol(12, 0, footer)))

	refreshCharts(xeth, monitored, data, units, charts, ctx, footer)
	termui.Body.Align()
	termui.Render(termui.Body)

	// Watch for various system events, and periodically refresh the charts
	refresh := time.Tick(time.Duration(ctx.Int(monitorCommandRefreshFlag.Name)) * time.Second)
	for {
		select {
		case event := <-termui.EventCh():
			if event.Type == termui.EventKey && event.Key == termui.KeyCtrlC {
				return
			}
			if event.Type == termui.EventResize {
				termui.Body.Width = termui.TermWidth()
				for _, chart := range charts {
					chart.Height = (termui.TermHeight() - footer.Height) / rows
				}
				termui.Body.Align()
				termui.Render(termui.Body)
			}
		case <-refresh:
			if refreshCharts(xeth, monitored, data, units, charts, ctx, footer) {
				termui.Body.Align()
			}
			termui.Render(termui.Body)
		}
	}
}

// retrieveMetrics contacts the attached geth node and retrieves the entire set
// of collected system metrics.
func retrieveMetrics(xeth *rpc.Xeth) (map[string]interface{}, error) {
	return xeth.Call("debug_metrics", []interface{}{true})
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
	results := []string{}

	// If a nested metric was requested, recurse optionally branching (via comma)
	parts := strings.SplitN(pattern, "/", 2)
	if len(parts) > 1 {
		for _, variation := range strings.Split(parts[0], ",") {
			if submetrics, ok := metrics[variation].(map[string]interface{}); !ok {
				utils.Fatalf("Failed to retrieve system metrics: %s", path+variation)
				return nil
			} else {
				results = append(results, resolveMetric(submetrics, parts[1], path+variation+"/")...)
			}
		}
		return results
	}
	// Depending what the last link is, return or expand
	for _, variation := range strings.Split(pattern, ",") {
		switch metric := metrics[variation].(type) {
		case float64:
			// Final metric value found, return as singleton
			results = append(results, path+variation)

		case map[string]interface{}:
			results = append(results, expandMetrics(metric, path+variation+"/")...)

		default:
			utils.Fatalf("Metric pattern resolved to unexpected type: %v", reflect.TypeOf(metric))
			return nil
		}
	}
	return results
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

// fetchMetric iterates over the metrics map and retrieves a specific one.
func fetchMetric(metrics map[string]interface{}, metric string) float64 {
	parts, found := strings.Split(metric, "/"), true
	for _, part := range parts[:len(parts)-1] {
		metrics, found = metrics[part].(map[string]interface{})
		if !found {
			return 0
		}
	}
	if v, ok := metrics[parts[len(parts)-1]].(float64); ok {
		return v
	}
	return 0
}

// refreshCharts retrieves a next batch of metrics, and inserts all the new
// values into the active datasets and charts
func refreshCharts(xeth *rpc.Xeth, metrics []string, data [][]float64, units []int, charts []*termui.LineChart, ctx *cli.Context, footer *termui.Par) (realign bool) {
	values, err := retrieveMetrics(xeth)
	for i, metric := range metrics {
		if len(data) < 512 {
			data[i] = append([]float64{fetchMetric(values, metric)}, data[i]...)
		} else {
			data[i] = append([]float64{fetchMetric(values, metric)}, data[i][:len(data[i])-1]...)
		}
		if updateChart(metric, data[i], &units[i], charts[i], err) {
			realign = true
		}
	}
	updateFooter(ctx, err, footer)
	return
}

// updateChart inserts a dataset into a line chart, scaling appropriately as to
// not display weird labels, also updating the chart label accordingly.
func updateChart(metric string, data []float64, base *int, chart *termui.LineChart, err error) (realign bool) {
	dataUnits := []string{"", "K", "M", "G", "T", "E"}
	timeUnits := []string{"ns", "µs", "ms", "s", "ks", "ms"}
	colors := []termui.Attribute{termui.ColorBlue, termui.ColorCyan, termui.ColorGreen, termui.ColorYellow, termui.ColorRed, termui.ColorRed}

	// Extract only part of the data that's actually visible
	if chart.Width*2 < len(data) {
		data = data[:chart.Width*2]
	}
	// Find the maximum value and scale under 1K
	high := 0.0
	if len(data) > 0 {
		high = data[0]
		for _, value := range data[1:] {
			high = math.Max(high, value)
		}
	}
	unit, scale := 0, 1.0
	for high >= 1000 {
		high, unit, scale = high/1000, unit+1, scale*1000
	}
	// If the unit changes, re-create the chart (hack to set max height...)
	if unit != *base {
		realign, *base, *chart = true, unit, *createChart(chart.Height)
	}
	// Update the chart's data points with the scaled values
	if cap(chart.Data) < len(data) {
		chart.Data = make([]float64, len(data))
	}
	chart.Data = chart.Data[:len(data)]
	for i, value := range data {
		chart.Data[i] = value / scale
	}
	// Update the chart's label with the scale units
	units := dataUnits
	if strings.Contains(metric, "/Percentiles/") || strings.Contains(metric, "/pauses/") || strings.Contains(metric, "/time/") {
		units = timeUnits
	}
	chart.Border.Label = metric
	if len(units[unit]) > 0 {
		chart.Border.Label += " [" + units[unit] + "]"
	}
	chart.LineColor = colors[unit] | termui.AttrBold
	if err != nil {
		chart.LineColor = termui.ColorRed | termui.AttrBold
	}
	return
}

// createChart creates an empty line chart with the default configs.
func createChart(height int) *termui.LineChart {
	chart := termui.NewLineChart()
	if runtime.GOOS == "windows" {
		chart.Mode = "dot"
	}
	chart.DataLabels = []string{""}
	chart.Height = height
	chart.AxesColor = termui.ColorWhite
	chart.PaddingBottom = -2

	chart.Border.LabelFgColor = chart.Border.FgColor | termui.AttrBold
	chart.Border.FgColor = chart.Border.BgColor

	return chart
}

// updateFooter updates the footer contents based on any encountered errors.
func updateFooter(ctx *cli.Context, err error, footer *termui.Par) {
	// Generate the basic footer
	refresh := time.Duration(ctx.Int(monitorCommandRefreshFlag.Name)) * time.Second
	footer.Text = fmt.Sprintf("Press Ctrl+C to quit. Refresh interval: %v.", refresh)
	footer.TextFgColor = termui.Theme().ParTextFg | termui.AttrBold

	// Append any encountered errors
	if err != nil {
		footer.Text = fmt.Sprintf("Error: %v.", err)
		footer.TextFgColor = termui.ColorRed | termui.AttrBold
	}
}
