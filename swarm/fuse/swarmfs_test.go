// Copyright 2018 The go-ethereum Authors
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
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"

	"github.com/ethereum/go-ethereum/log"

	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel    = flag.Int("loglevel", 4, "verbosity of logs")
	rawlog      = flag.Bool("rawlog", false, "turn off terminal formatting in logs")
	longrunning = flag.Bool("longrunning", false, "do run long-running tests")
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(!*rawlog))))
}

type fileInfo struct {
	perm     uint64
	uid      int
	gid      int
	contents []byte
}

//create files from the map of name and content provided and upload them to swarm via api
func createTestFilesAndUploadToSwarm(t *testing.T, api *api.API, files map[string]fileInfo, uploadDir string, toEncrypt bool) string {

	//iterate the map
	for fname, finfo := range files {
		actualPath := filepath.Join(uploadDir, fname)
		filePath := filepath.Dir(actualPath)

		//create directory
		err := os.MkdirAll(filePath, 0777)
		if err != nil {
			t.Fatalf("Error creating directory '%v' : %v", filePath, err)
		}

		//create file
		fd, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(finfo.perm))
		if err1 != nil {
			t.Fatalf("Error creating file %v: %v", actualPath, err1)
		}

		//write content to file
		_, err = fd.Write(finfo.contents)
		if err != nil {
			t.Fatalf("Error writing to file '%v' : %v", filePath, err)
		}
		/*
				Note @holisticode: It's not clear why the Chown command was added to the test suite.
				Some files are initialized with different permissions in the individual test,
				resulting in errors on Chown which were not checked.
			  After adding the checks tests would fail.

				What's then the reason to have this check in the first place?
				Disabling for now

			  err = fd.Chown(finfo.uid, finfo.gid)
			  if err != nil {
			    t.Fatalf("Error chown file '%v' : %v", filePath, err)
				}
		*/
		err = fd.Chmod(os.FileMode(finfo.perm))
		if err != nil {
			t.Fatalf("Error chmod file '%v' : %v", filePath, err)
		}
		err = fd.Sync()
		if err != nil {
			t.Fatalf("Error sync file '%v' : %v", filePath, err)
		}
		err = fd.Close()
		if err != nil {
			t.Fatalf("Error closing file '%v' : %v", filePath, err)
		}
	}

	//upload directory to swarm and return hash
	bzzhash, err := api.Upload(context.TODO(), uploadDir, "", toEncrypt)
	if err != nil {
		t.Fatalf("Error uploading directory %v: %vm encryption: %v", uploadDir, err, toEncrypt)
	}

	return bzzhash
}

//mount a swarm hash as a directory on files system via FUSE
func mountDir(t *testing.T, api *api.API, files map[string]fileInfo, bzzHash string, mountDir string) *SwarmFS {
	swarmfs := NewSwarmFS(api)
	_, err := swarmfs.Mount(bzzHash, mountDir)
	if isFUSEUnsupportedError(err) {
		t.Skip("FUSE not supported:", err)
	} else if err != nil {
		t.Fatalf("Error mounting hash %v: %v", bzzHash, err)
	}

	//check directory is mounted
	found := false
	mi := swarmfs.Listmounts()
	for _, minfo := range mi {
		minfo.lock.RLock()
		if minfo.MountPoint == mountDir {
			if minfo.StartManifest != bzzHash ||
				minfo.LatestManifest != bzzHash ||
				minfo.fuseConnection == nil {
				minfo.lock.RUnlock()
				t.Fatalf("Error mounting: exp(%s): act(%s)", bzzHash, minfo.StartManifest)
			}
			found = true
		}
		minfo.lock.RUnlock()
	}

	// Test listMounts
	if !found {
		t.Fatalf("Error getting mounts information for %v: %v", mountDir, err)
	}

	// Check if file and their attributes are as expected
	compareGeneratedFileWithFileInMount(t, files, mountDir)

	return swarmfs
}

// Check if file and their attributes are as expected
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
	if err != nil {
		t.Fatalf("Error walking dir %v", mountDir)
	}

	for fname, finfo := range files {
		destinationFile := filepath.Join(mountDir, fname)

		dfinfo, err := os.Stat(destinationFile)
		if err != nil {
			t.Fatalf("Destination file %v missing in mount: %v", fname, err)
		}

		if int64(len(finfo.contents)) != dfinfo.Size() {
			t.Fatalf("file %v Size mismatch  source (%v) vs destination(%v)", fname, int64(len(finfo.contents)), dfinfo.Size())
		}

		if dfinfo.Mode().Perm().String() != "-rwx------" {
			t.Fatalf("file %v Permission mismatch source (-rwx------) vs destination(%v)", fname, dfinfo.Mode().Perm())
		}

		fileContents, err := ioutil.ReadFile(filepath.Join(mountDir, fname))
		if err != nil {
			t.Fatalf("Could not readfile %v : %v", fname, err)
		}
		if !bytes.Equal(fileContents, finfo.contents) {
			t.Fatalf("File %v contents mismatch: %v , %v", fname, fileContents, finfo.contents)
		}
		// TODO: check uid and gid
	}
}

//check mounted file with provided content
func checkFile(t *testing.T, testMountDir, fname string, contents []byte) {
	destinationFile := filepath.Join(testMountDir, fname)
	dfinfo, err1 := os.Stat(destinationFile)
	if err1 != nil {
		t.Fatalf("Could not stat file %v", destinationFile)
	}
	if dfinfo.Size() != int64(len(contents)) {
		t.Fatalf("Mismatch in size  actual(%v) vs expected(%v)", dfinfo.Size(), int64(len(contents)))
	}

	fd, err2 := os.OpenFile(destinationFile, os.O_RDONLY, os.FileMode(0665))
	if err2 != nil {
		t.Fatalf("Could not open file %v", destinationFile)
	}
	newcontent := make([]byte, len(contents))
	_, err := fd.Read(newcontent)
	if err != nil {
		t.Fatalf("Could not read from file %v", err)
	}
	err = fd.Close()
	if err != nil {
		t.Fatalf("Could not close file %v", err)
	}

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
	api *api.API
}

