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

// TODO: naive stub implementation for tablewriter

//go:build tinygo
// +build tinygo

package rawdb

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	cellPadding  = 1               // Number of spaces on each side of cell content
	totalPadding = 2 * cellPadding // Total padding per cell. Its two because we pad equally on both sides
)

type Table struct {
	out     io.Writer
	headers []string
	footer  []string
	rows    [][]string
}

func newTableWriter(w io.Writer) *Table {
	return &Table{out: w}
}

// SetHeader sets the header row for the table. Headers define the column names
// and determine the number of columns for the entire table.
//
// All data rows and footer must have the same number of columns as the headers.
//
// Note: Headers are required - tables without headers will fail validation.
func (t *Table) SetHeader(headers []string) {
	t.headers = headers
}

// SetFooter sets an optional footer row for the table.
//
// The footer must have the same number of columns as the headers, or validation will fail.
func (t *Table) SetFooter(footer []string) {
	t.footer = footer
}

// AppendBulk sets all data rows for the table at once, replacing any existing rows.
//
// Each row must have the same number of columns as the headers, or validation
// will fail during Render().
func (t *Table) AppendBulk(rows [][]string) {
	t.rows = rows
}

// Render outputs the complete table to the configured writer. The table is rendered
// with headers, data rows, and optional footer.
//
// If validation fails, an error message is written to the output and rendering stops.
func (t *Table) Render() {
	if err := t.render(); err != nil {
		fmt.Fprintf(t.out, "Error: %v\n", err)
		return
	}
}

func (t *Table) render() error {
	if err := t.validateColumnCount(); err != nil {
		return err
	}

	widths := t.calculateColumnWidths()
	rowSeparator := t.buildRowSeparator(widths)

	if len(t.headers) > 0 {
		t.printRow(t.headers, widths)
		fmt.Fprintln(t.out, rowSeparator)
	}

	for _, row := range t.rows {
		t.printRow(row, widths)
	}

	if len(t.footer) > 0 {
		fmt.Fprintln(t.out, rowSeparator)
		t.printRow(t.footer, widths)
	}

	return nil
}

// validateColumnCount checks that all rows and footer match the header column count
func (t *Table) validateColumnCount() error {
	if len(t.headers) == 0 {
		return errors.New("table must have headers")
	}

	expectedCols := len(t.headers)

	// Check all rows have same column count as headers
	for i, row := range t.rows {
		if len(row) != expectedCols {
			return fmt.Errorf("row %d has %d columns, expected %d", i, len(row), expectedCols)
		}
	}

	// Check footer has same column count as headers (if present)
	footerPresent := len(t.footer) > 0
	if footerPresent && len(t.footer) != expectedCols {
		return fmt.Errorf("footer has %d columns, expected %d", len(t.footer), expectedCols)
	}

	return nil
}

// calculateColumnWidths determines the minimum width needed for each column.
//
// This is done by finding the longest content in each column across headers, rows, and footer.
//
// Returns an int slice where widths[i] is the display width for column i (including padding).
func (t *Table) calculateColumnWidths() []int {
	// Headers define the number of columns
	cols := len(t.headers)
	if cols == 0 {
		return nil
	}

	// Track maximum content width for each column (before padding)
	widths := make([]int, cols)

	// Start with header widths
	for i, h := range t.headers {
		widths[i] = len(h)
	}

	// Find max width needed for data cells in each column
	for _, row := range t.rows {
		for i, cell := range row {
			widths[i] = max(widths[i], len(cell))
		}
	}

	// Find max width needed for footer in each column
	for i, f := range t.footer {
		widths[i] = max(widths[i], len(f))
	}

	for i := range widths {
		widths[i] += totalPadding
	}

	return widths
}

// buildRowSeparator creates a horizontal line to separate table rows.
//
// It generates a string with dashes (-) for each column width, joined by plus signs (+).
//
// Example output: "----------+--------+-----------"
func (t *Table) buildRowSeparator(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = strings.Repeat("-", w)
	}
	return strings.Join(parts, "+")
}

// printRow outputs a single row to the table writer.
//
// Each cell is padded with spaces and separated by pipe characters (|).
//
// Example output: " Database |  Size  |  Items  "
func (t *Table) printRow(row []string, widths []int) {
	for i, cell := range row {
		if i > 0 {
			fmt.Fprint(t.out, "|")
		}

		// Calculate centering pad without padding
		contentWidth := widths[i] - totalPadding
		cellLen := len(cell)
		leftPadCentering := (contentWidth - cellLen) / 2
		rightPadCentering := contentWidth - cellLen - leftPadCentering

		// Build padded cell with centering
		leftPadding := strings.Repeat(" ", cellPadding+leftPadCentering)
		rightPadding := strings.Repeat(" ", cellPadding+rightPadCentering)

		fmt.Fprintf(t.out, "%s%s%s", leftPadding, cell, rightPadding)
	}
	fmt.Fprintln(t.out)
}
