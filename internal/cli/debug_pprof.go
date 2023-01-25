package cli

// Based on https://github.com/hashicorp/nomad/blob/main/command/operator_debug.go

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	empty "google.golang.org/protobuf/types/known/emptypb"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
)

type DebugPprofCommand struct {
	*Meta2

	seconds uint64
	output  string
}

func (p *DebugPprofCommand) MarkDown() string {
	items := []string{
		"# Debug Pprof",
		"The ```debug pprof <enode>``` command will create an archive containing bor pprof traces.",
		p.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (d *DebugPprofCommand) Help() string {
	return `Usage: bor debug

  Build an archive containing Bor pprof traces

  ` + d.Flags().Help()
}

func (d *DebugPprofCommand) Flags() *flagset.Flagset {
	flags := d.NewFlagSet("debug")

	flags.Uint64Flag(&flagset.Uint64Flag{
		Name:    "seconds",
		Usage:   "seconds to trace",
		Value:   &d.seconds,
		Default: 2,
	})
	flags.StringFlag(&flagset.StringFlag{
		Name:  "output",
		Value: &d.output,
		Usage: "Output directory",
	})

	return flags
}

// Synopsis implements the cli.Command interface
func (d *DebugPprofCommand) Synopsis() string {
	return "Build an archive containing Bor pprof traces"
}

// Run implements the cli.Command interface
func (d *DebugPprofCommand) Run(args []string) int {
	flags := d.Flags()
	if err := flags.Parse(args); err != nil {
		d.UI.Error(err.Error())
		return 1
	}

	clt, err := d.BorConn()
	if err != nil {
		d.UI.Error(err.Error())
		return 1
	}

	dEnv := &debugEnv{
		output: d.output,
		prefix: "bor-debug-",
	}
	if err := dEnv.init(); err != nil {
		d.UI.Error(err.Error())
		return 1
	}

	d.UI.Output("Starting debugger...")
	d.UI.Output("")

	pprofProfile := func(ctx context.Context, profile string, filename string) error {
		req := &proto.DebugPprofRequest{
			Seconds: int64(d.seconds),
		}

		switch profile {
		case "cpu":
			req.Type = proto.DebugPprofRequest_CPU
		case "trace":
			req.Type = proto.DebugPprofRequest_TRACE
		default:
			req.Type = proto.DebugPprofRequest_LOOKUP
			req.Profile = profile
		}

		stream, err := clt.DebugPprof(ctx, req, grpc.MaxCallRecvMsgSize(1024*1024*1024))

		if err != nil {
			return err
		}

		if err := dEnv.writeFromStream(filename+".prof", stream); err != nil {
			return err
		}

		return nil
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	trapSignal(cancelFn)

	profiles := map[string]string{
		"heap":  "heap",
		"cpu":   "cpu",
		"trace": "trace",
	}
	for profile, filename := range profiles {
		if err := pprofProfile(ctx, profile, filename); err != nil {
			d.UI.Error(fmt.Sprintf("Error creating profile '%s': %v", profile, err))
			return 1
		}
	}

	// append the status
	{
		statusResp, err := clt.Status(ctx, &empty.Empty{})
		if err != nil {
			d.UI.Output(fmt.Sprintf("Failed to get status: %v", err))
			return 1
		}
		if err := dEnv.writeJSON("status.json", statusResp); err != nil {
			d.UI.Error(err.Error())
			return 1
		}
	}

	if err := dEnv.finish(); err != nil {
		d.UI.Error(err.Error())
		return 1
	}

	if d.output != "" {
		d.UI.Output(fmt.Sprintf("Created debug directory: %s", dEnv.dst))
	} else {
		d.UI.Output(fmt.Sprintf("Created debug archive: %s", dEnv.tarName()))
	}

	return 0
}
