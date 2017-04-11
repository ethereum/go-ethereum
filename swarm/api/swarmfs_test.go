// Copyright 2016 The go-ethereum Authors
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

package api

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var testUploadDir, _ = ioutil.TempDir(os.TempDir(), "fuse-source")
var testMountDir, _ = ioutil.TempDir(os.TempDir(), "fuse-dest")

func testFuseFileSystem(t *testing.T, f func(*FileSystem)) {
	testApi(t, func(api *Api) {
		f(NewFileSystem(api))
	})
}

func createTestFiles(t *testing.T, files []string) {
	os.RemoveAll(testUploadDir)
	os.RemoveAll(testMountDir)
	defer os.MkdirAll(testMountDir, 0777)

	for f := range files {
		actualPath := filepath.Join(testUploadDir, files[f])
		filePath := filepath.Dir(actualPath)

		err := os.MkdirAll(filePath, 0777)
		if err != nil {
			t.Fatalf("Error creating directory '%v' : %v", filePath, err)
		}

		_, err1 := os.OpenFile(actualPath, os.O_RDONLY|os.O_CREATE, 0666)
		if err1 != nil {
			t.Fatalf("Error creating file %v: %v", actualPath, err1)
		}
	}

}

func compareFiles(t *testing.T, files []string) {
	for f := range files {
		sourceFile := filepath.Join(testUploadDir, files[f])
		destinationFile := filepath.Join(testMountDir, files[f])

		sfinfo, err := os.Stat(sourceFile)
		if err != nil {
			t.Fatalf("Source file %v missing in mount: %v", files[f], err)
		}

		dfinfo, err := os.Stat(destinationFile)
		if err != nil {
			t.Fatalf("Destination file %v missing in mount: %v", files[f], err)
		}

		if sfinfo.Size() != dfinfo.Size() {
			t.Fatalf("Size mismatch  source (%v) vs destination(%v)", sfinfo.Size(), dfinfo.Size())
		}

		if dfinfo.Mode().Perm().String() != "-r-x------" {
			t.Fatalf("Permission is not 0500for file: %v", err)
		}
	}
}

func doHashTest(fs *FileSystem, t *testing.T, ensName string, files ...string) {
	createTestFiles(t, files)
	bzzhash, err := fs.Upload(testUploadDir, "")
	if err != nil {
		t.Fatalf("Error uploading directory %v: %v", testUploadDir, err)
	}

	swarmfs := NewSwarmFS(fs.api)
	defer swarmfs.Stop()

	_, err = swarmfs.Mount(bzzhash, testMountDir)
	if isFUSEUnsupportedError(err) {
		t.Skip("FUSE not supported:", err)
	} else if err != nil {
		t.Fatalf("Error mounting hash %v: %v", bzzhash, err)
	}

	compareFiles(t, files)

	if _, err := swarmfs.Unmount(testMountDir); err != nil {
		t.Fatalf("Error unmounting path %v: %v", testMountDir, err)
	}
}

// mounting with manifest Hash
func TestFuseMountingScenarios(t *testing.T) {
	testFuseFileSystem(t, func(fs *FileSystem) {
		//doHashTest(fs,t, "test","1.txt")
		doHashTest(fs, t, "", "1.txt")
		doHashTest(fs, t, "", "1.txt", "11.txt", "111.txt", "two/2.txt", "two/two/2.txt", "three/3.txt")
		doHashTest(fs, t, "", "1/2/3/4/5/6/7/8/9/10/11/12/1.txt")
		doHashTest(fs, t, "", "one/one.txt", "one.txt", "once/one.txt", "one/one/one.txt")
	})
}
