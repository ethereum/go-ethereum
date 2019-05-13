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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/log"
)

var (
	errEmptyMountPoint      = errors.New("need non-empty mount point")
	errNoRelativeMountPoint = errors.New("invalid path for mount point (need absolute path)")
	errMaxMountCount        = errors.New("max FUSE mount count reached")
	errMountTimeout         = errors.New("mount timeout")
	errAlreadyMounted       = errors.New("mount point is already serving")
)

func isFUSEUnsupportedError(err error) bool {
	if perr, ok := err.(*os.PathError); ok {
		return perr.Op == "open" && perr.Path == "/dev/fuse"
	}
	return err == fuse.ErrOSXFUSENotFound
}

// MountInfo contains information about every active mount
type MountInfo struct {
	MountPoint     string
	StartManifest  string
	LatestManifest string
	rootDir        *SwarmDir
	fuseConnection *fuse.Conn
	swarmApi       *api.API
	lock           *sync.RWMutex
	serveClose     chan struct{}
}

func NewMountInfo(mhash, mpoint string, sapi *api.API) *MountInfo {
	log.Debug("swarmfs NewMountInfo", "hash", mhash, "mount point", mpoint)
	newMountInfo := &MountInfo{
		MountPoint:     mpoint,
		StartManifest:  mhash,
		LatestManifest: mhash,
		rootDir:        nil,
		fuseConnection: nil,
		swarmApi:       sapi,
		lock:           &sync.RWMutex{},
		serveClose:     make(chan struct{}),
	}
	return newMountInfo
}

