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
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"

	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/metrics/exp"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"gopkg.in/natefinch/lumberjack.v2"
	"gopkg.in/urfave/cli.v1"
)

var (
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	logVmoduleFlag = &cli.StringFlag{
		Name:  "log-vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value: "",
	}
	vmoduleFlag = cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. eth/*=5,p2p=4)",
		Value: "",
	}
	logjsonFlag = &cli.BoolFlag{
		Name:   "log-json",
		Usage:  "Format logs with JSON",
		Hidden: true,
	}
	logFormatFlag = &cli.StringFlag{
		Name:  "log-format",
		Usage: "Log format to use (json|logfmt|terminal)",
	}
	logFileFlag = &cli.StringFlag{
		Name:  "log-file",
		Usage: "Write logs to a file",
	}
	logRotateFlag = &cli.BoolFlag{
		Name:  "log-rotate",
		Usage: "Enables log file rotation",
	}
	logMaxSizeMBsFlag = &cli.IntFlag{
		Name:  "log-maxsize",
		Usage: "Maximum size in MBs of a single log file",
		Value: 100,
	}
	logMaxBackupsFlag = &cli.IntFlag{
		Name:  "log-maxbackups",
		Usage: "Maximum number of log files to retain",
		Value: 10,
	}
	logMaxAgeFlag = &cli.IntFlag{
		Name:  "log-maxage",
		Usage: "Maximum number of days to retain a log file",
		Value: 30,
	}
	logCompressFlag = &cli.BoolFlag{
		Name:  "log-compress",
		Usage: "Compress the log files",
	}
	pprofFlag = cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the pprof HTTP server",
	}
	pprofPortFlag = cli.IntFlag{
		Name:  "pprofport",
		Usage: "pprof HTTP server listening port",
		Value: 6060,
	}
	pprofAddrFlag = cli.StringFlag{
		Name:  "pprofaddr",
		Usage: "pprof HTTP server listening interface",
		Value: "127.0.0.1",
	}
	memprofilerateFlag = cli.IntFlag{
		Name:  "memprofilerate",
		Usage: "Turn on memory profiling with the given rate",
		Value: runtime.MemProfileRate,
	}
	blockprofilerateFlag = cli.IntFlag{
		Name:  "blockprofilerate",
		Usage: "Turn on block profiling with the given rate",
	}
	cpuprofileFlag = cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "Write CPU profile to the given file",
	}
	traceFlag = cli.StringFlag{
		Name:  "trace",
		Usage: "Write execution trace to the given file",
	}
	periodicProfilingFlag = cli.BoolFlag{
		Name:  "periodicprofile",
		Usage: "Periodically profile cpu and memory status",
	}
	debugDataDirFlag = cli.StringFlag{
		Name:  "debugdatadir",
		Usage: "Debug Data directory for profiling output",
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag,
	logVmoduleFlag,
	vmoduleFlag,
	logjsonFlag,
	logFormatFlag,
	logFileFlag,
	logRotateFlag,
	logMaxSizeMBsFlag,
	logMaxBackupsFlag,
	logMaxAgeFlag,
	logCompressFlag,
	pprofFlag,
	pprofAddrFlag,
	pprofPortFlag,
	memprofilerateFlag,
	//blockprofilerateFlag,
	cpuprofileFlag,
	//traceFlag,
	periodicProfilingFlag,
	debugDataDirFlag,
}

var (
	glogger                *log.GlogHandler
	logOutputFile          io.WriteCloser
	defaultTerminalHandler *log.TerminalHandler
)

func init() {
	defaultTerminalHandler = log.NewTerminalHandler(os.Stderr, false)
	glogger = log.NewGlogHandler(defaultTerminalHandler)
	glogger.Verbosity(log.LvlInfo)
	log.SetDefault(log.NewLogger(glogger))
}

func ResetLogging() {
	if defaultTerminalHandler != nil {
		defaultTerminalHandler.ResetFieldPadding()
	}
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx *cli.Context) error {
	var (
		handler        slog.Handler
		terminalOutput = io.Writer(os.Stderr)
		output         io.Writer
		logFmtFlag     = ctx.GlobalString(logFormatFlag.Name)
	)
	var (
		logFile  = ctx.GlobalString(logFileFlag.Name)
		rotation = ctx.GlobalBool(logRotateFlag.Name)
	)
	if len(logFile) > 0 {
		if err := validateLogLocation(filepath.Dir(logFile)); err != nil {
			return fmt.Errorf("failed to initiatilize file logger: %v", err)
		}
	}
	context := []interface{}{"rotate", rotation}
	if len(logFmtFlag) > 0 {
		context = append(context, "format", logFmtFlag)
	} else {
		context = append(context, "format", "terminal")
	}
	if rotation {
		// Lumberjack uses <processname>-lumberjack.log in is.TempDir() if empty.
		// so typically /tmp/geth-lumberjack.log on linux
		if len(logFile) > 0 {
			context = append(context, "location", logFile)
		} else {
			context = append(context, "location", filepath.Join(os.TempDir(), "geth-lumberjack.log"))
		}
		logOutputFile = &lumberjack.Logger{
			Filename:   logFile,
			MaxSize:    ctx.GlobalInt(logMaxSizeMBsFlag.Name),
			MaxBackups: ctx.GlobalInt(logMaxBackupsFlag.Name),
			MaxAge:     ctx.GlobalInt(logMaxAgeFlag.Name),
			Compress:   ctx.GlobalBool(logCompressFlag.Name),
		}
		output = io.MultiWriter(terminalOutput, logOutputFile)
	} else if logFile != "" {
		var err error
		if logOutputFile, err = os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err != nil {
			return err
		}
		output = io.MultiWriter(logOutputFile, terminalOutput)
		context = append(context, "location", logFile)
	} else {
		output = terminalOutput
	}

	switch {
	case ctx.GlobalBool(logjsonFlag.Name):
		// Retain backwards compatibility with `--log-json` flag if `--log-format` not set
		defer log.Warn("The flag '--log-json' is deprecated, please use '--log-format=json' instead")
		handler = log.JSONHandlerWithLevel(output, log.LevelInfo)
	case logFmtFlag == "json":
		handler = log.JSONHandlerWithLevel(output, log.LevelInfo)
	case logFmtFlag == "logfmt":
		handler = log.LogfmtHandler(output)
	case logFmtFlag == "", logFmtFlag == "terminal":
		useColor := (isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
		if useColor {
			terminalOutput = colorable.NewColorableStderr()
			if logOutputFile != nil {
				output = io.MultiWriter(logOutputFile, terminalOutput)
			} else {
				output = terminalOutput
			}
		}
		handler = log.NewTerminalHandler(output, useColor)
	default:
		// Unknown log format specified
		return fmt.Errorf("unknown log format: %v", ctx.GlobalString(logFormatFlag.Name))
	}

	glogger = log.NewGlogHandler(handler)

	// logging
	verbosity := log.FromLegacyLevel(ctx.GlobalInt(verbosityFlag.Name))
	glogger.Verbosity(verbosity)
	vmodule := ctx.GlobalString(logVmoduleFlag.Name)
	if vmodule == "" {
		// Retain backwards compatibility with `--vmodule` flag if `--log-vmodule` not set
		vmodule = ctx.GlobalString(vmoduleFlag.Name)
		if vmodule != "" {
			defer log.Warn("The flag '--vmodule' is deprecated, please use '--log-vmodule' instead")
		}
	}
	glogger.Vmodule(vmodule)

	log.SetDefault(log.NewLogger(glogger))

	// profiling, tracing
	runtime.MemProfileRate = ctx.GlobalInt(memprofilerateFlag.Name)
	Handler.SetBlockProfileRate(ctx.GlobalInt(blockprofilerateFlag.Name))
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
	Handler.filePath = ctx.GlobalString(debugDataDirFlag.Name)

	if periodicProfiling := ctx.GlobalBool(periodicProfilingFlag.Name); periodicProfiling {
		Handler.PeriodicComputeProfiling()
	}

	// pprof server
	if ctx.GlobalBool(pprofFlag.Name) {
		// Hook go-metrics into expvar on any /debug/metrics request, load all vars
		// from the registry into expvar, and execute regular expvar handler.
		exp.Exp(metrics.DefaultRegistry)

		address := fmt.Sprintf("%s:%d", ctx.GlobalString(pprofAddrFlag.Name), ctx.GlobalInt(pprofPortFlag.Name))
		go func() {
			log.Info("Starting pprof server", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
			if err := http.ListenAndServe(address, nil); err != nil {
				log.Error("Failure in running pprof server", "err", err)
			}
		}()
	}

	if len(logFile) > 0 || rotation {
		log.Info("Logging configured", context...)
	}

	return nil
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	Handler.StopCPUProfile()
	Handler.StopGoTrace()
	if logOutputFile != nil {
		logOutputFile.Close()
	}
}

func validateLogLocation(path string) error {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return fmt.Errorf("error creating the directory: %w", err)
	}
	// Check if the path is writable by trying to create a temporary file
	tmp := filepath.Join(path, "tmp")
	if f, err := os.Create(tmp); err != nil {
		return err
	} else {
		f.Close()
	}
	return os.Remove(tmp)
}
