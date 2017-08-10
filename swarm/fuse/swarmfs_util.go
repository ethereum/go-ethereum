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
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/ethereum/go-ethereum/log"
)

func externalUnmount(mountPoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), unmountTimeout)
	defer cancel()

	// Try generic umount.
	if err := exec.CommandContext(ctx, "umount", mountPoint).Run(); err == nil {
		return nil
	}
	// Try FUSE-specific commands if umount didn't work.
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "diskutil", "umount", "force", mountPoint).Run()
	case "linux":
		return exec.CommandContext(ctx, "fusermount", "-u", mountPoint).Run()
	default:
		return fmt.Errorf("unmount: unimplemented")
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
