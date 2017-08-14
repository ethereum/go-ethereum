// Copyright 2017 The go-ethereum Authors
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

// +build linux darwin freebsd

package fuse

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

func check(t *testing.T, err error, format string, args ...interface{}) {
	if err != nil {
		t.Fatal(fmt.Sprintf(format, args...) + ": " + err.Error())
	}
}

type fileInfo struct {
	perm     uint64
	uid      int
	gid      int
	contents []byte
}

func createTestFilesAndUploadToSwarm(t *testing.T, api *api.Api, files map[string]fileInfo, uploadDir string) string {
	os.RemoveAll(uploadDir)

	for fname, finfo := range files {
		actualPath := filepath.Join(uploadDir, fname)
		filePath := filepath.Dir(actualPath)

		err := os.MkdirAll(filePath, 0777)
		check(t, err, "error creating directory %v", filePath)

		fd, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(finfo.perm))
		check(t, err, "error creating file %v", actualPath)

		_, err = fd.Write(finfo.contents)
		check(t, err, "error write %v", actualPath)
		// err = fd.Chown(finfo.uid, finfo.gid)
		// check(t, err, "error chown %v", actualPath)
		err = fd.Chmod(os.FileMode(finfo.perm))
		check(t, err, "error chmod %v", actualPath)
		err = fd.Sync()
		check(t, err, "error sync %v", actualPath)
		err = fd.Close()
		check(t, err, "error close %v", actualPath)
	}

	bzzhash, err := api.Upload(uploadDir, "")
	check(t, err, "error uploading directory %v", uploadDir)

	return bzzhash
}

func mountDir(t *testing.T, api *api.Api, files map[string]fileInfo, bzzHash string, mountDir string) *SwarmFS {
	os.RemoveAll(mountDir)
	os.MkdirAll(mountDir, 0777)
	swarmfs := NewSwarmFS(api)
	_, err := swarmfs.Mount(bzzHash, mountDir)
	if isFUSEUnsupportedError(err) {
		t.Skip("FUSE not supported:", err)
	} else if err != nil {
		t.Fatalf("Error mounting hash %v: %v", bzzHash, err)
	}

	found := false
	for _, minfo := range swarmfs.Listmounts() {
		if minfo.MountPoint == mountDir {
			if minfo.StartManifest != bzzHash ||
				minfo.LatestManifest != bzzHash ||
				minfo.fuseConnection == nil {
				t.Fatalf("Error mounting: exp(%s): act(%s)", bzzHash, minfo.StartManifest)
			}
			found = true
		}
	}

	// Test listMounts
	if !found {
		t.Fatalf("Error getting mounts information for %v: %v", mountDir, err)
	}

	// Check if file and their attributes are as expected
	compareGeneratedFileWithFileInMount(t, files, mountDir)

	return swarmfs
}

func compareGeneratedFileWithFileInMount(t *testing.T, files map[string]fileInfo, mountDir string) {
	err := filepath.Walk(mountDir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		fname := path[len(mountDir)+1:]
		if _, ok := files[fname]; !ok {
			t.Fatalf(" file %v present in mount dir and is not expected", fname)
		}
		return nil
	})
	check(t, err, "error walking dir %v", mountDir)

	for fname, finfo := range files {
		destinationFile := filepath.Join(mountDir, fname)

		dfinfo, err := os.Stat(destinationFile)
		check(t, err, "destination file %v missing in mount", fname)

		if int64(len(finfo.contents)) != dfinfo.Size() {
			t.Fatalf("file %v Size mismatch  source (%v) vs destination(%v)", fname, int64(len(finfo.contents)), dfinfo.Size())
		}

		if dfinfo.Mode().Perm().String() != "-rwx------" {
			t.Fatalf("file %v Permission mismatch source (-rwx------) vs destination(%v)", fname, dfinfo.Mode().Perm())
		}

		fileContents, err := ioutil.ReadFile(filepath.Join(mountDir, fname))
		check(t, err, "could not readfile %v", fname)

		if !bytes.Equal(fileContents, finfo.contents) {
			t.Fatalf("File %v contents mismatch: %v , %v", fname, fileContents, finfo.contents)
		}
		// TODO: check uid and gid
	}
}

