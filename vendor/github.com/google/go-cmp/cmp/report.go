// Copyright 2017, The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package cmp

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/go-cmp/cmp/internal/value"
)

type defaultReporter struct {
	curPath Path

	diffs  []string // List of differences, possibly truncated
	ndiffs int      // Total number of differences
	nbytes int      // Number of bytes in diffs
	nlines int      // Number of lines in diffs
}

func (r *defaultReporter) PushStep(ps PathStep) {
	r.curPath.push(ps)
}
func (r *defaultReporter) Report(f reportFlags) {
	if f&reportUnequal > 0 {
		vx, vy := r.curPath.Last().Values()
		r.report(vx, vy, r.curPath)
	}
}
func (r *defaultReporter) PopStep() {
	r.curPath.pop()
}

func (r *defaultReporter) report(x, y reflect.Value, p Path) {
	const maxBytes = 4096
	const maxLines = 256
	r.ndiffs++
	if r.nbytes < maxBytes && r.nlines < maxLines {
		sx := value.Format(x, value.FormatConfig{UseStringer: true})
		sy := value.Format(y, value.FormatConfig{UseStringer: true})
		if sx == sy {
			// Unhelpful output, so use more exact formatting.
			sx = value.Format(x, value.FormatConfig{PrintPrimitiveType: true})
			sy = value.Format(y, value.FormatConfig{PrintPrimitiveType: true})
		}
		s := fmt.Sprintf("%#v:\n\t-: %s\n\t+: %s\n", p, sx, sy)
		r.diffs = append(r.diffs, s)
		r.nbytes += len(s)
		r.nlines += strings.Count(s, "\n")
	}
}

func (r *defaultReporter) String() string {
	s := strings.Join(r.diffs, "")
	if r.ndiffs == len(r.diffs) {
		return s
	}
	return fmt.Sprintf("%s... %d more differences ...", s, r.ndiffs-len(r.diffs))
}
