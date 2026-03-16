// Copyright 2026 The go-ethereum Authors
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
	"strings"

	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/grafana/pyroscope-go"
	"github.com/urfave/cli/v2"
)

var (
	pyroscopeFlag = &cli.BoolFlag{
		Name:     "pyroscope",
		Usage:    "Enable Pyroscope profiling",
		Value:    false,
		Category: flags.LoggingCategory,
	}
	pyroscopeServerFlag = &cli.StringFlag{
		Name:     "pyroscope.server",
		Usage:    "Pyroscope server URL to push profiling data to",
		Value:    "http://localhost:4040",
		Category: flags.LoggingCategory,
	}
	pyroscopeAuthUsernameFlag = &cli.StringFlag{
		Name:     "pyroscope.username",
		Usage:    "Pyroscope basic authentication username",
		Value:    "",
		Category: flags.LoggingCategory,
	}
	pyroscopeAuthPasswordFlag = &cli.StringFlag{
		Name:     "pyroscope.password",
		Usage:    "Pyroscope basic authentication password",
		Value:    "",
		Category: flags.LoggingCategory,
	}
	pyroscopeTagsFlag = &cli.StringFlag{
		Name:     "pyroscope.tags",
		Usage:    "Comma separated list of key=value tags to add to profiling data",
		Value:    "",
		Category: flags.LoggingCategory,
	}
)

// This holds the globally-configured Pyroscope instance.
var pyroscopeProfiler *pyroscope.Profiler

func startPyroscope(ctx *cli.Context) error {
	server := ctx.String(pyroscopeServerFlag.Name)
	authUsername := ctx.String(pyroscopeAuthUsernameFlag.Name)
	authPassword := ctx.String(pyroscopeAuthPasswordFlag.Name)

	rawTags := ctx.String(pyroscopeTagsFlag.Name)
	tags := make(map[string]string)
	for tag := range strings.SplitSeq(rawTags, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		k, v, _ := strings.Cut(tag, "=")
		tags[k] = v
	}

	config := pyroscope.Config{
		ApplicationName:   "geth",
		ServerAddress:     server,
		BasicAuthUser:     authUsername,
		BasicAuthPassword: authPassword,
		Logger:            &pyroscopeLogger{Logger: log.Root()},
		Tags:              tags,
		ProfileTypes: []pyroscope.ProfileType{
			// Enabling all profile types
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}

	profiler, err := pyroscope.Start(config)
	if err != nil {
		return err
	}
	pyroscopeProfiler = profiler
	log.Info("Enabled Pyroscope")
	return nil
}

func stopPyroscope() {
	if pyroscopeProfiler != nil {
		pyroscopeProfiler.Stop()
		pyroscopeProfiler = nil
	}
}

// Small wrapper for log.Logger to satisfy pyroscope.Logger interface
type pyroscopeLogger struct {
	Logger log.Logger
}

func (l *pyroscopeLogger) Infof(format string, v ...any) {
	l.Logger.Info(fmt.Sprintf("Pyroscope: "+format, v...))
}

func (l *pyroscopeLogger) Debugf(format string, v ...any) {
	l.Logger.Debug(fmt.Sprintf("Pyroscope: "+format, v...))
}

func (l *pyroscopeLogger) Errorf(format string, v ...any) {
	l.Logger.Error(fmt.Sprintf("Pyroscope: "+format, v...))
}