func checkFile(t *testing.T, testMountDir, fname string, contents []byte) {
	destinationFile := filepath.Join(testMountDir, fname)
	dfinfo, err := os.Stat(destinationFile)
	check(t, err, "could not stat file %v", destinationFile)
	if dfinfo.Size() != int64(len(contents)) {
		t.Fatalf("Mismatch in size  actual(%v) vs expected(%v)", dfinfo.Size(), int64(len(contents)))
	}

	fd, err := os.OpenFile(destinationFile, os.O_RDONLY, os.FileMode(0665))
	check(t, err, "could not open file %v", destinationFile)
	newcontent := make([]byte, len(contents))
	_, err = fd.Read(newcontent)
	check(t, err, "could not read %v", destinationFile)
	err = fd.Close()
	check(t, err, "could not close %v", destinationFile)

	if !bytes.Equal(contents, newcontent) {
		t.Fatalf("File content mismatch expected (%v): received (%v) ", contents, newcontent)
	}
}

func getRandomBytes(size int) []byte {
	contents := make([]byte, size)
	rand.Read(contents)
	return contents
}

func isDirEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	return err == io.EOF
}

type testAPI struct {
	api *api.Api
}

func (ta *testAPI) mountListAndUnmount(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "fuse-source")
	check(t, err, "failed to create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "fuse-dest")
	check(t, err, "failed to create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["2.txt"] = fileInfo{0711, 333, 444, getRandomBytes(10)}
	files["3.txt"] = fileInfo{0622, 333, 444, getRandomBytes(100)}
	files["4.txt"] = fileInfo{0533, 333, 444, getRandomBytes(1024)}
	files["5.txt"] = fileInfo{0544, 333, 444, getRandomBytes(10)}
	files["6.txt"] = fileInfo{0555, 333, 444, getRandomBytes(10)}
	files["7.txt"] = fileInfo{0666, 333, 444, getRandomBytes(10)}
	files["8.txt"] = fileInfo{0777, 333, 333, getRandomBytes(10)}
	files["11.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["111.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2./2.txt"] = fileInfo{0777, 444, 444, getRandomBytes(10)}
	files["twice/2.txt"] = fileInfo{0777, 444, 333, getRandomBytes(200)}
	files["one/two/three/four/five/six/seven/eight/nine/10.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10240)}
	files["one/two/three/four/five/six/six"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	// Check unmount
	_, err = swarmfs.Unmount(testMountDir)
	check(t, err, "could not unmount %v", bzzHash)
	if !isDirEmpty(testMountDir) {
		t.Fatalf("unmount didnt work for %v", testMountDir)
	}
}

func (ta *testAPI) maxMounts(t *testing.T) {
	files := make(map[string]fileInfo)
	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir1, err := ioutil.TempDir(os.TempDir(), "max-upload1")
	check(t, err, "could not create tempdir")
	bzzHash1 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir1)
	mount1, err := ioutil.TempDir(os.TempDir(), "max-mount1")
	check(t, err, "could not create tempdir")
	swarmfs1 := mountDir(t, ta.api, files, bzzHash1, mount1)
	defer swarmfs1.Stop()

	files["2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir2, err := ioutil.TempDir(os.TempDir(), "max-upload2")
	check(t, err, "could not create tempdir")
	bzzHash2 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir2)
	mount2, err := ioutil.TempDir(os.TempDir(), "max-mount2")
	check(t, err, "could not create tempdir")
	swarmfs2 := mountDir(t, ta.api, files, bzzHash2, mount2)
	defer swarmfs2.Stop()

	files["3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir3, err := ioutil.TempDir(os.TempDir(), "max-upload3")
	check(t, err, "could not create tempdir")
	bzzHash3 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir3)
	mount3, err := ioutil.TempDir(os.TempDir(), "max-mount3")
	check(t, err, "could not create tempdir")
	swarmfs3 := mountDir(t, ta.api, files, bzzHash3, mount3)
	defer swarmfs3.Stop()

	files["4.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir4, err := ioutil.TempDir(os.TempDir(), "max-upload4")
	check(t, err, "could not create tempdir")
	bzzHash4 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir4)
	mount4, err := ioutil.TempDir(os.TempDir(), "max-mount4")
	check(t, err, "could not create tempdir")

	swarmfs4 := mountDir(t, ta.api, files, bzzHash4, mount4)
	defer swarmfs4.Stop()

	files["5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir5, err := ioutil.TempDir(os.TempDir(), "max-upload5")
	check(t, err, "could not create tempdir")
	bzzHash5 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir5)
	mount5, err := ioutil.TempDir(os.TempDir(), "max-mount5")
	check(t, err, "could not create tempdir")

	swarmfs5 := mountDir(t, ta.api, files, bzzHash5, mount5)
	defer swarmfs5.Stop()

	files["6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir6, err := ioutil.TempDir(os.TempDir(), "max-upload6")
	check(t, err, "could not create tempdir")
	bzzHash6 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir6)
	mount6, err := ioutil.TempDir(os.TempDir(), "max-mount6")
	check(t, err, "could not create tempdir")

	os.RemoveAll(mount6)
	os.MkdirAll(mount6, 0777)
	_, err = swarmfs.Mount(bzzHash6, mount6)
	if err == nil {
		t.Fatalf("Error: Going beyond max mounts  %v", bzzHash6)
	}

}

func (ta *testAPI) remount(t *testing.T) {
	files := make(map[string]fileInfo)
	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir1, err := ioutil.TempDir(os.TempDir(), "re-upload1")
	check(t, err, "could not create tempdir")
	bzzHash1 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir1)
	testMountDir1, err := ioutil.TempDir(os.TempDir(), "re-mount1")
	check(t, err, "could not create tempdir")
	swarmfs := mountDir(t, ta.api, files, bzzHash1, testMountDir1)
	defer swarmfs.Stop()

	uploadDir2, err := ioutil.TempDir(os.TempDir(), "re-upload2")
	check(t, err, "could not create tempdir")
	bzzHash2 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir2)
	testMountDir2, err := ioutil.TempDir(os.TempDir(), "re-mount2")
	check(t, err, "could not create tempdir")

	// try mounting the same hash second time
	err = os.RemoveAll(testMountDir2)
	check(t, err, "failed to removeAll")
	err = os.MkdirAll(testMountDir2, 0777)
	check(t, err, "failed to mkdirAll")
	_, err = swarmfs.Mount(bzzHash1, testMountDir2)
	check(t, err, "error mounting hash %v", bzzHash1)

	// mount a different hash in already mounted point
	_, err = swarmfs.Mount(bzzHash2, testMountDir1)
	if err == nil {
		t.Fatalf("no error when remounting %v", bzzHash2)
	}

	// mount nonexistent hash
	_, err = swarmfs.Mount("0xfea11223344", testMountDir1)
	if err == nil {
		t.Fatalf("Error mounting hash  %v", bzzHash2)
	}
}

func (ta *testAPI) unmount(t *testing.T) {
	files := make(map[string]fileInfo)
	uploadDir, err := ioutil.TempDir(os.TempDir(), "ex-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "ex-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	_, err = swarmfs.Unmount(testMountDir)
	check(t, err, "could not unmount")

	for _, minfo := range swarmfs.Listmounts() {
		if minfo.MountPoint == testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", testMountDir)
		}
	}
}

func (ta *testAPI) unmountWhenResourceBusy(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "ex-upload")
	check(t, err, "unable to create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "ex-mount")
	check(t, err, "unable to create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	actualPath := filepath.Join(testMountDir, "1.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR, os.FileMode(0700))
	check(t, err, "could not open %v", actualPath)

	_, err = d.Write(getRandomBytes(10))
	check(t, err, "could not write %v", actualPath)

	_, err = swarmfs.Unmount(testMountDir)
	check(t, err, "could not unmount %v", bzzHash)

	err = d.Close()
	check(t, err, "unable to close %v", actualPath)

	for _, minfo := range swarmfs.Listmounts() {
		if minfo.MountPoint == testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", testMountDir)
		}
	}
}

