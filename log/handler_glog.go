// Copyright 2017 The go-ethereum Authors
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

package log

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// errVmoduleSyntax is returned when a user vmodule pattern is invalid.
var errVmoduleSyntax = errors.New("expect comma-separated list of filename=N")

// GlogHandler is a log handler that mimics the filtering features of Google's
// glog logger: setting global log levels; overriding with callsite pattern
// matches; and requesting backtraces at certain positions.
type GlogHandler struct {
	origin slog.Handler // The origin handler this wraps
	lock   sync.Mutex   // synchronizes writes to config
	config atomic.Pointer[glogConfig]
}

type glogConfig struct {
	patterns []pattern
	cache    sync.Map
	level    slog.Level
}

// NewGlogHandler creates a new log handler with filtering functionality similar
// to Google's glog logger. The returned handler implements Handler.
func NewGlogHandler(origin slog.Handler) *GlogHandler {
	h := &GlogHandler{origin: origin}
	h.config.Store(new(glogConfig))
	return h
}

// pattern contains a filter for the Vmodule option, holding a verbosity level
// and a file pattern to match.
type pattern struct {
	pattern *regexp.Regexp
	level   slog.Level
}

// Verbosity sets the glog verbosity ceiling. The verbosity of individual packages
// and source files can be raised using Vmodule.
func (h *GlogHandler) Verbosity(level slog.Level) {
	h.lock.Lock()
	defer h.lock.Unlock()

	cfg := h.config.Load()
	newcfg := &glogConfig{level: level, patterns: cfg.patterns}
	h.config.Store(newcfg)
}

// Vmodule sets the glog verbosity pattern.
//
// The syntax of the argument is a comma-separated list of pattern=N, where the
// pattern is a literal file name or "glob" pattern matching and N is a V level.
//
// For instance:
//
//	pattern="gopher.go=3"
//	 sets the V level to 3 in all Go files named "gopher.go"
//
//	pattern="foo=3"
//	 sets V to 3 in all files of any packages whose import path ends in "foo"
//
//	pattern="foo/*=3"
//	 sets V to 3 in all files of any packages whose import path contains "foo"
func (h *GlogHandler) Vmodule(ruleset string) error {
	var filter []pattern
	for _, rule := range strings.Split(ruleset, ",") {
		// Empty strings such as from a trailing comma can be ignored
		if len(rule) == 0 {
			continue
		}
		// Ensure we have a pattern = level filter rule
		parts := strings.Split(rule, "=")
		if len(parts) != 2 {
			return errVmoduleSyntax
		}
		parts[0] = strings.TrimSpace(parts[0])
		parts[1] = strings.TrimSpace(parts[1])
		if len(parts[0]) == 0 || len(parts[1]) == 0 {
			return errVmoduleSyntax
		}
		// Parse the level and if correct, assemble the filter rule
		l, err := strconv.Atoi(parts[1])
		if err != nil {
			return errVmoduleSyntax
		}
		level := FromLegacyLevel(l)

		if level == LevelCrit {
			continue // Ignore. It's harmless but no point in paying the overhead.
		}
		// Compile the rule pattern into a regular expression
		matcher := ".*"
		for _, comp := range strings.Split(parts[0], "/") {
			if comp == "*" {
				matcher += "(/.*)?"
			} else if comp != "" {
				matcher += "/" + regexp.QuoteMeta(comp)
			}
		}
		if !strings.HasSuffix(parts[0], ".go") {
			matcher += "/[^/]+\\.go"
		}
		matcher = matcher + "$"

		re, _ := regexp.Compile(matcher)
		filter = append(filter, pattern{re, level})
	}

	// Swap out the vmodule pattern for the new filter system
	h.lock.Lock()
	cfg := h.config.Load()
	newcfg := &glogConfig{level: cfg.level, patterns: filter}
	h.config.Store(newcfg)
	h.lock.Unlock()

	return nil
}

// Enabled implements slog.Handler, reporting whether the handler handles records
// at the given level.
func (h *GlogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	// fast-track skipping logging if vmodule is not enabled or level too low
	cfg := h.config.Load()
	return len(cfg.patterns) > 0 || cfg.level <= lvl
}

// WithGroup implements slog.Handler, returning a new Handler with the given
// group appended to the receiver's existing groups.
//
// Note, this function is not implemented.
func (h *GlogHandler) WithGroup(name string) slog.Handler {
	panic("not implemented")
}

// Handle implements slog.Handler, filtering a log record through the global,
// local and backtrace filters, finally emitting it if either allow it through.
func (h *GlogHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handle(ctx, r, h.origin)
}

func (h *GlogHandler) handle(ctx context.Context, r slog.Record, origin slog.Handler) error {
	cfg := h.config.Load()

	var lvl slog.Level
	cachedLvl, ok := cfg.cache.Load(r.PC)
	if ok {
		// Fast path: cache hit
		lvl = cachedLvl.(slog.Level)
	} else {
		// Resolve the callsite file.
		fs := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := fs.Next()
		file := frame.File
		// Match against patterns and cache the level applied at this callsite.
		lvl = cfg.level // default: use global level
		for _, rule := range cfg.patterns {
			if rule.pattern.MatchString("+" + file) {
				lvl = rule.level
			}
		}
		cfg.cache.Store(r.PC, lvl)
	}

	// Handle the message.
	if lvl <= r.Level {
		return origin.Handle(ctx, r)
	}
	return nil
}

// WithAttrs implements slog.Handler, returning a new Handler whose attributes
// consist of both the receiver's attributes and the arguments.
//
// Note the handler created here will still listen to Verbosity and Vmodule settings
// done on the original handler.
func (h *GlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &glogWithAttrs{base: h, origin: h.origin.WithAttrs(attrs)}
}

type glogWithAttrs struct {
	base   *GlogHandler
	origin slog.Handler
}

func (wh *glogWithAttrs) Enabled(ctx context.Context, lvl slog.Level) bool {
	return wh.base.Enabled(ctx, lvl)
}

func (wh *glogWithAttrs) Handle(ctx context.Context, r slog.Record) error {
	return wh.base.handle(ctx, r, wh.origin)
}

func (wh *glogWithAttrs) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &glogWithAttrs{base: wh.base, origin: wh.origin.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler, returning a new Handler with the given
// group appended to the receiver's existing groups.
//
// Note, this function is not implemented.
func (wh *glogWithAttrs) WithGroup(name string) slog.Handler {
	panic("not implemented")
}
