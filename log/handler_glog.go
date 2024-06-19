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
	"fmt"
	"log/slog"
	"maps"
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

	level    atomic.Int32 // Current log level, atomically accessible
	override atomic.Bool  // Flag whether overrides are used, atomically accessible

	patterns  []pattern              // Current list of patterns to override with
	siteCache map[uintptr]slog.Level // Cache of callsite pattern evaluations
	location  string                 // file:line location where to do a stackdump at
	lock      sync.RWMutex           // Lock protecting the override pattern list
}

// NewGlogHandler creates a new log handler with filtering functionality similar
// to Google's glog logger. The returned handler implements Handler.
func NewGlogHandler(h slog.Handler) *GlogHandler {
	return &GlogHandler{
		origin: h,
	}
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
	h.level.Store(int32(level))
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
	defer h.lock.Unlock()

	h.patterns = filter
	h.siteCache = make(map[uintptr]slog.Level)
	h.override.Store(len(filter) != 0)

	return nil
}

// Enabled implements slog.Handler, reporting whether the handler handles records
// at the given level.
func (h *GlogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	// fast-track skipping logging if override not enabled and the provided verbosity is above configured
	return h.override.Load() || slog.Level(h.level.Load()) <= lvl
}

// WithAttrs implements slog.Handler, returning a new Handler whose attributes
// consist of both the receiver's attributes and the arguments.
func (h *GlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h.lock.RLock()
	siteCache := maps.Clone(h.siteCache)
	h.lock.RUnlock()

	patterns := []pattern{}
	patterns = append(patterns, h.patterns...)

	res := GlogHandler{
		origin:    h.origin.WithAttrs(attrs),
		patterns:  patterns,
		siteCache: siteCache,
		location:  h.location,
	}

	res.level.Store(h.level.Load())
	res.override.Store(h.override.Load())
	return &res
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
func (h *GlogHandler) Handle(_ context.Context, r slog.Record) error {
	// If the global log level allows, fast track logging
	if slog.Level(h.level.Load()) <= r.Level {
		return h.origin.Handle(context.Background(), r)
	}

	// Check callsite cache for previously calculated log levels
	h.lock.RLock()
	lvl, ok := h.siteCache[r.PC]
	h.lock.RUnlock()

	// If we didn't cache the callsite yet, calculate it
	if !ok {
		h.lock.Lock()

		fs := runtime.CallersFrames([]uintptr{r.PC})
		frame, _ := fs.Next()

		for _, rule := range h.patterns {
			if rule.pattern.MatchString(fmt.Sprintf("+%s", frame.File)) {
				h.siteCache[r.PC], lvl, ok = rule.level, rule.level, true
			}
		}
		// If no rule matched, remember to drop log the next time
		if !ok {
			h.siteCache[r.PC] = 0
		}
		h.lock.Unlock()
	}
	if lvl <= r.Level {
		return h.origin.Handle(context.Background(), r)
	}
	return nil
}
