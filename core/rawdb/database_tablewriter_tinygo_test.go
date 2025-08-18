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

//go:build tinygo
// +build tinygo

package rawdb

import (
	"bytes"
	"strings"
	"testing"
)

func TestTableWriterTinyGo(t *testing.T) {
	var buf bytes.Buffer
	table := newTableWriter(&buf)

	headers := []string{"Database", "Size", "Items", "Status"}
	rows := [][]string{
		{"chaindata", "2.5 GB", "1,234,567", "Active"},
		{"state", "890 MB", "456,789", "Active"},
		{"ancient", "15.2 GB", "2,345,678", "Readonly"},
		{"logs", "120 MB", "89,012", "Active"},
	}
	footer := []string{"Total", "18.71 GB", "4,125,046", "-"}

	table.SetHeader(headers)
	table.AppendBulk(rows)
	table.SetFooter(footer)
	table.Render()

	output := buf.String()
	t.Logf("Table output using custom stub implementation:\n%s", output)
}

func TestTableWriterValidationErrors(t *testing.T) {
	// Test missing headers
	t.Run("MissingHeaders", func(t *testing.T) {
		var buf bytes.Buffer
		table := newTableWriter(&buf)

		rows := [][]string{{"x", "y", "z"}}

		table.AppendBulk(rows)
		table.Render()

		output := buf.String()
		if !strings.Contains(output, "table must have headers") {
			t.Errorf("Expected error for missing headers, got: %s", output)
		}
	})

	t.Run("NotEnoughRowColumns", func(t *testing.T) {
		var buf bytes.Buffer
		table := newTableWriter(&buf)

		headers := []string{"A", "B", "C"}
		badRows := [][]string{
			{"x", "y"}, // Missing column
		}

		table.SetHeader(headers)
		table.AppendBulk(badRows)
		table.Render()

		output := buf.String()
		if !strings.Contains(output, "row 0 has 2 columns, expected 3") {
			t.Errorf("Expected validation error for row 0, got: %s", output)
		}
	})

	t.Run("TooManyRowColumns", func(t *testing.T) {
		var buf bytes.Buffer
		table := newTableWriter(&buf)

		headers := []string{"A", "B", "C"}
		badRows := [][]string{
			{"p", "q", "r", "s"}, // Extra column
		}

		table.SetHeader(headers)
		table.AppendBulk(badRows)
		table.Render()

		output := buf.String()
		if !strings.Contains(output, "row 0 has 4 columns, expected 3") {
			t.Errorf("Expected validation error for row 0, got: %s", output)
		}
	})

	// Test mismatched footer columns
	t.Run("MismatchedFooterColumns", func(t *testing.T) {
		var buf bytes.Buffer
		table := newTableWriter(&buf)

		headers := []string{"A", "B", "C"}
		rows := [][]string{{"x", "y", "z"}}
		footer := []string{"total", "sum"} // Missing column

		table.SetHeader(headers)
		table.AppendBulk(rows)
		table.SetFooter(footer)
		table.Render()

		output := buf.String()
		if !strings.Contains(output, "footer has 2 columns, expected 3") {
			t.Errorf("Expected validation error for footer, got: %s", output)
		}
	})
}