type testData struct {
	testDir       string
	testUploadDir string
	testMountDir  string
	bzzHash       string
	files         map[string]fileInfo
	toEncrypt     bool
	swarmfs       *SwarmFS
}

//create the root dir of a test
func (ta *testAPI) initSubtest(name string) (*testData, error) {
	var err error
	d := &testData{}
	d.testDir, err = ioutil.TempDir(os.TempDir(), name)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create test dir: %v", err)
	}
	return d, nil
}

//upload data and mount directory
func (ta *testAPI) uploadAndMount(dat *testData, t *testing.T) (*testData, error) {
	//create upload dir
	err := os.MkdirAll(dat.testUploadDir, 0777)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create upload dir: %v", err)
	}
	//create mount dir
	err = os.MkdirAll(dat.testMountDir, 0777)
	if err != nil {
		return nil, fmt.Errorf("Couldn't create mount dir: %v", err)
	}
	//upload the file
	dat.bzzHash = createTestFilesAndUploadToSwarm(t, ta.api, dat.files, dat.testUploadDir, dat.toEncrypt)
	log.Debug("Created test files and uploaded to Swarm")
	//mount the directory
	dat.swarmfs = mountDir(t, ta.api, dat.files, dat.bzzHash, dat.testMountDir)
	log.Debug("Mounted swarm fs")
	return dat, nil
}

//add a directory to the test directory tree
func addDir(root string, name string) (string, error) {
	d := filepath.Join(root, name)
	err := os.MkdirAll(d, 0777)
	if err != nil {
		return "", fmt.Errorf("Couldn't create dir inside test dir: %v", err)
	}
	return d, nil
}

func (ta *testAPI) mountListAndUnmountEncrypted(t *testing.T) {
	log.Debug("Starting mountListAndUnmountEncrypted test")
	ta.mountListAndUnmount(t, true)
	log.Debug("Test mountListAndUnmountEncrypted terminated")
}

func (ta *testAPI) mountListAndUnmountNonEncrypted(t *testing.T) {
	log.Debug("Starting mountListAndUnmountNonEncrypted test")
	ta.mountListAndUnmount(t, false)
	log.Debug("Test mountListAndUnmountNonEncrypted terminated")
}

