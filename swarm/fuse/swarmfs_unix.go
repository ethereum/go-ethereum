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
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
)

var (
	errEmptyMountPoint = errors.New("need non-empty mount point")
	errMaxMountCount   = errors.New("max FUSE mount count reached")
	errMountTimeout    = errors.New("mount timeout")
	errAlreadyMounted  = errors.New("mount point is already serving")
)

func isFUSEUnsupportedError(err error) bool {
	if perr, ok := err.(*os.PathError); ok {
		return perr.Op == "open" && perr.Path == "/dev/fuse"
	}
	return err == fuse.ErrOSXFUSENotFound
}

// information about every active mount
type MountInfo struct {
	MountPoint     string
	StartManifest  string
	LatestManifest string
	rootDir        *SwarmDir
	fuseConnection *fuse.Conn
	swarmApi       *api.Api
	lock           *sync.RWMutex
}

// Inode numbers need to be unique, they are used for caching inside fuse
func newInode() uint64 {
	inodeLock.Lock()
	defer inodeLock.Unlock()
	inode += 1
	return inode
}

func NewMountInfo(mhash, mpoint string, sapi *api.Api) *MountInfo {
	newMountInfo := &MountInfo{
		MountPoint:     mpoint,
		StartManifest:  mhash,
		LatestManifest: mhash,
		rootDir:        nil,
		fuseConnection: nil,
		swarmApi:       sapi,
		lock:           &sync.RWMutex{},
	}
	return newMountInfo
}

func (self *SwarmFS) Mount(mhash, mountpoint string) (*MountInfo, error) {

	if mountpoint == "" {
		return nil, errEmptyMountPoint
	}
	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return nil, err
	}

	self.swarmFsLock.Lock()
	defer self.swarmFsLock.Unlock()

	noOfActiveMounts := len(self.activeMounts)
	if noOfActiveMounts >= maxFuseMounts {
		return nil, errMaxMountCount
	}

	if _, ok := self.activeMounts[cleanedMountPoint]; ok {
		return nil, errAlreadyMounted
	}

	log.Info(fmt.Sprintf("Attempting to mount %s ", cleanedMountPoint))
	key, manifestEntryMap, err := self.swarmApi.BuildDirectoryTree(mhash, true)
	if err != nil {
		return nil, err
	}

	mi := NewMountInfo(mhash, cleanedMountPoint, self.swarmApi)

	dirTree := map[string]*SwarmDir{}
	rootDir := NewSwarmDir("/", mi)
	dirTree["/"] = rootDir
	mi.rootDir = rootDir

	for suffix, entry := range manifestEntryMap {

		key = common.Hex2Bytes(entry.Hash)
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
		thisFile.key = key

		parentDir.files = append(parentDir.files, thisFile)
	}

	fconn, err := fuse.Mount(cleanedMountPoint, fuse.FSName("swarmfs"), fuse.VolumeName(mhash))
	if isFUSEUnsupportedError(err) {
		log.Warn("Fuse not installed", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err
	} else if err != nil {
		fuse.Unmount(cleanedMountPoint)
		log.Warn("Error mounting swarm manifest", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err
	}
	mi.fuseConnection = fconn

	serverr := make(chan error, 1)
	go func() {
		log.Info(fmt.Sprintf("Serving %s at %s", mhash, cleanedMountPoint))
		filesys := &SwarmRoot{root: rootDir}
		if err := fs.Serve(fconn, filesys); err != nil {
			log.Warn(fmt.Sprintf("Could not Serve SwarmFileSystem error: %v", err))
			serverr <- err
		}

	}()

	// Check if the mount process has an error to report.
	select {
	case <-time.After(mountTimeout):
		fuse.Unmount(cleanedMountPoint)
		return nil, errMountTimeout

	case err := <-serverr:
		fuse.Unmount(cleanedMountPoint)
		log.Warn("Error serving swarm FUSE FS", "mountpoint", cleanedMountPoint, "err", err)
		return nil, err

	case <-fconn.Ready:
		log.Info("Now serving swarm FUSE FS", "manifest", mhash, "mountpoint", cleanedMountPoint)
	}

	self.activeMounts[cleanedMountPoint] = mi
	return mi, nil
}

func (self *SwarmFS) Unmount(mountpoint string) (*MountInfo, error) {

	self.swarmFsLock.Lock()
	defer self.swarmFsLock.Unlock()

	cleanedMountPoint, err := filepath.Abs(filepath.Clean(mountpoint))
	if err != nil {
		return nil, err
	}

	mountInfo := self.activeMounts[cleanedMountPoint]

	if mountInfo == nil || mountInfo.MountPoint != cleanedMountPoint {
		return nil, fmt.Errorf("%s is not mounted", cleanedMountPoint)
	}
	err = fuse.Unmount(cleanedMountPoint)
	if err != nil {
		err1 := externalUnmount(cleanedMountPoint)
		if err1 != nil {
			errStr := fmt.Sprintf("UnMount error: %v", err)
			log.Warn(errStr)
			return nil, err1
		}
	}

	mountInfo.fuseConnection.Close()
	delete(self.activeMounts, cleanedMountPoint)

	succString := fmt.Sprintf("UnMounting %v succeeded", cleanedMountPoint)
	log.Info(succString)

	return mountInfo, nil
}

func (self *SwarmFS) Listmounts() []*MountInfo {
	self.swarmFsLock.RLock()
	defer self.swarmFsLock.RUnlock()

	rows := make([]*MountInfo, 0, len(self.activeMounts))
	for _, mi := range self.activeMounts {
		rows = append(rows, mi)
	}
	return rows
}

func (self *SwarmFS) Stop() bool {
	for mp := range self.activeMounts {
		mountInfo := self.activeMounts[mp]
		self.Unmount(mountInfo.MountPoint)
	}
	return true
}
