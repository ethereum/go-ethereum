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

package flags

import (
	"os"
	"testing"

	"github.com/urfave/cli/v2"
)

func testAppFlags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{Name: "port", Aliases: []string{"p"}, Value: 30303},
		&cli.StringFlag{Name: "datadir"},
		&cli.BoolFlag{Name: "http"},
	}
}

func TestDropEnvVarsShadowedByArgs(t *testing.T) {
	flags := testAppFlags()
	AutoEnvVars(flags, "GETH")

	t.Run("separate value", func(t *testing.T) {
		t.Setenv("GETH_PORT", "NOT AN INT")
		t.Setenv("GETH_DATADIR", "/keep")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "--port", "12313"})
		if _, ok := os.LookupEnv("GETH_PORT"); ok {
			t.Fatal("GETH_PORT still set")
		}
		if got := os.Getenv("GETH_DATADIR"); got != "/keep" {
			t.Fatalf("GETH_DATADIR = %q", got)
		}
	})

	t.Run("equals form and alias", func(t *testing.T) {
		t.Setenv("GETH_PORT", "tcp://10.0.0.1:6060")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "-p=12313"})
		if _, ok := os.LookupEnv("GETH_PORT"); ok {
			t.Fatal("GETH_PORT still set")
		}
	})

	t.Run("bool flag does not swallow the next flag", func(t *testing.T) {
		t.Setenv("GETH_PORT", "NOT AN INT")
		t.Setenv("GETH_HTTP", "true")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "--http", "--port", "1"})
		if _, ok := os.LookupEnv("GETH_PORT"); ok {
			t.Fatal("GETH_PORT still set")
		}
		if _, ok := os.LookupEnv("GETH_HTTP"); ok {
			t.Fatal("GETH_HTTP still set")
		}
	})

	t.Run("flag-like value is not a flag", func(t *testing.T) {
		t.Setenv("GETH_PORT", "NOT AN INT")
		t.Setenv("GETH_DATADIR", "/keep")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "--datadir", "--port"})
		if got := os.Getenv("GETH_PORT"); got != "NOT AN INT" {
			t.Fatalf("GETH_PORT = %q", got)
		}
		if _, ok := os.LookupEnv("GETH_DATADIR"); ok {
			t.Fatal("GETH_DATADIR still set")
		}
	})

	t.Run("stop at double dash", func(t *testing.T) {
		t.Setenv("GETH_PORT", "NOT AN INT")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "--", "--port", "1"})
		if got := os.Getenv("GETH_PORT"); got != "NOT AN INT" {
			t.Fatalf("GETH_PORT = %q", got)
		}
	})

	t.Run("unset flag keeps invalid env", func(t *testing.T) {
		t.Setenv("GETH_PORT", "NOT AN INT")
		DropEnvVarsShadowedByArgs(flags, []string{"geth", "--http"})
		if got := os.Getenv("GETH_PORT"); got != "NOT AN INT" {
			t.Fatalf("GETH_PORT = %q", got)
		}
	})
}

func TestIntFlagCLIOverridesInvalidEnv(t *testing.T) {
	t.Setenv("GETH_PORT", "NOT AN INT")
	var got int
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port", Value: 30303},
		},
		Action: func(ctx *cli.Context) error {
			got = ctx.Int("port")
			return nil
		},
	}
	AutoEnvVars(app.Flags, "GETH")
	args := []string{"geth", "--port", "12313"}
	DropEnvVarsShadowedByArgs(app.Flags, args)
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
	if got != 12313 {
		t.Fatalf("port = %d, want 12313", got)
	}
}

func TestIntFlagInvalidEnvStillErrorsWithoutCLI(t *testing.T) {
	t.Setenv("GETH_PORT", "NOT AN INT")
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port", Value: 30303},
		},
		Action: func(ctx *cli.Context) error { return nil },
	}
	AutoEnvVars(app.Flags, "GETH")
	args := []string{"geth"}
	DropEnvVarsShadowedByArgs(app.Flags, args)
	if err := app.Run(args); err == nil {
		t.Fatal("expected env parse error")
	}
}
