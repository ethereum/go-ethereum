// Copyright 2025 The go-ethereum Authors
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

//go:build !tinygo
// +build !tinygo

package rawdb

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Lightweight internal table writer used instead of the external dependency.
// It provides the small subset of the tablewriter API that the project uses.
type Table struct {
	out     io.Writer
	headers []string
	footer  []string
	rows    [][]string
}

func newTableWriter(w io.Writer) *Table {
	return &Table{out: w}
}

func (t *Table) SetHeader(headers []string) {
	t.headers = headers
}

func (t *Table) SetFooter(footer []string) {
	t.footer = footer
}

func (t *Table) AppendBulk(rows [][]string) {
	t.rows = rows
}

// Render prints a simple tab-separated table using text/tabwriter. This
// intentionally prints a compact, readable table for CLI consumption and
// fulfils the small API surface used in the codebase.
func (t *Table) Render() {
	w := tabwriter.NewWriter(t.out, 0, 0, 2, ' ', 0)
	if len(t.headers) > 0 {
		fmt.Fprintln(w, strings.Join(t.headers, "\t"))
	}
	for _, row := range t.rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	if len(t.footer) > 0 {
		fmt.Fprintln(w, strings.Join(t.footer, "\t"))
	}
	_ = w.Flush()
}
