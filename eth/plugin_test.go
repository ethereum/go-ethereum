package eth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/internal/build"
)

func Test_loadRPCPlugins(t *testing.T) {
	tc := build.GoToolchain{}
	cmd := tc.Go("build", "-buildmode=plugin", "-o", "testdata/service.so", "testdata/service_mock.go")
	build.MustRun(cmd)

	defer func() {
		_ = os.Remove("testdata/service.so")
	}()

	path, err := filepath.Abs("testdata/service.so")
	if err != nil {
		t.Fatal(err)
	}

	plugins, err := loadRPCPlugins([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if plugins[0].Public != true {
		t.Fatal("public was not true")
	}

	if plugins[0].Namespace != "test" {
		t.Fatalf("expected %s, actual %s", plugins[0].Namespace, "tset")
	}

	if plugins[0].Version != "1.0.0" {
		t.Fatalf("expected %s, actual %s", plugins[0].Version, "1.0.0")
	}
}