func (ta *testAPI) seekInMultiChunkFile(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "seek-upload")
	check(t, err, "unable to create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "seek-mount")
	check(t, err, "unable to create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10240)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	// Create a new file seek the second chunk
	actualPath := filepath.Join(testMountDir, "1.txt")
	d, err := os.OpenFile(actualPath, os.O_RDONLY, os.FileMode(0700))
	check(t, err, "could not open %v", actualPath)

	_, err = d.Seek(5000, 0)
	check(t, err, "could not seek %v", actualPath)
	contents := make([]byte, 1024)
	_, err = d.Read(contents)
	check(t, err, "could not seek %v", actualPath)

	finfo := files["1.txt"]

	if !bytes.Equal(finfo.contents[:6024][5000:], contents) {
		t.Fatalf("File seek contents mismatch")
	}
	err = d.Close()
	check(t, err, "could not close %v", actualPath)
}

func (ta *testAPI) createNewFile(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "create-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "create-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file in the root dir and check
	actualPath := filepath.Join(testMountDir, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	check(t, err, "could not create file %v", actualPath)

	contents := make([]byte, 11)
	rand.Read(contents)
	_, err = d.Write(contents)
	check(t, err, "could not write %v", actualPath)
	err = d.Close()
	check(t, err, "could not close %v", actualPath)
	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount %v", err)

	// mount again and see if things are okay
	files["2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "2.txt", contents)
}

func (ta *testAPI) createNewFileInsideDirectory(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "createinsidedir-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "createinsidedir-mount")
	check(t, err, "could not create tempdir")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate := filepath.Join(testMountDir, "one")
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	check(t, err, "could not create file %v", actualPath)

	contents := make([]byte, 11)
	rand.Read(contents)
	_, err = d.Write(contents)
	check(t, err, "could not write %v", actualPath)
	err = d.Close()
	check(t, err, "could not close %v", actualPath)
	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount %v", err)

	// mount again and see if things are okay
	files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "one/2.txt", contents)
}

func (ta *testAPI) createNewFileInsideNewDirectory(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "createinsidenewdir-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "createinsidenewdir-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate := filepath.Join(testMountDir, "one")
	os.MkdirAll(dirToCreate, 0777)
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	check(t, err, "could not create file %v", actualPath)

	contents := make([]byte, 11)
	rand.Read(contents)
	_, err = d.Write(contents)
	check(t, err, "could not write %v", actualPath)
	err = d.Close()
	check(t, err, "could not close %v", actualPath)
	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount %v", err)

	// mount again and see if things are okay
	files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "one/2.txt", contents)
}

func (ta *testAPI) removeExistingFile(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "remove-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "remove-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(testMountDir, "five.txt")
	os.Remove(actualPath)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	delete(files, "five.txt")
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeExistingFileInsideDir(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "remove-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "remove-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["one/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["one/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(testMountDir, "one/five.txt")
	os.Remove(actualPath)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	delete(files, "one/five.txt")
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeNewlyAddedFile(t *testing.T) {

	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "removenew-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "removenew-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Adda a new file and remove it
	dirToCreate := filepath.Join(testMountDir, "one")
	os.MkdirAll(dirToCreate, os.FileMode(0665))
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	check(t, err, "could not create file %v", actualPath)

	contents := make([]byte, 11)
	rand.Read(contents)
	_, err = d.Write(contents)
	check(t, err, "could not write %v", actualPath)
	err = d.Close()
	check(t, err, "could not close %v", actualPath)

	checkFile(t, testMountDir, "one/2.txt", contents)

	err = os.Remove(actualPath)
	check(t, err, "could not remove %v", actualPath)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	if bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", bzzHash, mi.LatestManifest)
	}
}

func (ta *testAPI) addNewFileAndModifyContents(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "modifyfile-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "modifyfile-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file in the root dir and check
	actualPath := filepath.Join(testMountDir, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	check(t, err, "could not create %v", actualPath)

	line1 := []byte("Line 1")
	_, err = rand.Read(line1)
	check(t, err, "failed to read rand")
	_, err = d.Write(line1)
	check(t, err, "failed to write %v", actualPath)
	err = d.Close()
	check(t, err, "failed to close %v", actualPath)

	mi1, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount 1")

	// mount again and see if things are okay
	files["2.txt"] = fileInfo{0700, 333, 444, line1}
	swarmfs2 := mountDir(t, ta.api, files, mi1.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "2.txt", line1)

	mi2, err := swarmfs2.Unmount(testMountDir)
	check(t, err, "could not unmount 2")

	// mount again and modify
	swarmfs3 := mountDir(t, ta.api, files, mi2.LatestManifest, testMountDir)
	defer swarmfs3.Stop()

	fd, err := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	check(t, err, "could not create %v", actualPath)

	line2 := []byte("Line 2")
	rand.Read(line2)
	_, err = fd.Seek(int64(len(line1)), 0)
	check(t, err, "failed to seek 2 %v", actualPath)
	_, err = fd.Write(line2)
	check(t, err, "failed to write 2 %v", actualPath)
	err = fd.Close()
	check(t, err, "failed to close 2 %v", actualPath)

	mi3, err := swarmfs3.Unmount(testMountDir)
	check(t, err, "could not unmount 3")

	// mount again and see if things are okay
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	files["2.txt"] = fileInfo{0700, 333, 444, line1and2}
	swarmfs4 := mountDir(t, ta.api, files, mi3.LatestManifest, testMountDir)
	defer swarmfs4.Stop()

	checkFile(t, testMountDir, "2.txt", line1and2)
}

func (ta *testAPI) removeEmptyDir(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "rmdir-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "rmdir-mount")
	check(t, err, "could not create tempdir")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	os.MkdirAll(filepath.Join(testMountDir, "newdir"), 0777)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")
	if bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", bzzHash, mi.LatestManifest)
	}
}

func (ta *testAPI) removeDirWhichHasFiles(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "rmdir-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "rmdir-mount")
	check(t, err, "could not create tempdir")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	dirPath := filepath.Join(testMountDir, "two")
	os.RemoveAll(dirPath)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	delete(files, "two/five.txt")
	delete(files, "two/six.txt")

	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeDirWhichHasSubDirs(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, err := ioutil.TempDir(os.TempDir(), "rmsubdir-upload")
	check(t, err, "could not create tempdir")
	testMountDir, err := ioutil.TempDir(os.TempDir(), "rmsubdir-mount")
	check(t, err, "could not create tempdir")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/three/2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/three/3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/six/7.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	dirPath := filepath.Join(testMountDir, "two")
	os.RemoveAll(dirPath)

	mi, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	delete(files, "two/three/2.txt")
	delete(files, "two/three/3.txt")
	delete(files, "two/four/5.txt")
	delete(files, "two/four/6.txt")
	delete(files, "two/four/six/7.txt")

	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) appendFileContentsToEnd(t *testing.T) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "appendlargefile-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "appendlargefile-mount")

	line1 := make([]byte, 10)
	rand.Read(line1)
	files["1.txt"] = fileInfo{0700, 333, 444, line1}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	actualPath := filepath.Join(testMountDir, "1.txt")
	fd, err := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	check(t, err, "could not create %v", actualPath)

	line2 := make([]byte, 5)
	rand.Read(line2)
	_, err = fd.Seek(int64(len(line1)), 0)
	check(t, err, "failed to seek %v", actualPath)
	_, err = fd.Write(line2)
	check(t, err, "failed to write %v", actualPath)
	err = fd.Close()
	check(t, err, "failed to close %v", actualPath)

	mi1, err := swarmfs1.Unmount(testMountDir)
	check(t, err, "could not unmount")

	// mount again and see if things are okay
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	files["1.txt"] = fileInfo{0700, 333, 444, line1and2}
	swarmfs2 := mountDir(t, ta.api, files, mi1.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "1.txt", line1and2)
}

func TestFUSE(t *testing.T) {
	datadir, err := ioutil.TempDir("", "fuse")
	check(t, err, "unable to create tempdir")

	os.RemoveAll(datadir)

	dpa, err := storage.NewLocalDPA(datadir)
	check(t, err, "could not create DPA")

	ta := &testAPI{api: api.NewApi(dpa, nil)}
	dpa.Start()
	defer dpa.Stop()

	t.Run("mountListAndUmount", ta.mountListAndUnmount)
	t.Run("maxMounts", ta.maxMounts)
	t.Run("remount", ta.remount)
	t.Run("unmount", ta.unmount)
	t.Run("unmountWhenResourceBusy", ta.unmountWhenResourceBusy)
	t.Run("seekInMultiChunkFile", ta.seekInMultiChunkFile)
	t.Run("createNewFile", ta.createNewFile)
	t.Run("createNewFileInsideDirectory", ta.createNewFileInsideDirectory)
	t.Run("createNewFileInsideNewDirectory", ta.createNewFileInsideNewDirectory)
	t.Run("removeExistingFile", ta.removeExistingFile)
	t.Run("removeExistingFileInsideDir", ta.removeExistingFileInsideDir)
	t.Run("removeNewlyAddedFile", ta.removeNewlyAddedFile)
	t.Run("addNewFileAndModifyContents", ta.addNewFileAndModifyContents)
	t.Run("removeEmptyDir", ta.removeEmptyDir)
	t.Run("removeDirWhichHasFiles", ta.removeDirWhichHasFiles)
	t.Run("removeDirWhichHasSubDirs", ta.removeDirWhichHasSubDirs)
	t.Run("appendFileContentsToEnd", ta.appendFileContentsToEnd)
}
