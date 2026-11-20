package params

import (
	"bytes"
	"errors"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestChainConfig_LoadForks(t *testing.T) {
	const config = `
GENESIS_FORK_VERSION: 0x00000000

ALTAIR_FORK_VERSION: 0x00000001
ALTAIR_FORK_EPOCH: 1

EIP7928_FORK_VERSION: 0xb0000038
EIP7928_FORK_EPOCH: 18446744073709551615

EIP7XXX_FORK_VERSION: 
EIP7XXX_FORK_EPOCH: 

BLOB_SCHEDULE: []
`
	c := &ChainConfig{}
	err := c.LoadForks([]byte(config))
	if err != nil {
		t.Fatal(err)
	}

	for _, fork := range c.Forks {
		if fork.Name == "GENESIS" && (fork.Epoch != 0) {
			t.Errorf("unexpected genesis fork epoch %d", fork.Epoch)
		}
		if fork.Name == "ALTAIR" && (fork.Epoch != 1 || !bytes.Equal(fork.Version, []byte{0, 0, 0, 1})) {
			t.Errorf("unexpected altair fork epoch %d version %x", fork.Epoch, fork.Version)
		}
	}
}

func TestSaveCheckpointToFile(t *testing.T) {
	var (
		dir  = t.TempDir()
		file = filepath.Join(dir, "checkpoint")
		link = filepath.Join(dir, "link")
		cp1  = common.Hash{1}
		cp2  = common.Hash{2}
	)
	c := &ChainConfig{CheckpointFile: file}
	if saved, err := c.SaveCheckpointToFile(cp1); !saved || err != nil {
		t.Fatalf("first save: saved %v, error %v", saved, err)
	}
	// A hard link keeps the file the first save wrote. It sees the second
	// checkpoint only if the second save rewrites that file in place.
	linked := os.Link(file, link) == nil
	if !linked {
		t.Log("No hard links on this file system: not checking that the file is replaced")
	}
	if saved, err := c.SaveCheckpointToFile(cp2); !saved || err != nil {
		t.Fatalf("second save: saved %v, error %v", saved, err)
	}
	if linked {
		if old, err := os.ReadFile(link); err != nil || string(old) != cp1.Hex() {
			t.Errorf("the second save rewrote the file in place: %q, %v", old, err)
		}
	}
	// The second checkpoint loads, and no temporary file is left behind.
	checkLoad(t, file, cp2)
	checkNoTemp(t, dir)
}

// checkNoTemp checks that no temporary file is left in dir.
func checkNoTemp(t *testing.T, dir string) {
	t.Helper()
	if tmp, _ := filepath.Glob(filepath.Join(dir, ".beacon-checkpoint-*.tmp")); len(tmp) != 0 {
		t.Errorf("temporary files left behind: %v", tmp)
	}
}

// TestSaveCheckpointToFileConcurrent checks that saves at the same time each use
// a temporary file of their own: the file holds one of the checkpoints, whole (each
// one's bytes are all the same, so a mix of two shows).
func TestSaveCheckpointToFileConcurrent(t *testing.T) {
	if runtime.GOOS == "windows" {
		// A file being replaced can't be opened there (a sharing violation). geth
		// saves one checkpoint at a time; this test is for the temporary files.
		t.Skip("concurrent replacement of one file fails on Windows")
	}
	var (
		dir  = t.TempDir()
		file = filepath.Join(dir, "checkpoint")
		c    = &ChainConfig{CheckpointFile: file}
		wg   sync.WaitGroup
	)
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 20 {
				var cp common.Hash
				for k := range cp {
					cp[k] = byte(i*20 + j)
				}
				if _, err := c.SaveCheckpointToFile(cp); err != nil {
					t.Errorf("save: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	loaded := new(ChainConfig)
	if ok, err := loaded.SetCheckpointFile(file); !ok || err != nil {
		t.Fatalf("load: loaded %v, error %v", ok, err)
	}
	cp := loaded.Checkpoint
	if !bytes.Equal(cp[:], bytes.Repeat(cp[:1], len(cp))) || cp[0] >= 160 {
		t.Errorf("not one of the saved checkpoints: %v", cp)
	}
	checkNoTemp(t, dir)
}

// TestSaveCheckpointToFileHeldOpen checks a save while another handle holds the
// file open: replaced where that's possible (Unix), written in place where an open
// file can't be replaced (Windows).
func TestSaveCheckpointToFileHeldOpen(t *testing.T) {
	var (
		dir  = t.TempDir()
		file = filepath.Join(dir, "checkpoint")
		c    = &ChainConfig{CheckpointFile: file}
	)
	if _, err := c.SaveCheckpointToFile(common.Hash{1}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if saved, err := c.SaveCheckpointToFile(common.Hash{2}); !saved || err != nil {
		t.Fatalf("save: saved %v, error %v", saved, err)
	}
	checkLoad(t, file, common.Hash{2})
	checkNoTemp(t, dir)
}

// TestSaveCheckpointToFileNotAFile checks that a save doesn't replace what isn't a
// file (a directory; a socket, as a device or a pipe would be): it fails, and the
// thing stays.
func TestSaveCheckpointToFileNotAFile(t *testing.T) {
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "directory")}
	if err := os.Mkdir(paths[0], 0700); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		sock := filepath.Join(dir, "socket")
		l, err := net.Listen("unix", sock)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()
		paths = append(paths, sock)
	}
	for _, path := range paths {
		before, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		c := &ChainConfig{CheckpointFile: path}
		if saved, err := c.SaveCheckpointToFile(common.Hash{1}); saved || err == nil {
			t.Errorf("save to %s: saved %v, error %v", path, saved, err)
		}
		after, err := os.Lstat(path)
		if err != nil || after.Mode().Type() != before.Mode().Type() {
			t.Errorf("the save replaced %s (%v)", path, err)
		}
	}
	checkNoTemp(t, dir)
}

// TestNotReplaceable checks which errors let a save write the file in place: only
// those that mean it can't be replaced at all.
func TestNotReplaceable(t *testing.T) {
	for _, tt := range []struct {
		err   error
		place bool
	}{
		{&fs.PathError{Op: "open", Path: "x", Err: fs.ErrPermission}, true},
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.EROFS}, true},
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.EBUSY}, true},
		{&os.LinkError{Op: "rename", Old: "a", New: "b", Err: syscall.EXDEV}, true},
		{&fs.PathError{Op: "open", Path: "x", Err: syscall.ENOSPC}, false},
		{&fs.PathError{Op: "open", Path: "x", Err: syscall.ENAMETOOLONG}, false},
		{&fs.PathError{Op: "open", Path: "x", Err: syscall.EIO}, false},
	} {
		if got := errors.Is(notReplaceable(tt.err), errNotReplaceable); got != tt.place {
			t.Errorf("%v: in place %v, want %v", tt.err, got, tt.place)
		}
	}
}

func TestSaveCheckpointToFileSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	dir := t.TempDir()

	// A link to an existing file: the file gets the checkpoint and keeps its
	// permissions.
	var (
		target = filepath.Join(dir, "target")
		link   = filepath.Join(dir, "link")
	)
	if err := os.WriteFile(target, []byte(common.Hash{1}.Hex()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	c := &ChainConfig{CheckpointFile: link}
	if saved, err := c.SaveCheckpointToFile(common.Hash{2}); !saved || err != nil {
		t.Fatalf("save: saved %v, error %v", saved, err)
	}
	checkLink(t, link)
	checkLoad(t, target, common.Hash{2})
	if info, err := os.Stat(target); err != nil || info.Mode().Perm() != 0640 {
		t.Errorf("the save changed the file's permissions (%v)", err)
	}

	// A link to a file that doesn't exist yet, by absolute and by relative path:
	// the save creates the file, through a new file renamed into place, not in
	// place.
	inPlaceWarned.Store(false)
	defer inPlaceWarned.Store(false)
	for _, dest := range []string{filepath.Join(dir, "later"), "later-relative"} {
		link := filepath.Join(dir, "link-"+filepath.Base(dest))
		if err := os.Symlink(dest, link); err != nil {
			t.Fatal(err)
		}
		c := &ChainConfig{CheckpointFile: link}
		if saved, err := c.SaveCheckpointToFile(common.Hash{3}); !saved || err != nil {
			t.Fatalf("save through a link to %s: saved %v, error %v", dest, saved, err)
		}
		checkLink(t, link)
		checkLoad(t, filepath.Join(dir, filepath.Base(dest)), common.Hash{3})
	}
	checkNoTemp(t, dir)

	// A link whose path has ".." after a linked directory: ".." goes up from where
	// that link leads, as the system resolves it, not from the link.
	store := filepath.Join(dir, "store", "sub")
	if err := os.MkdirAll(store, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(store, filepath.Join(dir, "jump")); err != nil {
		t.Fatal(err)
	}
	dots := filepath.Join(dir, "link-dots")
	if err := os.Symlink("jump/../saved", dots); err != nil {
		t.Fatal(err)
	}
	c = &ChainConfig{CheckpointFile: dots}
	if saved, err := c.SaveCheckpointToFile(common.Hash{4}); !saved || err != nil {
		t.Fatalf("save through %s: saved %v, error %v", dots, saved, err)
	}
	checkLink(t, dots)
	checkLoad(t, filepath.Join(dir, "store", "saved"), common.Hash{4})
	checkNoTemp(t, filepath.Join(dir, "store"))
	if _, err := os.Stat(filepath.Join(dir, "saved")); !os.IsNotExist(err) {
		t.Errorf("the save wrote %s, where the link's path leads before following it (%v)", filepath.Join(dir, "saved"), err)
	}
	if inPlaceWarned.Load() {
		t.Error("a save through a link to a file that doesn't exist yet went in place")
	}
}

// TestSaveCheckpointToFileDotDot checks a first save to a path with ".." after a
// linked directory, whose directory as written (before following the link) can't
// take new files: the file is created where the system resolves the path, through
// a new file, not in place.
func TestSaveCheckpointToFileDotDot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symbolic links need privileges on Windows")
	}
	dir := t.TempDir()
	store := filepath.Join(dir, "store")
	if err := os.MkdirAll(filepath.Join(store, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(store, "sub"), filepath.Join(dir, "jump")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	if probe, err := os.CreateTemp(dir, "probe"); err == nil {
		probe.Close()
		os.Remove(probe.Name())
		t.Skip("directory permissions aren't enforced here")
	}
	inPlaceWarned.Store(false)
	defer inPlaceWarned.Store(false)

	// Not filepath.Join: it would take ".." away before the link is followed.
	file := dir + string(filepath.Separator) + "jump" + string(filepath.Separator) + ".." + string(filepath.Separator) + "checkpoint"
	c := &ChainConfig{CheckpointFile: file}
	if saved, err := c.SaveCheckpointToFile(common.Hash{5}); !saved || err != nil {
		t.Fatalf("save: saved %v, error %v", saved, err)
	}
	checkLoad(t, filepath.Join(store, "checkpoint"), common.Hash{5})
	checkNoTemp(t, store)
	if inPlaceWarned.Load() {
		t.Error("the save went in place")
	}
}

// checkLink checks that the save left the symbolic link a link.
func checkLink(t *testing.T, link string) {
	t.Helper()
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the save replaced the symbolic link %s (%v)", link, err)
	}
}

func TestSaveCheckpointToFileInPlace(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a directory the test can't create files in")
	}
	var (
		dir  = t.TempDir()
		file = filepath.Join(dir, "checkpoint")
	)
	if err := os.WriteFile(file, bytes.Repeat([]byte{'x'}, 100), 0600); err != nil {
		t.Fatal(err)
	}
	// The file is writable but its directory isn't: the save writes over it in
	// place, and cuts it to the checkpoint's length.
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)
	if probe, err := os.CreateTemp(dir, "probe"); err == nil {
		probe.Close()
		os.Remove(probe.Name())
		t.Skip("directory permissions aren't enforced here")
	}
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	inPlaceWarned.Store(false)
	defer inPlaceWarned.Store(false)
	c := &ChainConfig{CheckpointFile: file}
	if saved, err := c.SaveCheckpointToFile(common.Hash{1}); !saved || err != nil {
		t.Fatalf("save: saved %v, error %v", saved, err)
	}
	checkLoad(t, file, common.Hash{1})
	// The same file, written in place: where directory permissions aren't
	// enforced (as for root), the save could have replaced it instead.
	after, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !inPlaceWarned.Load() {
		t.Errorf("the save didn't write the file in place (same file %v, warned %v)", os.SameFile(before, after), inPlaceWarned.Load())
	}
}

