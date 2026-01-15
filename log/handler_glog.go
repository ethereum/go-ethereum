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

	level    atomic.Int32 // Current log level, atomically accessible
	override atomic.Bool  // Flag whether overrides are used, atomically accessible

	patterns []pattern                // Current list of patterns to override with
	cachePtr atomic.Pointer[sync.Map] // Pointer to cache of callsite pattern evaluations, maps uintptr -> slog.Level
	location string                   // file:line location where to do a stackdump at
	lock     sync.Mutex               // Lock protecting the override pattern list
}

// NewGlogHandler creates a new log handler with filtering functionality similar
// to Google's glog logger. The returned handler implements Handler.
func NewGlogHandler(h slog.Handler) *GlogHandler {
	g := &GlogHandler{origin: h}
	m := new(sync.Map)
	g.cachePtr.Store(m)
	return g
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
	h.cachePtr.Store(new(sync.Map)) // atomic swap of cache instead of Clear
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
	h.patterns = filter
	h.lock.Unlock()
	h.cachePtr.Store(new(sync.Map)) // atomic swap of cache instead of Clear
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
	patterns := []pattern{}
	h.lock.Lock()
	patterns = append(patterns, h.patterns...)
	h.lock.Unlock()

	res := GlogHandler{
		origin:   h.origin.WithAttrs(attrs),
		patterns: patterns,
		location: h.location,
	}

	res.level.Store(h.level.Load())
	res.override.Store(h.override.Load())
	res.cachePtr.Store(h.cachePtr.Load())
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
func (h *GlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Get current cache
	cache := h.cachePtr.Load()

	// Fast path: cache hit
	if lvl, ok := cache.Load(r.PC); ok {
		if lvl.(slog.Level) <= r.Level {
			return h.origin.Handle(ctx, r)
		}
		return nil
	}

	// Resolve the callsite file once.
	fs := runtime.CallersFrames([]uintptr{r.PC})
	frame, _ := fs.Next()
	file := frame.File

	// Snapshot the current pattern slice under lock.
	h.lock.Lock()
	curPatterns := h.patterns
	h.lock.Unlock()

	// Match without holding the lock.
	var (
		lvl slog.Level
		ok  bool
	)
	for _, rule := range curPatterns {
		if rule.pattern.MatchString("+" + file) {
			lvl, ok = rule.level, true
			// TODO: Not breaking allows the last match to win. Is this what we want?
		}
	}
	if !ok {
		// No rule matched: use the current global/default level.
		lvl = slog.Level(h.level.Load())
	}

	// Check if we should cache this result
	h.lock.Lock()
	shouldCache := false
	switch {
	case len(curPatterns) == 0 && len(h.patterns) == 0:
		// Cache the default/global level to avoid re-evaluating the callsite each time.
		shouldCache = true
	case len(h.patterns) > 0 && len(curPatterns) > 0:
		// Only cache the result if the vmodule patterns have not changed since we
		// snapshotted them. This avoids inserting stale entries if Vmodule() updates
		// the pattern list concurrently with Handle().
		shouldCache = (&h.patterns[0] == &curPatterns[0])
	}
	if shouldCache {
		cache.Store(r.PC, lvl)
	}
	h.lock.Unlock()

	if lvl <= r.Level {
		return h.origin.Handle(ctx, r)
	}
	return nil
}
