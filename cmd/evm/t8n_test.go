package main

import (
	"fmt"
	"github.com/docker/docker/pkg/reexec"
	"github.com/ethereum/go-ethereum/internal/cmdtest"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	// Run the app if we've been exec'd as "ethkey-test" in runEthkey.
	reexec.Register("evm-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}

type testT8n struct {
	*cmdtest.TestCmd
}

type t8nArgs struct {
	inAlloc  string
	inTxs    string
	inEnv    string
	stFork   string
	stReward string
}

func (args *t8nArgs) get(base string) []string {
	var out []string
	if opt := args.inAlloc; opt != "" {
		out = append(out, "--input.alloc")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inTxs; opt != "" {
		out = append(out, "--input.txs")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.inEnv; opt != "" {
		out = append(out, "--input.env")
		out = append(out, fmt.Sprintf("%v/%v", base, opt))
	}
	if opt := args.stFork; opt != "" {
		out = append(out, "--state.fork", opt)
	}
	if opt := args.stReward; opt != "" {
		out = append(out, "--state.reward", opt)
	}
	return out
}

func TestT8n(t *testing.T) {
	tt := new(testT8n)
	tt.TestCmd = cmdtest.NewTestCmd(t, tt)
	for i, tc := range []struct {
		base        string
		args        t8nArgs
		expExitCode int
		expOut      string
	}{
		{ // Test exit (3) on bad config
			base: "./testdata/1",
			args: t8nArgs{
				"alloc.json", "txs.json", "env.json", "Frontier+1346", "",
			},
			expExitCode: 3,
		},
		{
			base: "./testdata/1",
			args: t8nArgs{
				"alloc.json", "txs.json", "env.json", "Byzantium", "",
			},
			expOut: "exp.json",
		},
		{// blockhash test
			base: "./testdata/3",
			args: t8nArgs{
				"alloc.json", "txs.json", "env.json", "Berlin", "",
			},
			expOut: "exp.json",
		},
		{// missing blockhash test
			base: "./testdata/4",
			args: t8nArgs{
				"alloc.json", "txs.json", "env.json", "Berlin", "",
			},
			expExitCode: 4,
		},
		{ // Ommer test
			base: "./testdata/5",
			args: t8nArgs{
				"alloc.json", "txs.json", "env.json", "Byzantium", "0x80",
			},
			expOut: "exp.json",
		},
	} {
		args := append([]string{"t8n",
			"--output.result", "stdout",
			"--output.alloc", "stdout"}, tc.args.get(tc.base)...)
		tt.Run("evm-test", args...)
		fmt.Printf("args: %v\n", strings.Join(args, " "))
		// Compare the expected output, if provided
		if tc.expOut != "" {
			want, err := os.ReadFile(fmt.Sprintf("%v/%v", tc.base, tc.expOut))
			if err != nil {
				t.Fatalf("test %d: could not read expected output: %v", i, err)
			}
			tt.Expect(string(want))
		}
		tt.WaitExit()
		if have, want := tt.ExitStatus(), tc.expExitCode; have != want {
			t.Fatalf("test %d: wrong exit code, have %d, want %d", i, have, want)
		}

	}

	//./evm t8n --input.alloc=./testdata/1/alloc.json --input.txs=./testdata/1/txs.json --input.env=./testdata/1/env.json

}