func (swarmfs *SwarmFS) Mount(mhash, mountpoint string) (*MountInfo, error) {
	log.Info("swarmfs", "mounting hash", mhash, "mount point", mountpoint)
	if mountpoint == "" {
		return nil, errEmptyMountPoint
	}
	if !strings.HasPrefix(mountpoint, "/") {
		return nil, errNoRelativeMountPoint
	}
	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return nil, err
	}
	log.Trace("swarmfs mount", "cleanedMountPoint", cleanedMountPoint)

	swarmfs.swarmFsLock.Lock()
	defer swarmfs.swarmFsLock.Unlock()

	noOfActiveMounts := len(swarmfs.activeMounts)
	log.Debug("swarmfs mount", "# active mounts", noOfActiveMounts)
	if noOfActiveMounts >= maxFUSEMounts {
		return nil, errMaxMountCount
	}

	if _, ok := swarmfs.activeMounts[cleanedMountPoint]; ok {
		return nil, errAlreadyMounted
	}

	log.Trace("swarmfs mount: getting manifest tree")
	_, manifestEntryMap, err := swarmfs.swarmApi.BuildDirectoryTree(context.TODO(), mhash, true)
	if err != nil {
		return nil, err
	}

	log.Trace("swarmfs mount: building mount info")
	mi := NewMountInfo(mhash, cleanedMountPoint, swarmfs.swarmApi)

	dirTree := map[string]*SwarmDir{}
	rootDir := NewSwarmDir("/", mi)
	log.Trace("swarmfs mount", "rootDir", rootDir)
	mi.rootDir = rootDir

	log.Trace("swarmfs mount: traversing manifest map")
	for suffix, entry := range manifestEntryMap {
		if suffix == "" { //empty suffix means that the file has no name - i.e. this is the default entry in a manifest. Since we cannot have files without a name, let us ignore this entry
			log.Warn("Manifest has an empty-path (default) entry which will be ignored in FUSE mount.")
			continue
		}
		addr := common.Hex2Bytes(entry.Hash)
		fullpath := "/" + suffix
		basepath := filepath.Dir(fullpath)
		parentDir := rootDir
		dirUntilNow := ""
		paths := strings.Split(basepath, "/")
		for i := range paths {
			if paths[i] != "" {
				thisDir := paths[i]
				dirUntilNow = dirUntilNow + "/" + thisDir

				if _, ok := dirTree[dirUntilNow]; !ok {
					dirTree[dirUntilNow] = NewSwarmDir(dirUntilNow, mi)
					parentDir.directories = append(parentDir.directories, dirTree[dirUntilNow])
					parentDir = dirTree[dirUntilNow]

				} else {
					parentDir = dirTree[dirUntilNow]
				}
			}
		}
		thisFile := NewSwarmFile(basepath, filepath.Base(fullpath), mi)
		thisFile.addr = addr

		parentDir.files = append(parentDir.files, thisFile)
	}

	fconn, err := fuse.Mount(cleanedMountPoint, fuse.FSName("swarmfs"), fuse.VolumeName(mhash))
	if isFUSEUnsupportedError(err) {
		log.Error("swarmfs error - FUSE not installed", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err
	} else if err != nil {
		fuse.Unmount(cleanedMountPoint)
		log.Error("swarmfs error mounting swarm manifest", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err
	}
	mi.fuseConnection = fconn

	serverr := make(chan error, 1)
	go func() {
		log.Info("swarmfs", "serving hash", mhash, "at", cleanedMountPoint)
		filesys := &SwarmRoot{root: rootDir}
		//start serving the actual file system; see note below
		if err := fs.Serve(fconn, filesys); err != nil {
			log.Warn("swarmfs could not serve the requested hash", "error", err)
			serverr <- err
		}
		mi.serveClose <- struct{}{}
	}()

	/*
	   IMPORTANT NOTE: the fs.Serve function is blocking;
	   Serve builds up the actual fuse file system by calling the
	   Attr functions on each SwarmFile, creating the file inodes;
	   specifically calling the swarm's LazySectionReader.Size() to set the file size.

	   This can take some time, and it appears that if we access the fuse file system
	   too early, we can bring the tests to deadlock. The assumption so far is that
	   at this point, the fuse driver didn't finish to initialize the file system.

	   Accessing files too early not only deadlocks the tests, but locks the access
	   of the fuse file completely, resulting in blocked resources at OS system level.
	   Even a simple `ls /tmp/testDir/testMountDir` could deadlock in a shell.

	   Workaround so far is to wait some time to give the OS enough time to initialize
	   the fuse file system. During tests, this seemed to address the issue.

	   HOWEVER IT SHOULD BE NOTED THAT THIS MAY ONLY BE AN EFFECT,
	   AND THE DEADLOCK CAUSED BY SOMETHING ELSE BLOCKING ACCESS DUE TO SOME RACE CONDITION
	   (caused in the bazil.org library and/or the SwarmRoot, SwarmDir and SwarmFile implementations)
	*/
	time.Sleep(2 * time.Second)

	timer := time.NewTimer(mountTimeout)
	defer timer.Stop()
	// Check if the mount process has an error to report.
	select {
	case <-timer.C:
		log.Warn("swarmfs timed out mounting over FUSE", "mountpoint", cleanedMountPoint, "err", err)
		err := fuse.Unmount(cleanedMountPoint)
		if err != nil {
			return nil, err
		}
		return nil, errMountTimeout
	case err := <-serverr:
		log.Warn("swarmfs error serving over FUSE", "mountpoint", cleanedMountPoint, "err", err)
		err = fuse.Unmount(cleanedMountPoint)
		return nil, err

	case <-fconn.Ready:
		//this signals that the actual mount point from the fuse.Mount call is ready;
		//it does not signal though that the file system from fs.Serve is actually fully built up
		if err := fconn.MountError; err != nil {
			log.Error("Mounting error from fuse driver: ", "err", err)
			return nil, err
		}
		log.Info("swarmfs now served over FUSE", "manifest", mhash, "mountpoint", cleanedMountPoint)
	}

	timer.Stop()
	swarmfs.activeMounts[cleanedMountPoint] = mi
	return mi, nil
}

func (swarmfs *SwarmFS) Unmount(mountpoint string) (*MountInfo, error) {
	swarmfs.swarmFsLock.Lock()
	defer swarmfs.swarmFsLock.Unlock()

	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return nil, err
	}

	mountInfo := swarmfs.activeMounts[cleanedMountPoint]

	if mountInfo == nil || mountInfo.MountPoint != cleanedMountPoint {
		return nil, fmt.Errorf("swarmfs %s is not mounted", cleanedMountPoint)
	}
	err = fuse.Unmount(cleanedMountPoint)
	if err != nil {
		err1 := externalUnmount(cleanedMountPoint)
		if err1 != nil {
			errStr := fmt.Sprintf("swarmfs unmount error: %v", err)
			log.Warn(errStr)
			return nil, err1
		}
	}

	err = mountInfo.fuseConnection.Close()
	if err != nil {
		return nil, err
	}
	delete(swarmfs.activeMounts, cleanedMountPoint)

	<-mountInfo.serveClose

	succString := fmt.Sprintf("swarmfs unmounting %v succeeded", cleanedMountPoint)
	log.Info(succString)

	return mountInfo, nil
}

func (swarmfs *SwarmFS) Listmounts() []*MountInfo {
	swarmfs.swarmFsLock.RLock()
	defer swarmfs.swarmFsLock.RUnlock()
	rows := make([]*MountInfo, 0, len(swarmfs.activeMounts))
	for _, mi := range swarmfs.activeMounts {
		rows = append(rows, mi)
	}
	return rows
}

func (swarmfs *SwarmFS) Stop() bool {
	for mp := range swarmfs.activeMounts {
		mountInfo := swarmfs.activeMounts[mp]
		swarmfs.Unmount(mountInfo.MountPoint)
	}
	return true
}
