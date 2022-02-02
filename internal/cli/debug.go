package cli

// Based on https://github.com/hashicorp/nomad/blob/main/command/operator_debug.go

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server/proto"
	"github.com/golang/protobuf/jsonpb"
	gproto "github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	grpc_net_conn "github.com/mitchellh/go-grpc-net-conn"
)

type DebugCommand struct {
	*Meta2

	seconds uint64
	output  string
}

// Help implements the cli.Command interface
func (d *DebugCommand) Help() string {
	return `Usage: bor debug

  Build an archive containing Bor pprof traces

  ` + d.Flags().Help()
}

func (d *DebugCommand) Flags() *flagset.Flagset {
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
func (d *DebugCommand) Synopsis() string {
	return "Build an archive containing Bor pprof traces"
}

// Run implements the cli.Command interface
func (d *DebugCommand) Run(args []string) int {
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

	stamped := "bor-debug-" + time.Now().UTC().Format("2006-01-02-150405Z")

	// Create the output directory
	var tmp string
	if d.output != "" {
		// User specified output directory
		tmp = filepath.Join(d.output, stamped)
		_, err := os.Stat(tmp)
		if !os.IsNotExist(err) {
			d.UI.Error("Output directory already exists")
			return 1
		}
	} else {
		// Generate temp directory
		tmp, err = ioutil.TempDir(os.TempDir(), stamped)
		if err != nil {
			d.UI.Error(fmt.Sprintf("Error creating tmp directory: %s", err.Error()))
			return 1
		}
		defer os.RemoveAll(tmp)
	}

	d.UI.Output("Starting debugger...")
	d.UI.Output("")

	// ensure destine folder exists
	if err := os.MkdirAll(tmp, os.ModePerm); err != nil {
		d.UI.Error(fmt.Sprintf("failed to create parent directory: %v", err))
		return 1
	}

	pprofProfile := func(ctx context.Context, profile string, filename string) error {
		req := &proto.PprofRequest{
			Seconds: int64(d.seconds),
		}
		switch profile {
		case "cpu":
			req.Type = proto.PprofRequest_CPU
		case "trace":
			req.Type = proto.PprofRequest_TRACE
		default:
			req.Type = proto.PprofRequest_LOOKUP
			req.Profile = profile
		}
		stream, err := clt.Pprof(ctx, req)
		if err != nil {
			return err
		}
		// wait for open request
		msg, err := stream.Recv()
		if err != nil {
			return err
		}
		if _, ok := msg.Event.(*proto.PprofResponse_Open_); !ok {
			return fmt.Errorf("expected open message")
		}

		// create the stream
		conn := &grpc_net_conn.Conn{
			Stream:   stream,
			Response: &proto.PprofResponse_Input{},
			Decode: grpc_net_conn.SimpleDecoder(func(msg gproto.Message) *[]byte {
				return &msg.(*proto.PprofResponse_Input).Data
			}),
		}

		file, err := os.OpenFile(filepath.Join(tmp, filename+".prof"), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(file, conn); err != nil {
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
		m := jsonpb.Marshaler{}
		data, err := m.MarshalToString(statusResp)
		if err != nil {
			d.UI.Output(err.Error())
			return 1
		}
		if err := ioutil.WriteFile(filepath.Join(tmp, "status.json"), []byte(data), 0644); err != nil {
			d.UI.Output(fmt.Sprintf("Failed to write status: %v", err))
			return 1
		}
	}

	// Exit before archive if output directory was specified
	if d.output != "" {
		d.UI.Output(fmt.Sprintf("Created debug directory: %s", tmp))
		return 0
	}

	// Create archive tarball
	archiveFile := stamped + ".tar.gz"
	if err = tarCZF(archiveFile, tmp, stamped); err != nil {
		d.UI.Error(fmt.Sprintf("Error creating archive: %s", err.Error()))
		return 1
	}

	d.UI.Output(fmt.Sprintf("Created debug archive: %s", archiveFile))
	return 0
}

func trapSignal(cancel func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		<-sigCh
		cancel()
	}()
}

func tarCZF(archive string, src, target string) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files - %v", err.Error())
	}

	// create the archive
	fh, err := os.Create(archive)
	if err != nil {
		return err
	}
	defer fh.Close()

	zz := gzip.NewWriter(fh)
	defer zz.Close()

	tw := tar.NewWriter(zz)
	defer tw.Close()

	// tar
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// remove leading path to the src, so files are relative to the archive
		path := strings.ReplaceAll(file, src, "")
		if target != "" {
			path = filepath.Join([]string{target, path}...)
		}
		path = strings.TrimPrefix(path, string(filepath.Separator))

		header.Name = path

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// copy the file contents
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		f.Close()
		return nil
	})
}