//mount a directory unmount and check the directory is empty afterwards
func (ta *testAPI) mountListAndUnmount(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("mountListAndUnmount")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "testUploadDir")
	dat.testMountDir = filepath.Join(dat.testDir, "testMountDir")
	dat.files = make(map[string]fileInfo)

	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["2.txt"] = fileInfo{0711, 333, 444, getRandomBytes(10)}
	dat.files["3.txt"] = fileInfo{0622, 333, 444, getRandomBytes(100)}
	dat.files["4.txt"] = fileInfo{0533, 333, 444, getRandomBytes(1024)}
	dat.files["5.txt"] = fileInfo{0544, 333, 444, getRandomBytes(10)}
	dat.files["6.txt"] = fileInfo{0555, 333, 444, getRandomBytes(10)}
	dat.files["7.txt"] = fileInfo{0666, 333, 444, getRandomBytes(10)}
	dat.files["8.txt"] = fileInfo{0777, 333, 333, getRandomBytes(10)}
	dat.files["11.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	dat.files["111.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	dat.files["two/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	dat.files["two/2/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	dat.files["two/2./2.txt"] = fileInfo{0777, 444, 444, getRandomBytes(10)}
	dat.files["twice/2.txt"] = fileInfo{0777, 444, 333, getRandomBytes(200)}
	dat.files["one/two/three/four/five/six/seven/eight/nine/10.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10240)}
	dat.files["one/two/three/four/five/six/six"] = fileInfo{0777, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()
	// Check unmount
	_, err = dat.swarmfs.Unmount(dat.testMountDir)
	if err != nil {
		t.Fatalf("could not unmount  %v", dat.bzzHash)
	}
	log.Debug("Unmount successful")
	if !isDirEmpty(dat.testMountDir) {
		t.Fatalf("unmount didnt work for %v", dat.testMountDir)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) maxMountsEncrypted(t *testing.T) {
	log.Debug("Starting maxMountsEncrypted test")
	ta.runMaxMounts(t, true)
	log.Debug("Test maxMountsEncrypted terminated")
}

func (ta *testAPI) maxMountsNonEncrypted(t *testing.T) {
	log.Debug("Starting maxMountsNonEncrypted test")
	ta.runMaxMounts(t, false)
	log.Debug("Test maxMountsNonEncrypted terminated")
}

//mount several different directories until the maximum has been reached
func (ta *testAPI) runMaxMounts(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("runMaxMounts")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "max-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "max-mount1")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	dat.testUploadDir = filepath.Join(dat.testDir, "max-upload2")
	dat.testMountDir = filepath.Join(dat.testDir, "max-mount2")
	dat.files["2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}

	dat.testUploadDir = filepath.Join(dat.testDir, "max-upload3")
	dat.testMountDir = filepath.Join(dat.testDir, "max-mount3")
	dat.files["3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}

	dat.testUploadDir = filepath.Join(dat.testDir, "max-upload4")
	dat.testMountDir = filepath.Join(dat.testDir, "max-mount4")
	dat.files["4.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}

	dat.testUploadDir = filepath.Join(dat.testDir, "max-upload5")
	dat.testMountDir = filepath.Join(dat.testDir, "max-mount5")
	dat.files["5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}

	//now try an additional mount, should fail due to max mounts reached
	testUploadDir6 := filepath.Join(dat.testDir, "max-upload6")
	err = os.MkdirAll(testUploadDir6, 0777)
	if err != nil {
		t.Fatalf("Couldn't create upload dir 6: %v", err)
	}
	dat.files["6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	testMountDir6 := filepath.Join(dat.testDir, "max-mount6")
	err = os.MkdirAll(testMountDir6, 0777)
	if err != nil {
		t.Fatalf("Couldn't create mount dir 5: %v", err)
	}
	bzzHash6 := createTestFilesAndUploadToSwarm(t, ta.api, dat.files, testUploadDir6, toEncrypt)
	log.Debug("Created test files and uploaded to swarm with uploadDir6")
	_, err = dat.swarmfs.Mount(bzzHash6, testMountDir6)
	if err == nil {
		t.Fatalf("Expected this mount to fail due to exceeding max number of allowed mounts, but succeeded. %v", bzzHash6)
	}
	log.Debug("Maximum mount reached, additional mount failed. Correct.")
}

func (ta *testAPI) remountEncrypted(t *testing.T) {
	log.Debug("Starting remountEncrypted test")
	ta.remount(t, true)
	log.Debug("Test remountEncrypted terminated")
}
func (ta *testAPI) remountNonEncrypted(t *testing.T) {
	log.Debug("Starting remountNonEncrypted test")
	ta.remount(t, false)
	log.Debug("Test remountNonEncrypted terminated")
}

//test remounting same hash second time and different hash in already mounted point
func (ta *testAPI) remount(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("remount")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "remount-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "remount-mount1")
	dat.files = make(map[string]fileInfo)

	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// try mounting the same hash second time
	testMountDir2, err2 := addDir(dat.testDir, "remount-mount2")
	if err2 != nil {
		t.Fatalf("Error creating second mount dir: %v", err2)
	}
	_, err2 = dat.swarmfs.Mount(dat.bzzHash, testMountDir2)
	if err2 != nil {
		t.Fatalf("Error mounting hash second time on different dir  %v", dat.bzzHash)
	}

	// mount a different hash in already mounted point
	dat.files["2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	testUploadDir2, err3 := addDir(dat.testDir, "remount-upload2")
	if err3 != nil {
		t.Fatalf("Error creating second upload dir: %v", err3)
	}
	bzzHash2 := createTestFilesAndUploadToSwarm(t, ta.api, dat.files, testUploadDir2, toEncrypt)
	_, err = swarmfs.Mount(bzzHash2, dat.testMountDir)
	if err == nil {
		t.Fatalf("Error mounting hash  %v", bzzHash2)
	}
	log.Debug("Mount on existing mount point failed. Correct.")

	// mount nonexistent hash
	failDir, err3 := addDir(dat.testDir, "remount-fail")
	if err3 != nil {
		t.Fatalf("Error creating remount dir: %v", bzzHash2)
	}
	failHash := "0xfea11223344"
	_, err = swarmfs.Mount(failHash, failDir)
	if err == nil {
		t.Fatalf("Expected this mount to fail due to non existing hash. But succeeded %v", failHash)
	}
	log.Debug("Nonexistent hash hasn't been mounted. Correct.")
}

func (ta *testAPI) unmountEncrypted(t *testing.T) {
	log.Debug("Starting unmountEncrypted test")
	ta.unmount(t, true)
	log.Debug("Test unmountEncrypted terminated")
}

func (ta *testAPI) unmountNonEncrypted(t *testing.T) {
	log.Debug("Starting unmountNonEncrypted test")
	ta.unmount(t, false)
	log.Debug("Test unmountNonEncrypted terminated")
}

//mount then unmount and check that it has been unmounted
func (ta *testAPI) unmount(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("unmount")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "ex-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "ex-mount1")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	_, err = dat.swarmfs.Unmount(dat.testMountDir)
	if err != nil {
		t.Fatalf("could not unmount  %v", dat.bzzHash)
	}
	log.Debug("Unmounted Dir")

	mi := swarmfs.Listmounts()
	log.Debug("Going to list mounts")
	for _, minfo := range mi {
		log.Debug("Mount point in list: ", "point", minfo.MountPoint)
		if minfo.MountPoint == dat.testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", dat.testMountDir)
		}
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) unmountWhenResourceBusyEncrypted(t *testing.T) {
	log.Debug("Starting unmountWhenResourceBusyEncrypted test")
	ta.unmountWhenResourceBusy(t, true)
	log.Debug("Test unmountWhenResourceBusyEncrypted terminated")
}
func (ta *testAPI) unmountWhenResourceBusyNonEncrypted(t *testing.T) {
	log.Debug("Starting unmountWhenResourceBusyNonEncrypted test")
	ta.unmountWhenResourceBusy(t, false)
	log.Debug("Test unmountWhenResourceBusyNonEncrypted terminated")
}

//unmount while a resource is busy; should fail
func (ta *testAPI) unmountWhenResourceBusy(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("unmountWhenResourceBusy")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "ex-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "ex-mount1")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	//create a file in the mounted directory, then try to unmount - should fail
	actualPath := filepath.Join(dat.testMountDir, "2.txt")
	//d, err := os.OpenFile(actualPath, os.O_RDWR, os.FileMode(0700))
	d, err := os.Create(actualPath)
	if err != nil {
		t.Fatalf("Couldn't create new file: %v", err)
	}
	//we need to manually close the file before mount for this test
	//but let's defer too in case of errors
	defer d.Close()
	_, err = d.Write(getRandomBytes(10))
	if err != nil {
		t.Fatalf("Couldn't write to file: %v", err)
	}
	log.Debug("Bytes written")

	_, err = dat.swarmfs.Unmount(dat.testMountDir)
	if err == nil {
		t.Fatalf("Expected mount to fail due to resource busy, but it succeeded...")
	}
	//free resources
	err = d.Close()
	if err != nil {
		t.Fatalf("Couldn't close file!  %v", dat.bzzHash)
	}
	log.Debug("File closed")

	//now unmount after explicitly closed file
	_, err = dat.swarmfs.Unmount(dat.testMountDir)
	if err != nil {
		t.Fatalf("Expected mount to succeed after freeing resource, but it failed: %v", err)
	}
	//check if the dir is still mounted
	mi := dat.swarmfs.Listmounts()
	log.Debug("Going to list mounts")
	for _, minfo := range mi {
		log.Debug("Mount point in list: ", "point", minfo.MountPoint)
		if minfo.MountPoint == dat.testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", dat.testMountDir)
		}
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) seekInMultiChunkFileEncrypted(t *testing.T) {
	log.Debug("Starting seekInMultiChunkFileEncrypted test")
	ta.seekInMultiChunkFile(t, true)
	log.Debug("Test seekInMultiChunkFileEncrypted terminated")
}

func (ta *testAPI) seekInMultiChunkFileNonEncrypted(t *testing.T) {
	log.Debug("Starting seekInMultiChunkFileNonEncrypted test")
	ta.seekInMultiChunkFile(t, false)
	log.Debug("Test seekInMultiChunkFileNonEncrypted terminated")
}

//open a file in a mounted dir and go to a certain position
func (ta *testAPI) seekInMultiChunkFile(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("seekInMultiChunkFile")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "seek-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "seek-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10240)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Open the file in the mounted dir and seek the second chunk
	actualPath := filepath.Join(dat.testMountDir, "1.txt")
	d, err := os.OpenFile(actualPath, os.O_RDONLY, os.FileMode(0700))
	if err != nil {
		t.Fatalf("Couldn't open file: %v", err)
	}
	log.Debug("Opened file")
	defer func() {
		err := d.Close()
		if err != nil {
			t.Fatalf("Error closing file! %v", err)
		}
	}()

	_, err = d.Seek(5000, 0)
	if err != nil {
		t.Fatalf("Error seeking in file: %v", err)
	}

	contents := make([]byte, 1024)
	_, err = d.Read(contents)
	if err != nil {
		t.Fatalf("Error reading file: %v", err)
	}
	log.Debug("Read contents")
	finfo := dat.files["1.txt"]

	if !bytes.Equal(finfo.contents[:6024][5000:], contents) {
		t.Fatalf("File seek contents mismatch")
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) createNewFileEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileEncrypted test")
	ta.createNewFile(t, true)
	log.Debug("Test createNewFileEncrypted terminated")
}

func (ta *testAPI) createNewFileNonEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileNonEncrypted test")
	ta.createNewFile(t, false)
	log.Debug("Test createNewFileNonEncrypted terminated")
}

//create a new file in a mounted swarm directory,
//unmount the fuse dir and then remount to see if new file is still there
func (ta *testAPI) createNewFile(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("createNewFile")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "create-upload1")
	dat.testMountDir = filepath.Join(dat.testDir, "create-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Create a new file in the root dir and check
	actualPath := filepath.Join(dat.testMountDir, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not open file %s : %v", actualPath, err1)
	}
	defer d.Close()
	log.Debug("Opened file")
	contents := make([]byte, 11)
	_, err = rand.Read(contents)
	if err != nil {
		t.Fatalf("Could not rand read contents %v", err)
	}
	log.Debug("content read")
	_, err = d.Write(contents)
	if err != nil {
		t.Fatalf("Couldn't write contents: %v", err)
	}
	log.Debug("content written")
	err = d.Close()
	if err != nil {
		t.Fatalf("Couldn't close file: %v", err)
	}
	log.Debug("file closed")

	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	testMountDir2, err3 := addDir(dat.testDir, "create-mount2")
	if err3 != nil {
		t.Fatalf("Error creating mount dir2: %v", err3)
	}
	// mount again and see if things are okay
	dat.files["2.txt"] = fileInfo{0700, 333, 444, contents}
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, testMountDir2)
	log.Debug("Directory mounted again")

	checkFile(t, testMountDir2, "2.txt", contents)
	_, err2 = dat.swarmfs.Unmount(testMountDir2)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) createNewFileInsideDirectoryEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileInsideDirectoryEncrypted test")
	ta.createNewFileInsideDirectory(t, true)
	log.Debug("Test createNewFileInsideDirectoryEncrypted terminated")
}

func (ta *testAPI) createNewFileInsideDirectoryNonEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileInsideDirectoryNonEncrypted test")
	ta.createNewFileInsideDirectory(t, false)
	log.Debug("Test createNewFileInsideDirectoryNonEncrypted terminated")
}

//create a new file inside a directory inside the mount
func (ta *testAPI) createNewFileInsideDirectory(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("createNewFileInsideDirectory")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "createinsidedir-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "createinsidedir-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate := filepath.Join(dat.testMountDir, "one")
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	defer d.Close()
	log.Debug("File opened")
	contents := make([]byte, 11)
	_, err = rand.Read(contents)
	if err != nil {
		t.Fatalf("Error filling random bytes into byte array %v", err)
	}
	log.Debug("Content read")
	_, err = d.Write(contents)
	if err != nil {
		t.Fatalf("Error writing random bytes into file %v", err)
	}
	log.Debug("Content written")
	err = d.Close()
	if err != nil {
		t.Fatalf("Error closing file %v", err)
	}
	log.Debug("File closed")

	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	testMountDir2, err3 := addDir(dat.testDir, "createinsidedir-mount2")
	if err3 != nil {
		t.Fatalf("Error creating mount dir2: %v", err3)
	}
	// mount again and see if things are okay
	dat.files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, testMountDir2)
	log.Debug("Directory mounted again")

	checkFile(t, testMountDir2, "one/2.txt", contents)
	_, err = dat.swarmfs.Unmount(testMountDir2)
	if err != nil {
		t.Fatalf("could not unmount  %v", dat.bzzHash)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) createNewFileInsideNewDirectoryEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileInsideNewDirectoryEncrypted test")
	ta.createNewFileInsideNewDirectory(t, true)
	log.Debug("Test createNewFileInsideNewDirectoryEncrypted terminated")
}

func (ta *testAPI) createNewFileInsideNewDirectoryNonEncrypted(t *testing.T) {
	log.Debug("Starting createNewFileInsideNewDirectoryNonEncrypted test")
	ta.createNewFileInsideNewDirectory(t, false)
	log.Debug("Test createNewFileInsideNewDirectoryNonEncrypted terminated")
}

//create a new directory in mount and a new file
func (ta *testAPI) createNewFileInsideNewDirectory(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("createNewFileInsideNewDirectory")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "createinsidenewdir-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "createinsidenewdir-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate, err2 := addDir(dat.testMountDir, "one")
	if err2 != nil {
		t.Fatalf("Error creating mount dir2: %v", err2)
	}
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	defer d.Close()
	log.Debug("File opened")
	contents := make([]byte, 11)
	_, err = rand.Read(contents)
	if err != nil {
		t.Fatalf("Error writing random bytes to byte array: %v", err)
	}
	log.Debug("content read")
	_, err = d.Write(contents)
	if err != nil {
		t.Fatalf("Error writing to file: %v", err)
	}
	log.Debug("content written")
	err = d.Close()
	if err != nil {
		t.Fatalf("Error closing file: %v", err)
	}
	log.Debug("File closed")

	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	// mount again and see if things are okay
	dat.files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, dat.testMountDir)
	log.Debug("Directory mounted again")

	checkFile(t, dat.testMountDir, "one/2.txt", contents)
	_, err2 = dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeExistingFileEncrypted(t *testing.T) {
	log.Debug("Starting removeExistingFileEncrypted test")
	ta.removeExistingFile(t, true)
	log.Debug("Test removeExistingFileEncrypted terminated")
}

func (ta *testAPI) removeExistingFileNonEncrypted(t *testing.T) {
	log.Debug("Starting removeExistingFileNonEncrypted test")
	ta.removeExistingFile(t, false)
	log.Debug("Test removeExistingFileNonEncrypted terminated")
}

//remove existing file in mount
func (ta *testAPI) removeExistingFile(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeExistingFile")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "remove-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "remove-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(dat.testMountDir, "five.txt")
	err = os.Remove(actualPath)
	if err != nil {
		t.Fatalf("Error removing file! %v", err)
	}
	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	// mount again and see if things are okay
	delete(dat.files, "five.txt")
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, dat.testMountDir)
	_, err = os.Stat(actualPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	_, err2 = dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeExistingFileInsideDirEncrypted(t *testing.T) {
	log.Debug("Starting removeExistingFileInsideDirEncrypted test")
	ta.removeExistingFileInsideDir(t, true)
	log.Debug("Test removeExistingFileInsideDirEncrypted terminated")
}

func (ta *testAPI) removeExistingFileInsideDirNonEncrypted(t *testing.T) {
	log.Debug("Starting removeExistingFileInsideDirNonEncrypted test")
	ta.removeExistingFileInsideDir(t, false)
	log.Debug("Test removeExistingFileInsideDirNonEncrypted terminated")
}

//remove a file inside a directory inside a mount
func (ta *testAPI) removeExistingFileInsideDir(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeExistingFileInsideDir")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "remove-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "remove-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["one/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["one/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(dat.testMountDir, "one")
	actualPath = filepath.Join(actualPath, "five.txt")
	err = os.Remove(actualPath)
	if err != nil {
		t.Fatalf("Error removing file! %v", err)
	}
	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	// mount again and see if things are okay
	delete(dat.files, "one/five.txt")
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, dat.testMountDir)
	_, err = os.Stat(actualPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}

	okPath := filepath.Join(dat.testMountDir, "one")
	okPath = filepath.Join(okPath, "six.txt")
	_, err = os.Stat(okPath)
	if err != nil {
		t.Fatal("Expected file to be present in re-mount after removal, but it is not there")
	}
	_, err2 = dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeNewlyAddedFileEncrypted(t *testing.T) {
	log.Debug("Starting removeNewlyAddedFileEncrypted test")
	ta.removeNewlyAddedFile(t, true)
	log.Debug("Test removeNewlyAddedFileEncrypted terminated")
}

func (ta *testAPI) removeNewlyAddedFileNonEncrypted(t *testing.T) {
	log.Debug("Starting removeNewlyAddedFileNonEncrypted test")
	ta.removeNewlyAddedFile(t, false)
	log.Debug("Test removeNewlyAddedFileNonEncrypted terminated")
}

//add a file in mount and then remove it; on remount file should not be there
func (ta *testAPI) removeNewlyAddedFile(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeNewlyAddedFile")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "removenew-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "removenew-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Add a a new file and remove it
	dirToCreate := filepath.Join(dat.testMountDir, "one")
	err = os.MkdirAll(dirToCreate, os.FileMode(0665))
	if err != nil {
		t.Fatalf("Error creating dir in mounted dir: %v", err)
	}
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	defer d.Close()
	log.Debug("file opened")
	contents := make([]byte, 11)
	_, err = rand.Read(contents)
	if err != nil {
		t.Fatalf("Error writing random bytes to byte array: %v", err)
	}
	log.Debug("content read")
	_, err = d.Write(contents)
	if err != nil {
		t.Fatalf("Error writing random bytes to file: %v", err)
	}
	log.Debug("content written")
	err = d.Close()
	if err != nil {
		t.Fatalf("Error closing file: %v", err)
	}
	log.Debug("file closed")

	checkFile(t, dat.testMountDir, "one/2.txt", contents)
	log.Debug("file checked")

	err = os.Remove(actualPath)
	if err != nil {
		t.Fatalf("Error removing file: %v", err)
	}
	log.Debug("file removed")

	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	testMountDir2, err3 := addDir(dat.testDir, "removenew-mount2")
	if err3 != nil {
		t.Fatalf("Error creating mount dir2: %v", err3)
	}
	// mount again and see if things are okay
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, testMountDir2)
	log.Debug("Directory mounted again")

	if dat.bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", dat.bzzHash, mi.LatestManifest)
	}
	_, err2 = dat.swarmfs.Unmount(testMountDir2)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) addNewFileAndModifyContentsEncrypted(t *testing.T) {
	log.Debug("Starting addNewFileAndModifyContentsEncrypted test")
	ta.addNewFileAndModifyContents(t, true)
	log.Debug("Test addNewFileAndModifyContentsEncrypted terminated")
}

func (ta *testAPI) addNewFileAndModifyContentsNonEncrypted(t *testing.T) {
	log.Debug("Starting addNewFileAndModifyContentsNonEncrypted test")
	ta.addNewFileAndModifyContents(t, false)
	log.Debug("Test addNewFileAndModifyContentsNonEncrypted terminated")
}

//add a new file and modify content; remount and check the modified file is intact
func (ta *testAPI) addNewFileAndModifyContents(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("addNewFileAndModifyContents")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "modifyfile-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "modifyfile-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	// Create a new file in the root dir
	actualPath := filepath.Join(dat.testMountDir, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	defer d.Close()
	//write some random data into the file
	log.Debug("file opened")
	line1 := []byte("Line 1")
	_, err = rand.Read(line1)
	if err != nil {
		t.Fatalf("Error writing random bytes to byte array: %v", err)
	}
	log.Debug("line read")
	_, err = d.Write(line1)
	if err != nil {
		t.Fatalf("Error writing random bytes to file: %v", err)
	}
	log.Debug("line written")
	err = d.Close()
	if err != nil {
		t.Fatalf("Error closing file: %v", err)
	}
	log.Debug("file closed")

	//unmount the hash on the mounted dir
	mi1, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	log.Debug("Directory unmounted")

	//mount on a different dir to see if modified file is correct
	testMountDir2, err3 := addDir(dat.testDir, "modifyfile-mount2")
	if err3 != nil {
		t.Fatalf("Error creating mount dir2: %v", err3)
	}
	dat.files["2.txt"] = fileInfo{0700, 333, 444, line1}
	_ = mountDir(t, ta.api, dat.files, mi1.LatestManifest, testMountDir2)
	log.Debug("Directory mounted again")

	checkFile(t, testMountDir2, "2.txt", line1)
	log.Debug("file checked")

	//unmount second dir
	mi2, err4 := dat.swarmfs.Unmount(testMountDir2)
	if err4 != nil {
		t.Fatalf("Could not unmount %v", err4)
	}
	log.Debug("Directory unmounted again")

	//mount again on original dir and modify the file
	//let's clean up the mounted dir first: remove...
	err = os.RemoveAll(dat.testMountDir)
	if err != nil {
		t.Fatalf("Error cleaning up mount dir: %v", err)
	}
	//...and re-create
	err = os.MkdirAll(dat.testMountDir, 0777)
	if err != nil {
		t.Fatalf("Error re-creating mount dir: %v", err)
	}
	//now remount
	_ = mountDir(t, ta.api, dat.files, mi2.LatestManifest, dat.testMountDir)
	log.Debug("Directory mounted yet again")

	//open the file....
	fd, err5 := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	if err5 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err5)
	}
	defer fd.Close()
	log.Debug("file opened")
	//...and modify something
	line2 := []byte("Line 2")
	_, err = rand.Read(line2)
	if err != nil {
		t.Fatalf("Error modifying random bytes to byte array: %v", err)
	}
	log.Debug("line read")
	_, err = fd.Seek(int64(len(line1)), 0)
	if err != nil {
		t.Fatalf("Error seeking position for modification: %v", err)
	}
	_, err = fd.Write(line2)
	if err != nil {
		t.Fatalf("Error modifying file: %v", err)
	}
	log.Debug("line written")
	err = fd.Close()
	if err != nil {
		t.Fatalf("Error closing modified file; %v", err)
	}
	log.Debug("file closed")

	//unmount the modified directory
	mi3, err6 := dat.swarmfs.Unmount(dat.testMountDir)
	if err6 != nil {
		t.Fatalf("Could not unmount %v", err6)
	}
	log.Debug("Directory unmounted yet again")

	//now remount on a different dir and check that the modified file is ok
	testMountDir4, err7 := addDir(dat.testDir, "modifyfile-mount4")
	if err7 != nil {
		t.Fatalf("Could not unmount %v", err7)
	}
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	dat.files["2.txt"] = fileInfo{0700, 333, 444, line1and2}
	_ = mountDir(t, ta.api, dat.files, mi3.LatestManifest, testMountDir4)
	log.Debug("Directory mounted final time")

	checkFile(t, testMountDir4, "2.txt", line1and2)
	_, err = dat.swarmfs.Unmount(testMountDir4)
	if err != nil {
		t.Fatalf("Could not unmount %v", err)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeEmptyDirEncrypted(t *testing.T) {
	log.Debug("Starting removeEmptyDirEncrypted test")
	ta.removeEmptyDir(t, true)
	log.Debug("Test removeEmptyDirEncrypted terminated")
}

func (ta *testAPI) removeEmptyDirNonEncrypted(t *testing.T) {
	log.Debug("Starting removeEmptyDirNonEncrypted test")
	ta.removeEmptyDir(t, false)
	log.Debug("Test removeEmptyDirNonEncrypted terminated")
}

//remove an empty dir inside mount
func (ta *testAPI) removeEmptyDir(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeEmptyDir")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "rmdir-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "rmdir-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	_, err2 := addDir(dat.testMountDir, "newdir")
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}
	mi, err := dat.swarmfs.Unmount(dat.testMountDir)
	if err != nil {
		t.Fatalf("Could not unmount %v", err)
	}
	log.Debug("Directory unmounted")
	//by just adding an empty dir, the hash doesn't change; test this
	if dat.bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", dat.bzzHash, mi.LatestManifest)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeDirWhichHasFilesEncrypted(t *testing.T) {
	log.Debug("Starting removeDirWhichHasFilesEncrypted test")
	ta.removeDirWhichHasFiles(t, true)
	log.Debug("Test removeDirWhichHasFilesEncrypted terminated")
}
func (ta *testAPI) removeDirWhichHasFilesNonEncrypted(t *testing.T) {
	log.Debug("Starting removeDirWhichHasFilesNonEncrypted test")
	ta.removeDirWhichHasFiles(t, false)
	log.Debug("Test removeDirWhichHasFilesNonEncrypted terminated")
}

//remove a directory with a file; check on remount file isn't there
func (ta *testAPI) removeDirWhichHasFiles(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeDirWhichHasFiles")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "rmdir-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "rmdir-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	//delete a directory inside the mounted dir with all its files
	dirPath := filepath.Join(dat.testMountDir, "two")
	err = os.RemoveAll(dirPath)
	if err != nil {
		t.Fatalf("Error removing directory in mounted dir: %v", err)
	}

	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v ", err2)
	}
	log.Debug("Directory unmounted")

	//we deleted files in the OS, so let's delete them also in the files map
	delete(dat.files, "two/five.txt")
	delete(dat.files, "two/six.txt")

	// mount again and see if deleted files have been deleted indeed
	testMountDir2, err3 := addDir(dat.testDir, "remount-mount2")
	if err3 != nil {
		t.Fatalf("Could not unmount %v", err3)
	}
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, testMountDir2)
	log.Debug("Directory mounted")
	actualPath := filepath.Join(dirPath, "five.txt")
	_, err = os.Stat(actualPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	_, err = os.Stat(dirPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	_, err = dat.swarmfs.Unmount(testMountDir2)
	if err != nil {
		t.Fatalf("Could not unmount %v", err)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) removeDirWhichHasSubDirsEncrypted(t *testing.T) {
	log.Debug("Starting removeDirWhichHasSubDirsEncrypted test")
	ta.removeDirWhichHasSubDirs(t, true)
	log.Debug("Test removeDirWhichHasSubDirsEncrypted terminated")
}

func (ta *testAPI) removeDirWhichHasSubDirsNonEncrypted(t *testing.T) {
	log.Debug("Starting removeDirWhichHasSubDirsNonEncrypted test")
	ta.removeDirWhichHasSubDirs(t, false)
	log.Debug("Test removeDirWhichHasSubDirsNonEncrypted terminated")
}

//remove a directory with subdirectories inside mount; on remount check they are not there
func (ta *testAPI) removeDirWhichHasSubDirs(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("removeDirWhichHasSubDirs")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "rmsubdir-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "rmsubdir-mount")
	dat.files = make(map[string]fileInfo)
	dat.files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/three/2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/three/3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/four/5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/four/6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	dat.files["two/four/six/7.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	dirPath := filepath.Join(dat.testMountDir, "two")
	err = os.RemoveAll(dirPath)
	if err != nil {
		t.Fatalf("Error removing directory in mounted dir: %v", err)
	}

	//delete a directory inside the mounted dir with all its files
	mi, err2 := dat.swarmfs.Unmount(dat.testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v ", err2)
	}
	log.Debug("Directory unmounted")

	//we deleted files in the OS, so let's delete them also in the files map
	delete(dat.files, "two/three/2.txt")
	delete(dat.files, "two/three/3.txt")
	delete(dat.files, "two/four/5.txt")
	delete(dat.files, "two/four/6.txt")
	delete(dat.files, "two/four/six/7.txt")

	// mount again and see if things are okay
	testMountDir2, err3 := addDir(dat.testDir, "remount-mount2")
	if err3 != nil {
		t.Fatalf("Could not unmount %v", err3)
	}
	_ = mountDir(t, ta.api, dat.files, mi.LatestManifest, testMountDir2)
	log.Debug("Directory mounted again")
	actualPath := filepath.Join(dirPath, "three")
	actualPath = filepath.Join(actualPath, "2.txt")
	_, err = os.Stat(actualPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	actualPath = filepath.Join(dirPath, "four")
	_, err = os.Stat(actualPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	_, err = os.Stat(dirPath)
	if err == nil {
		t.Fatal("Expected file to not be present in re-mount after removal, but it is there")
	}
	_, err = dat.swarmfs.Unmount(testMountDir2)
	if err != nil {
		t.Fatalf("Could not unmount %v", err)
	}
	log.Debug("subtest terminated")
}

func (ta *testAPI) appendFileContentsToEndEncrypted(t *testing.T) {
	log.Debug("Starting appendFileContentsToEndEncrypted test")
	ta.appendFileContentsToEnd(t, true)
	log.Debug("Test appendFileContentsToEndEncrypted terminated")
}

func (ta *testAPI) appendFileContentsToEndNonEncrypted(t *testing.T) {
	log.Debug("Starting appendFileContentsToEndNonEncrypted test")
	ta.appendFileContentsToEnd(t, false)
	log.Debug("Test appendFileContentsToEndNonEncrypted terminated")
}

//append contents to the end of a file; remount and check it's intact
func (ta *testAPI) appendFileContentsToEnd(t *testing.T, toEncrypt bool) {
	dat, err := ta.initSubtest("appendFileContentsToEnd")
	if err != nil {
		t.Fatalf("Couldn't initialize subtest dirs: %v", err)
	}
	defer os.RemoveAll(dat.testDir)

	dat.toEncrypt = toEncrypt
	dat.testUploadDir = filepath.Join(dat.testDir, "appendlargefile-upload")
	dat.testMountDir = filepath.Join(dat.testDir, "appendlargefile-mount")
	dat.files = make(map[string]fileInfo)

	line1 := make([]byte, 10)
	_, err = rand.Read(line1)
	if err != nil {
		t.Fatalf("Error writing random bytes to byte array: %v", err)
	}

	dat.files["1.txt"] = fileInfo{0700, 333, 444, line1}

	dat, err = ta.uploadAndMount(dat, t)
	if err != nil {
		t.Fatalf("Error during upload of files to swarm / mount of swarm dir: %v", err)
	}
	defer dat.swarmfs.Stop()

	actualPath := filepath.Join(dat.testMountDir, "1.txt")
	fd, err4 := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	if err4 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err4)
	}
	defer fd.Close()
	log.Debug("file opened")
	line2 := make([]byte, 5)
	_, err = rand.Read(line2)
	if err != nil {
		t.Fatalf("Error writing random bytes to byte array: %v", err)
	}
	log.Debug("line read")
	_, err = fd.Seek(int64(len(line1)), 0)
	if err != nil {
		t.Fatalf("Error searching for position to append: %v", err)
	}
	_, err = fd.Write(line2)
	if err != nil {
		t.Fatalf("Error appending: %v", err)
	}
	log.Debug("line written")
	err = fd.Close()
	if err != nil {
		t.Fatalf("Error closing file: %v", err)
	}
	log.Debug("file closed")

	mi1, err5 := dat.swarmfs.Unmount(dat.testMountDir)
	if err5 != nil {
		t.Fatalf("Could not unmount %v ", err5)
	}
	log.Debug("Directory unmounted")

	// mount again and see if appended file is correct
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	dat.files["1.txt"] = fileInfo{0700, 333, 444, line1and2}
	testMountDir2, err6 := addDir(dat.testDir, "remount-mount2")
	if err6 != nil {
		t.Fatalf("Could not unmount %v", err6)
	}
	_ = mountDir(t, ta.api, dat.files, mi1.LatestManifest, testMountDir2)
	log.Debug("Directory mounted")

	checkFile(t, testMountDir2, "1.txt", line1and2)

	_, err = dat.swarmfs.Unmount(testMountDir2)
	if err != nil {
		t.Fatalf("Could not unmount %v", err)
	}
	log.Debug("subtest terminated")
}

//run all the tests
func TestFUSE(t *testing.T) {
	t.Skip("disable fuse tests until they are stable")
	//create a data directory for swarm
	datadir, err := ioutil.TempDir("", "fuse")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(datadir)

	fileStore, err := storage.NewLocalFileStore(datadir, make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ta := &testAPI{api: api.NewAPI(fileStore, nil, nil, nil)}

	//run a short suite of tests
	//approx time: 28s
	t.Run("mountListAndUnmountEncrypted", ta.mountListAndUnmountEncrypted)
	t.Run("remountEncrypted", ta.remountEncrypted)
	t.Run("unmountWhenResourceBusyNonEncrypted", ta.unmountWhenResourceBusyNonEncrypted)
	t.Run("removeExistingFileEncrypted", ta.removeExistingFileEncrypted)
	t.Run("addNewFileAndModifyContentsNonEncrypted", ta.addNewFileAndModifyContentsNonEncrypted)
	t.Run("removeDirWhichHasFilesNonEncrypted", ta.removeDirWhichHasFilesNonEncrypted)
	t.Run("appendFileContentsToEndEncrypted", ta.appendFileContentsToEndEncrypted)

	//provide longrunning flag to execute all tests
	//approx time with longrunning: 140s
	if *longrunning {
		t.Run("mountListAndUnmountNonEncrypted", ta.mountListAndUnmountNonEncrypted)
		t.Run("maxMountsEncrypted", ta.maxMountsEncrypted)
		t.Run("maxMountsNonEncrypted", ta.maxMountsNonEncrypted)
		t.Run("remountNonEncrypted", ta.remountNonEncrypted)
		t.Run("unmountEncrypted", ta.unmountEncrypted)
		t.Run("unmountNonEncrypted", ta.unmountNonEncrypted)
		t.Run("unmountWhenResourceBusyEncrypted", ta.unmountWhenResourceBusyEncrypted)
		t.Run("unmountWhenResourceBusyNonEncrypted", ta.unmountWhenResourceBusyNonEncrypted)
		t.Run("seekInMultiChunkFileEncrypted", ta.seekInMultiChunkFileEncrypted)
		t.Run("seekInMultiChunkFileNonEncrypted", ta.seekInMultiChunkFileNonEncrypted)
		t.Run("createNewFileEncrypted", ta.createNewFileEncrypted)
		t.Run("createNewFileNonEncrypted", ta.createNewFileNonEncrypted)
		t.Run("createNewFileInsideDirectoryEncrypted", ta.createNewFileInsideDirectoryEncrypted)
		t.Run("createNewFileInsideDirectoryNonEncrypted", ta.createNewFileInsideDirectoryNonEncrypted)
		t.Run("createNewFileInsideNewDirectoryEncrypted", ta.createNewFileInsideNewDirectoryEncrypted)
		t.Run("createNewFileInsideNewDirectoryNonEncrypted", ta.createNewFileInsideNewDirectoryNonEncrypted)
		t.Run("removeExistingFileNonEncrypted", ta.removeExistingFileNonEncrypted)
		t.Run("removeExistingFileInsideDirEncrypted", ta.removeExistingFileInsideDirEncrypted)
		t.Run("removeExistingFileInsideDirNonEncrypted", ta.removeExistingFileInsideDirNonEncrypted)
		t.Run("removeNewlyAddedFileEncrypted", ta.removeNewlyAddedFileEncrypted)
		t.Run("removeNewlyAddedFileNonEncrypted", ta.removeNewlyAddedFileNonEncrypted)
		t.Run("addNewFileAndModifyContentsEncrypted", ta.addNewFileAndModifyContentsEncrypted)
		t.Run("removeEmptyDirEncrypted", ta.removeEmptyDirEncrypted)
		t.Run("removeEmptyDirNonEncrypted", ta.removeEmptyDirNonEncrypted)
		t.Run("removeDirWhichHasFilesEncrypted", ta.removeDirWhichHasFilesEncrypted)
		t.Run("removeDirWhichHasSubDirsEncrypted", ta.removeDirWhichHasSubDirsEncrypted)
		t.Run("removeDirWhichHasSubDirsNonEncrypted", ta.removeDirWhichHasSubDirsNonEncrypted)
		t.Run("appendFileContentsToEndNonEncrypted", ta.appendFileContentsToEndNonEncrypted)
	}
}
