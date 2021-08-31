// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package debug

import (
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/metrics/exp"
	"github.com/fjl/memsize/memsizeui"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"gopkg.in/urfave/cli.v1"
)

var Memsize memsizeui.Handler

var (
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	vmoduleFlag = cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value: "",
	}
	logjsonFlag = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Format logs with JSON",
	}
	backtraceAtFlag = cli.StringFlag{
		Name:  "log.backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: "",
	}
	debugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	pprofFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the pprof HTTP server",
	}
	pprofPortFlag = cli.IntFlag{
		Name:  "pprof.port",
		Usage: "pprof HTTP server listening port",
		Value: 6060,
	}
	pprofAddrFlag = cli.StringFlag{
		Name:  "pprof.addr",
		Usage: "pprof HTTP server listening interface",
		Value: "127.0.0.1",
	}
	memprofilerateFlag = cli.IntFlag{
		Name:  "pprof.memprofilerate",
		Usage: "Turn on memory profiling with the given rate",
		Value: runtime.MemProfileRate,
	}
	blockprofilerateFlag = cli.IntFlag{
		Name:  "pprof.blockprofilerate",
		Usage: "Turn on block profiling with the given rate",
	}
	cpuprofileFlag = cli.StringFlag{
		Name:  "pprof.cpuprofile",
		Usage: "Write CPU profile to the given file",
	}
	traceFlag = cli.StringFlag{
		Name:  "trace",
		Usage: "Write execution trace to the given file",
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag,
	vmoduleFlag,
	logjsonFlag,
	backtraceAtFlag,
	debugFlag,
	pprofFlag,
	pprofAddrFlag,
	pprofPortFlag,
	memprofilerateFlag,
	blockprofilerateFlag,
	cpuprofileFlag,
	traceFlag,
}

var glogger *log.GlogHandler

func init() {
	glogger = log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx *cli.Context) error {
	var ostream log.Handler
	output := io.Writer(os.Stderr)
	if ctx.GlobalBool(logjsonFlag.Name) {
		ostream = log.StreamHandler(output, log.JSONFormat())
	} else {
		usecolor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if usecolor {
			output = colorable.NewColorableStderr()
		}
		ostream = log.StreamHandler(output, log.TerminalFormat(usecolor))
	}
	glogger.SetHandler(ostream)

	// logging
	verbosity := ctx.GlobalInt(verbosityFlag.Name)
	glogger.Verbosity(log.Lvl(verbosity))
	vmodule := ctx.GlobalString(vmoduleFlag.Name)
	glogger.Vmodule(vmodule)

	debug := ctx.GlobalBool(debugFlag.Name)
	if ctx.GlobalIsSet(debugFlag.Name) {
		debug = ctx.GlobalBool(debugFlag.Name)
	}
	log.PrintOrigins(debug)

	backtrace := ctx.GlobalString(backtraceAtFlag.Name)
	glogger.BacktraceAt(backtrace)

	log.Root().SetHandler(glogger)

	// profiling, tracing
	runtime.MemProfileRate = memprofilerateFlag.Value
	if ctx.GlobalIsSet(memprofilerateFlag.Name) {
		runtime.MemProfileRate = ctx.GlobalInt(memprofilerateFlag.Name)
	}

	blockProfileRate := ctx.GlobalInt(blockprofilerateFlag.Name)
	Handler.SetBlockProfileRate(blockProfileRate)

	if traceFile := ctx.GlobalString(traceFlag.Name); traceFile != "" {
		if err := Handler.StartGoTrace(traceFile); err != nil {
			return err
		}
	}

	if cpuFile := ctx.GlobalString(cpuprofileFlag.Name); cpuFile != "" {
		if err := Handler.StartCPUProfile(cpuFile); err != nil {
			return err
		}
	}

	// pprof server
	if ctx.GlobalBool(pprofFlag.Name) {
		listenHost := ctx.GlobalString(pprofAddrFlag.Name)

		port := ctx.GlobalInt(pprofPortFlag.Name)

		address := fmt.Sprintf("%s:%d", listenHost, port)
		// This context value ("metrics.addr") represents the utils.MetricsHTTPFlag.Name.
		// It cannot be imported because it will cause a cyclical dependency.
		StartPProf(address, !ctx.GlobalIsSet("metrics.addr"))
	}
	return nil
}

func StartPProf(address string, withMetrics bool) {
	// Hook go-metrics into expvar on any /debug/metrics request, load all vars
	// from the registry into expvar, and execute regular expvar handler.
	if withMetrics {
		exp.Exp(metrics.DefaultRegistry)
	}
	http.Handle("/memsize/", http.StripPrefix("/memsize", &Memsize))
	log.Info("Starting pprof server", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
	go func() {
		if err := http.ListenAndServe(address, nil); err != nil {
			log.Error("Failure in running pprof server", "err", err)
		}
	}()
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	Handler.StopCPUProfile()
	Handler.StopGoTrace()
}