func TestSetCheckpointFileEmpty(t *testing.T) {
	file := filepath.Join(t.TempDir(), "checkpoint")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	// An empty file holds no checkpoint: the configured one is kept, and the
	// next save writes one into the file.
	c := &ChainConfig{Checkpoint: common.Hash{1}}
	if loaded, err := c.SetCheckpointFile(file); loaded || err != nil {
		t.Fatalf("empty file: loaded %v, error %v", loaded, err)
	}
	if c.Checkpoint != (common.Hash{1}) {
		t.Errorf("empty file changed the checkpoint to %v", c.Checkpoint)
	}
	if saved, err := c.SaveCheckpointToFile(common.Hash{2}); !saved || err != nil {
		t.Fatalf("save: saved %v, error %v", saved, err)
	}
	checkLoad(t, file, common.Hash{2})

	// Other content that isn't a checkpoint is still an error.
	if err := os.WriteFile(file, []byte("not a checkpoint"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := new(ChainConfig).SetCheckpointFile(file); err == nil {
		t.Error("no error for a file that isn't a checkpoint")
	}
}

// checkLoad checks that the checkpoint file loads, and holds want.
func checkLoad(t *testing.T, file string, want common.Hash) {
	t.Helper()
	c := new(ChainConfig)
	if loaded, err := c.SetCheckpointFile(file); !loaded || err != nil {
		t.Fatalf("load: loaded %v, error %v", loaded, err)
	}
	if c.Checkpoint != want {
		t.Errorf("loaded checkpoint %v, want %v", c.Checkpoint, want)
	}
}
