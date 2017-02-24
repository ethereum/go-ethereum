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
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"os/exec"
	"runtime"
	"time"
)

func externalUnMount(mountPoint string) error {

	var cmd *exec.Cmd

	switch runtime.GOOS {

	case "darwin":
		cmd = exec.Command("/usr/bin/diskutil", "umount", "force", mountPoint)

	case "linux":
		cmd = exec.Command("fusermount", "-u", mountPoint)

	default:
		return fmt.Errorf("unmount: unimplemented")
	}

	errc := make(chan error, 1)
	go func() {
		defer close(errc)

		if err := exec.Command("umount", mountPoint).Run(); err == nil {
			return
		}
		errc <- cmd.Run()
	}()

	select {

	case <-time.After(unmountTimeout):
		return fmt.Errorf("umount timeout")

	case err := <-errc:
		return err
	}
}

func addFileToSwarm(sf *SwarmFile, content []byte, size int) error {

	fkey, mhash, err := sf.mountInfo.swarmApi.AddFile(sf.mountInfo.LatestManifest, sf.path, sf.name, content, true)
	if err != nil {
		return err
	}

	sf.lock.Lock()
	defer sf.lock.Unlock()
	sf.key = fkey
	sf.fileSize = int64(size)

	sf.mountInfo.lock.Lock()
	defer sf.mountInfo.lock.Unlock()
	sf.mountInfo.LatestManifest = mhash

	log.Info("Added new file:", "fname", sf.name, "New Manifest hash", mhash)
	return nil

}

func removeFileFromSwarm(sf *SwarmFile) error {

	mkey, err := sf.mountInfo.swarmApi.RemoveFile(sf.mountInfo.LatestManifest, sf.path, sf.name, true)
	if err != nil {
		return err
	}

	sf.mountInfo.lock.Lock()
	defer sf.mountInfo.lock.Unlock()
	sf.mountInfo.LatestManifest = mkey

	log.Info("Removed file:", "fname", sf.name, "New Manifest hash", mkey)
	return nil
}

func removeDirectoryFromSwarm(sd *SwarmDir) error {

	if len(sd.directories) == 0 && len(sd.files) == 0 {
		return nil
	}

	for _, d := range sd.directories {
		err := removeDirectoryFromSwarm(d)
		if err != nil {
			return err
		}
	}

	for _, f := range sd.files {
		err := removeFileFromSwarm(f)
		if err != nil {
			return err
		}
	}

	return nil

}

func appendToExistingFileInSwarm(sf *SwarmFile, content []byte, offset int64, length int64) error {

	fkey, mhash, err := sf.mountInfo.swarmApi.AppendFile(sf.mountInfo.LatestManifest, sf.path, sf.name, sf.fileSize, content, sf.key, offset, length, true)
	if err != nil {
		return err
	}

	sf.lock.Lock()
	defer sf.lock.Unlock()
	sf.key = fkey
	sf.fileSize = sf.fileSize + int64(len(content))

	sf.mountInfo.lock.Lock()
	defer sf.mountInfo.lock.Unlock()
	sf.mountInfo.LatestManifest = mhash

	log.Info("Appended file:", "fname", sf.name, "New Manifest hash", mhash)
	return nil

}
