/*
 * Copyright 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package badger

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/options"

	"github.com/dgraph-io/badger/y"
	"github.com/pkg/errors"
	"golang.org/x/net/trace"
)

// Values have their first byte being byteData or byteDelete. This helps us distinguish between
// a key that has never been seen and a key that has been explicitly deleted.
const (
	bitDelete       byte = 1 << 0 // Set if the key has been deleted.
	bitValuePointer byte = 1 << 1 // Set if the value is NOT stored directly next to key.

	// The MSB 2 bits are for transactions.
	bitTxn    byte = 1 << 6 // Set if the entry is part of a txn.
	bitFinTxn byte = 1 << 7 // Set if the entry is to indicate end of txn in value log.

	mi int64 = 1 << 20
)

type logFile struct {
	path string
	// This is a lock on the log file. It guards the fd’s value, the file’s
	// existence and the file’s memory map.
	//
	// Use shared ownership when reading/writing the file or memory map, use
	// exclusive ownership to open/close the descriptor, unmap or remove the file.
	lock        sync.RWMutex
	fd          *os.File
	fid         uint32
	fmap        []byte
	size        uint32
	loadingMode options.FileLoadingMode
}

// openReadOnly assumes that we have a write lock on logFile.
func (lf *logFile) openReadOnly() error {
	var err error
	lf.fd, err = os.OpenFile(lf.path, os.O_RDONLY, 0666)
	if err != nil {
		return errors.Wrapf(err, "Unable to open %q as RDONLY.", lf.path)
	}

	fi, err := lf.fd.Stat()
	if err != nil {
		return errors.Wrapf(err, "Unable to check stat for %q", lf.path)
	}
	lf.size = uint32(fi.Size())

	if err = lf.mmap(fi.Size()); err != nil {
		_ = lf.fd.Close()
		return y.Wrapf(err, "Unable to map file")
	}

	return nil
}

func (lf *logFile) mmap(size int64) (err error) {
	if lf.loadingMode != options.MemoryMap {
		// Nothing to do
		return nil
	}
	lf.fmap, err = y.Mmap(lf.fd, false, size)
	if err == nil {
		err = y.Madvise(lf.fmap, false) // Disable readahead
	}
	return err
}

func (lf *logFile) munmap() (err error) {
	if lf.loadingMode != options.MemoryMap {
		// Nothing to do
		return nil
	}
	if err := y.Munmap(lf.fmap); err != nil {
		return errors.Wrapf(err, "Unable to munmap value log: %q", lf.path)
	}
	return nil
}

// Acquire lock on mmap/file if you are calling this
func (lf *logFile) read(p valuePointer, s *y.Slice) (buf []byte, err error) {
	var nbr int64
	offset := p.Offset
	if lf.loadingMode == options.FileIO {
		buf = s.Resize(int(p.Len))
		var n int
		n, err = lf.fd.ReadAt(buf, int64(offset))
		nbr = int64(n)
	} else {
		size := uint32(len(lf.fmap))
		valsz := p.Len
		if offset >= size || offset+valsz > size {
			err = y.ErrEOF
		} else {
			buf = lf.fmap[offset : offset+valsz]
			nbr = int64(valsz)
		}
	}
	y.NumReads.Add(1)
	y.NumBytesRead.Add(nbr)
	return buf, err
}

func (lf *logFile) doneWriting(offset uint32) error {
	// Sync before acquiring lock.  (We call this from write() and thus know we have shared access
	// to the fd.)
	if err := lf.fd.Sync(); err != nil {
		return errors.Wrapf(err, "Unable to sync value log: %q", lf.path)
	}
	// Close and reopen the file read-only.  Acquire lock because fd will become invalid for a bit.
	// Acquiring the lock is bad because, while we don't hold the lock for a long time, it forces
	// one batch of readers wait for the preceding batch of readers to finish.
	//
	// If there's a benefit to reopening the file read-only, it might be on Windows.  I don't know
	// what the benefit is.  Consider keeping the file read-write, or use fcntl to change
	// permissions.
	lf.lock.Lock()
	defer lf.lock.Unlock()
	if err := lf.munmap(); err != nil {
		return err
	}
	// TODO: Confirm if we need to run a file sync after truncation.
	// Truncation must run after unmapping, otherwise Windows would crap itself.
	if err := lf.fd.Truncate(int64(offset)); err != nil {
		return errors.Wrapf(err, "Unable to truncate file: %q", lf.path)
	}
	if err := lf.fd.Close(); err != nil {
		return errors.Wrapf(err, "Unable to close value log: %q", lf.path)
	}

	return lf.openReadOnly()
}

// You must hold lf.lock to sync()
func (lf *logFile) sync() error {
	return lf.fd.Sync()
}

var errStop = errors.New("Stop iteration")

type logEntry func(e Entry, vp valuePointer) error

// iterate iterates over log file. It doesn't not allocate new memory for every kv pair.
// Therefore, the kv pair is only valid for the duration of fn call.
func (vlog *valueLog) iterate(lf *logFile, offset uint32, fn logEntry) error {
	_, err := lf.fd.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return y.Wrap(err)
	}

	reader := bufio.NewReader(lf.fd)
	var hbuf [headerBufSize]byte
	var h header
	k := make([]byte, 1<<10)
	v := make([]byte, 1<<20)

	truncate := false
	recordOffset := offset
	for {
		hash := crc32.New(y.CastagnoliCrcTable)
		tee := io.TeeReader(reader, hash)

		// TODO: Move this entry decode into structs.go
		if _, err = io.ReadFull(tee, hbuf[:]); err != nil {
			if err == io.EOF {
				break
			} else if err == io.ErrUnexpectedEOF {
				truncate = true
				break
			}
			return err
		}

		var e Entry
		e.offset = recordOffset
		h.Decode(hbuf[:])
		if h.klen > maxKeySize {
			truncate = true
			break
		}
		vl := int(h.vlen)
		if cap(v) < vl {
			v = make([]byte, 2*vl)
		}

		kl := int(h.klen)
		if cap(k) < kl {
			k = make([]byte, 2*kl)
		}
		e.Key = k[:kl]
		e.Value = v[:vl]

		if _, err = io.ReadFull(tee, e.Key); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				truncate = true
				break
			}
			return err
		}
		if _, err = io.ReadFull(tee, e.Value); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				truncate = true
				break
			}
			return err
		}

		var crcBuf [4]byte
		if _, err = io.ReadFull(reader, crcBuf[:]); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				truncate = true
				break
			}
			return err
		}
		crc := binary.BigEndian.Uint32(crcBuf[:])
		if crc != hash.Sum32() {
			truncate = true
			break
		}
		e.meta = h.meta
		e.UserMeta = h.userMeta
		e.ExpiresAt = h.expiresAt

		var vp valuePointer
		vp.Len = headerBufSize + h.klen + h.vlen + uint32(len(crcBuf))
		recordOffset += vp.Len

		vp.Offset = e.offset
		vp.Fid = lf.fid

		if err := fn(e, vp); err != nil {
			if err == errStop {
				break
			}
			return y.Wrap(err)
		}
	}

	if truncate && len(lf.fmap) == 0 {
		// Only truncate if the file isn't mmaped. Otherwise, Windows would puke.
		if err := lf.fd.Truncate(int64(recordOffset)); err != nil {
			return err
		}
	}

	return nil
}

func (vlog *valueLog) rewrite(f *logFile) error {
	maxFid := atomic.LoadUint32(&vlog.maxFid)
	y.AssertTruef(uint32(f.fid) < maxFid, "fid to move: %d. Current max fid: %d", f.fid, maxFid)

	elog := trace.NewEventLog("badger", "vlog-rewrite")
	defer elog.Finish()
	elog.Printf("Rewriting fid: %d", f.fid)

	wb := make([]*Entry, 0, 1000)
	var size int64

	y.AssertTrue(vlog.kv != nil)
	var count int
	fe := func(e Entry) error {
		count++
		if count%10000 == 0 {
			elog.Printf("Processing entry %d", count)
		}

		vs, err := vlog.kv.get(e.Key)
		if err != nil {
			return err
		}
		if discardEntry(e, vs) {
			return nil
		}

		// Value is still present in value log.
		if len(vs.Value) == 0 {
			return errors.Errorf("Empty value: %+v", vs)
		}
		var vp valuePointer
		vp.Decode(vs.Value)

		if vp.Fid > f.fid {
			return nil
		}
		if vp.Offset > e.offset {
			return nil
		}
		if vp.Fid == f.fid && vp.Offset == e.offset {
			// This new entry only contains the key, and a pointer to the value.
			ne := new(Entry)
			ne.meta = 0 // Remove all bits.
			ne.UserMeta = e.UserMeta
			ne.Key = make([]byte, len(e.Key))
			copy(ne.Key, e.Key)
			ne.Value = make([]byte, len(e.Value))
			copy(ne.Value, e.Value)
			wb = append(wb, ne)
			size += int64(e.estimateSize(vlog.opt.ValueThreshold))
			if size >= 64*mi {
				elog.Printf("request has %d entries, size %d", len(wb), size)
				if err := vlog.kv.batchSet(wb); err != nil {
					return err
				}
				size = 0
				wb = wb[:0]
			}
		} else {
			log.Printf("WARNING: This entry should have been caught. %+v\n", e)
		}
		return nil
	}

	err := vlog.iterate(f, 0, func(e Entry, vp valuePointer) error {
		return fe(e)
	})
	if err != nil {
		return err
	}

	elog.Printf("request has %d entries, size %d", len(wb), size)
	batchSize := 1024
	var loops int
	for i := 0; i < len(wb); {
		loops++
		if batchSize == 0 {
			log.Printf("WARNING: We shouldn't reach batch size of zero.")
			return ErrNoRewrite
		}
		end := i + batchSize
		if end > len(wb) {
			end = len(wb)
		}
		if err := vlog.kv.batchSet(wb[i:end]); err != nil {
			if err == ErrTxnTooBig {
				// Decrease the batch size to half.
				batchSize = batchSize / 2
				elog.Printf("Dropped batch size to %d", batchSize)
				continue
			}
			return err
		}
		i += batchSize
	}
	elog.Printf("Processed %d entries in %d loops", len(wb), loops)

	elog.Printf("Removing fid: %d", f.fid)
	var deleteFileNow bool
	// Entries written to LSM. Remove the older file now.
	{
		vlog.filesLock.Lock()
		// Just a sanity-check.
		if _, ok := vlog.filesMap[f.fid]; !ok {
			vlog.filesLock.Unlock()
			return errors.Errorf("Unable to find fid: %d", f.fid)
		}
		if vlog.numActiveIterators == 0 {
			delete(vlog.filesMap, f.fid)
			deleteFileNow = true
		} else {
			vlog.filesToBeDeleted = append(vlog.filesToBeDeleted, f.fid)
		}
		vlog.filesLock.Unlock()
	}

	if deleteFileNow {
		vlog.deleteLogFile(f)
	}

	return nil
}

func (vlog *valueLog) incrIteratorCount() {
	atomic.AddInt32(&vlog.numActiveIterators, 1)
}

func (vlog *valueLog) decrIteratorCount() error {
	num := atomic.AddInt32(&vlog.numActiveIterators, -1)
	if num != 0 {
		return nil
	}

	vlog.filesLock.Lock()
	lfs := make([]*logFile, 0, len(vlog.filesToBeDeleted))
	for _, id := range vlog.filesToBeDeleted {
		lfs = append(lfs, vlog.filesMap[id])
		delete(vlog.filesMap, id)
	}
	vlog.filesToBeDeleted = nil
	vlog.filesLock.Unlock()

	for _, lf := range lfs {
		if err := vlog.deleteLogFile(lf); err != nil {
			return err
		}
	}
	return nil
}

func (vlog *valueLog) deleteLogFile(lf *logFile) error {
	path := vlog.fpath(lf.fid)
	if err := lf.munmap(); err != nil {
		_ = lf.fd.Close()
		return err
	}
	if err := lf.fd.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

// lfDiscardStats keeps track of the amount of data that could be discarded for
// a given logfile.
type lfDiscardStats struct {
	sync.Mutex
	m map[uint32]int64
}

type valueLog struct {
	buf     bytes.Buffer
	dirPath string
	elog    trace.EventLog

	// guards our view of which files exist, which to be deleted, how many active iterators
	filesLock        sync.RWMutex
	filesMap         map[uint32]*logFile
	filesToBeDeleted []uint32
	// A refcount of iterators -- when this hits zero, we can delete the filesToBeDeleted.
	numActiveIterators int32

	kv                *DB
	maxFid            uint32
	writableLogOffset uint32
	opt               Options

	garbageCh      chan struct{}
	lfDiscardStats *lfDiscardStats
}

func vlogFilePath(dirPath string, fid uint32) string {
	return fmt.Sprintf("%s%s%06d.vlog", dirPath, string(os.PathSeparator), fid)
}

func (vlog *valueLog) fpath(fid uint32) string {
	return vlogFilePath(vlog.dirPath, fid)
}

func (vlog *valueLog) openOrCreateFiles() error {
	files, err := ioutil.ReadDir(vlog.dirPath)
	if err != nil {
		return errors.Wrapf(err, "Error while opening value log")
	}

	found := make(map[uint64]struct{})
	var maxFid uint32 // Beware len(files) == 0 case, this starts at 0.
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".vlog") {
			continue
		}
		fsz := len(file.Name())
		fid, err := strconv.ParseUint(file.Name()[:fsz-5], 10, 32)
		if err != nil {
			return errors.Wrapf(err, "Error while parsing value log id for file: %q", file.Name())
		}
		if _, ok := found[fid]; ok {
			return errors.Errorf("Found the same value log file twice: %d", fid)
		}
		found[fid] = struct{}{}

		lf := &logFile{
			fid:         uint32(fid),
			path:        vlog.fpath(uint32(fid)),
			loadingMode: vlog.opt.ValueLogLoadingMode,
		}
		vlog.filesMap[uint32(fid)] = lf
		if uint32(fid) > maxFid {
			maxFid = uint32(fid)
		}
	}
	vlog.maxFid = uint32(maxFid)

	// Open all previous log files as read only. Open the last log file
	// as read write.
	for fid, lf := range vlog.filesMap {
		if fid == maxFid {
			if lf.fd, err = y.OpenExistingSyncedFile(vlog.fpath(fid),
				vlog.opt.SyncWrites); err != nil {
				return errors.Wrapf(err, "Unable to open value log file as RDWR")
			}
		} else {
			if err := lf.openReadOnly(); err != nil {
				return err
			}
		}
	}

	// If no files are found, then create a new file.
	if len(vlog.filesMap) == 0 {
		// We already set vlog.maxFid above
		_, err := vlog.createVlogFile(0)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vlog *valueLog) createVlogFile(fid uint32) (*logFile, error) {
	path := vlog.fpath(fid)
	lf := &logFile{fid: fid, path: path, loadingMode: vlog.opt.ValueLogLoadingMode}
	vlog.writableLogOffset = 0

	var err error
	if lf.fd, err = y.CreateSyncedFile(path, vlog.opt.SyncWrites); err != nil {
		return nil, errors.Wrapf(err, "Unable to create value log file")
	}

	if err = syncDir(vlog.dirPath); err != nil {
		return nil, errors.Wrapf(err, "Unable to sync value log file dir")
	}

	vlog.filesLock.Lock()
	vlog.filesMap[fid] = lf
	vlog.filesLock.Unlock()

	return lf, nil
}

func (vlog *valueLog) Open(kv *DB, opt Options) error {
	vlog.dirPath = opt.ValueDir
	vlog.opt = opt
	vlog.kv = kv
	vlog.filesMap = make(map[uint32]*logFile)
	if err := vlog.openOrCreateFiles(); err != nil {
		return errors.Wrapf(err, "Unable to open value log")
	}

	vlog.elog = trace.NewEventLog("Badger", "Valuelog")
	vlog.garbageCh = make(chan struct{}, 1) // Only allow one GC at a time.
	vlog.lfDiscardStats = &lfDiscardStats{m: make(map[uint32]int64)}
	return nil
}

func (vlog *valueLog) Close() error {
	vlog.elog.Printf("Stopping garbage collection of values.")
	defer vlog.elog.Finish()

	var err error
	for id, f := range vlog.filesMap {

		f.lock.Lock() // We won’t release the lock.
		if munmapErr := f.munmap(); munmapErr != nil && err == nil {
			err = munmapErr
		}

		if id == vlog.maxFid {
			// truncate writable log file to correct offset.
			if truncErr := f.fd.Truncate(
				int64(vlog.writableLogOffset)); truncErr != nil && err == nil {
				err = truncErr
			}
		}

		if closeErr := f.fd.Close(); closeErr != nil && err == nil {
			err = closeErr
		}

	}
	return err
}

// sortedFids returns the file id's not pending deletion, sorted.  Assumes we have shared access to
// filesMap.
func (vlog *valueLog) sortedFids() []uint32 {
	toBeDeleted := make(map[uint32]struct{})
	for _, fid := range vlog.filesToBeDeleted {
		toBeDeleted[fid] = struct{}{}
	}
	ret := make([]uint32, 0, len(vlog.filesMap))
	for fid := range vlog.filesMap {
		if _, ok := toBeDeleted[fid]; !ok {
			ret = append(ret, fid)
		}
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

// Replay replays the value log. The kv provided is only valid for the lifetime of function call.
func (vlog *valueLog) Replay(ptr valuePointer, fn logEntry) error {
	fid := ptr.Fid
	offset := ptr.Offset + ptr.Len
	vlog.elog.Printf("Seeking at value pointer: %+v\n", ptr)

	fids := vlog.sortedFids()

	for _, id := range fids {
		if id < fid {
			continue
		}
		of := offset
		if id > fid {
			of = 0
		}
		f := vlog.filesMap[id]
		err := vlog.iterate(f, of, fn)
		if err != nil {
			return errors.Wrapf(err, "Unable to replay value log: %q", f.path)
		}
	}

	// Seek to the end to start writing.
	var err error
	last := vlog.filesMap[vlog.maxFid]
	lastOffset, err := last.fd.Seek(0, io.SeekEnd)
	atomic.AddUint32(&vlog.writableLogOffset, uint32(lastOffset))
	return errors.Wrapf(err, "Unable to seek to end of value log: %q", last.path)
}

type request struct {
	// Input values
	Entries []*Entry
	// Output values and wait group stuff below
	Ptrs []valuePointer
	Wg   sync.WaitGroup
	Err  error
}

func (req *request) Wait() error {
	req.Wg.Wait()
	req.Entries = nil
	err := req.Err
	requestPool.Put(req)
	return err
}

// sync is thread-unsafe and should not be called concurrently with write.
func (vlog *valueLog) sync() error {
	if vlog.opt.SyncWrites {
		return nil
	}

	vlog.filesLock.RLock()
	if len(vlog.filesMap) == 0 {
		vlog.filesLock.RUnlock()
		return nil
	}
	curlf := vlog.filesMap[vlog.maxFid]
	curlf.lock.RLock()
	vlog.filesLock.RUnlock()

	dirSyncCh := make(chan error)
	go func() { dirSyncCh <- syncDir(vlog.opt.ValueDir) }()
	err := curlf.sync()
	curlf.lock.RUnlock()
	dirSyncErr := <-dirSyncCh
	if err != nil {
		err = dirSyncErr
	}
	return err
}

func (vlog *valueLog) writableOffset() uint32 {
	return atomic.LoadUint32(&vlog.writableLogOffset)
}

// write is thread-unsafe by design and should not be called concurrently.
func (vlog *valueLog) write(reqs []*request) error {
	vlog.filesLock.RLock()
	curlf := vlog.filesMap[vlog.maxFid]
	vlog.filesLock.RUnlock()

	toDisk := func() error {
		if vlog.buf.Len() == 0 {
			return nil
		}
		vlog.elog.Printf("Flushing %d blocks of total size: %d", len(reqs), vlog.buf.Len())
		n, err := curlf.fd.Write(vlog.buf.Bytes())
		if err != nil {
			return errors.Wrapf(err, "Unable to write to value log file: %q", curlf.path)
		}
		y.NumWrites.Add(1)
		y.NumBytesWritten.Add(int64(n))
		vlog.elog.Printf("Done")
		atomic.AddUint32(&vlog.writableLogOffset, uint32(n))
		vlog.buf.Reset()

		if vlog.writableOffset() > uint32(vlog.opt.ValueLogFileSize) {
			var err error
			if err = curlf.doneWriting(vlog.writableLogOffset); err != nil {
				return err
			}

			newid := atomic.AddUint32(&vlog.maxFid, 1)
			y.AssertTruef(newid <= math.MaxUint32, "newid will overflow uint32: %v", newid)
			newlf, err := vlog.createVlogFile(newid)
			if err != nil {
				return err
			}

			if err = newlf.mmap(2 * vlog.opt.ValueLogFileSize); err != nil {
				return err
			}

			curlf = newlf
		}
		return nil
	}

	for i := range reqs {
		b := reqs[i]
		b.Ptrs = b.Ptrs[:0]
		for j := range b.Entries {
			e := b.Entries[j]
			var p valuePointer

			p.Fid = curlf.fid
			// Use the offset including buffer length so far.
			p.Offset = vlog.writableOffset() + uint32(vlog.buf.Len())
			plen, err := encodeEntry(e, &vlog.buf) // Now encode the entry into buffer.
			if err != nil {
				return err
			}
			p.Len = uint32(plen)
			b.Ptrs = append(b.Ptrs, p)

			if p.Offset > uint32(vlog.opt.ValueLogFileSize) {
				if err := toDisk(); err != nil {
					return err
				}
			}
		}
	}
	return toDisk()

	// Acquire mutex locks around this manipulation, so that the reads don't try to use
	// an invalid file descriptor.
}

// Gets the logFile and acquires and RLock() for the mmap. You must call RUnlock on the file
// (if non-nil)
func (vlog *valueLog) getFileRLocked(fid uint32) (*logFile, error) {
	vlog.filesLock.RLock()
	defer vlog.filesLock.RUnlock()
	ret, ok := vlog.filesMap[fid]
	if !ok {
		// log file has gone away, will need to retry the operation.
		return nil, ErrRetry
	}
	ret.lock.RLock()
	return ret, nil
}

// Read reads the value log at a given location.
// TODO: Make this read private.
func (vlog *valueLog) Read(vp valuePointer, s *y.Slice) ([]byte, func(), error) {
	// Check for valid offset if we are reading to writable log.
	if vp.Fid == vlog.maxFid && vp.Offset >= vlog.writableOffset() {
		return nil, nil, errors.Errorf(
			"Invalid value pointer offset: %d greater than current offset: %d",
			vp.Offset, vlog.writableOffset())
	}

	buf, cb, err := vlog.readValueBytes(vp, s)
	if err != nil {
		return nil, cb, err
	}
	var h header
	h.Decode(buf)
	n := uint32(headerBufSize) + h.klen
	return buf[n : n+h.vlen], cb, nil
}

func (vlog *valueLog) readValueBytes(vp valuePointer, s *y.Slice) ([]byte, func(), error) {
	lf, err := vlog.getFileRLocked(vp.Fid)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "Unable to read from value log: %+v", vp)
	}

	buf, err := lf.read(vp, s)
	if vlog.opt.ValueLogLoadingMode == options.MemoryMap {
		return buf, lf.lock.RUnlock, err
	}
	// If we are using File I/O we unlock the file immediately
	// and return an empty function as callback.
	lf.lock.RUnlock()
	return buf, nil, err
}

// Test helper
func valueBytesToEntry(buf []byte) (e Entry) {
	var h header
	h.Decode(buf)
	n := uint32(headerBufSize)

	e.Key = buf[n : n+h.klen]
	n += h.klen
	e.meta = h.meta
	e.UserMeta = h.userMeta
	e.Value = buf[n : n+h.vlen]
	return
}

func (vlog *valueLog) pickLog(head valuePointer) *logFile {
	vlog.filesLock.RLock()
	defer vlog.filesLock.RUnlock()
	fids := vlog.sortedFids()
	if len(fids) <= 1 || head.Fid == 0 {
		return nil
	}

	// Pick a candidate that contains the largest amount of discardable data
	candidate := struct {
		fid     uint32
		discard int64
	}{math.MaxUint32, 0}
	vlog.lfDiscardStats.Lock()
	for _, fid := range fids {
		if fid >= head.Fid {
			break
		}
		if vlog.lfDiscardStats.m[fid] > candidate.discard {
			candidate.fid = fid
			candidate.discard = vlog.lfDiscardStats.m[fid]
		}
	}
	vlog.lfDiscardStats.Unlock()

	if candidate.fid != math.MaxUint32 { // Found a candidate
		return vlog.filesMap[candidate.fid]
	}

	// Fallback to randomly picking a log file
	var idxHead int
	for i, fid := range fids {
		if fid == head.Fid {
			idxHead = i
			break
		}
	}
	if idxHead == 0 { // Not found or first file
		return nil
	}
	idx := rand.Intn(idxHead) // Don’t include head.Fid. We pick a random file before it.
	if idx > 0 {
		idx = rand.Intn(idx + 1) // Another level of rand to favor smaller fids.
	}
	return vlog.filesMap[fids[idx]]
}

func discardEntry(e Entry, vs y.ValueStruct) bool {
	if vs.Version != y.ParseTs(e.Key) {
		// Version not found. Discard.
		return true
	}
	if isDeletedOrExpired(vs.Meta, vs.ExpiresAt) {
		return true
	}
	if (vs.Meta & bitValuePointer) == 0 {
		// Key also stores the value in LSM. Discard.
		return true
	}
	if (vs.Meta & bitFinTxn) > 0 {
		// Just a txn finish entry. Discard.
		return true
	}
	return false
}

func (vlog *valueLog) doRunGC(gcThreshold float64, head valuePointer) (err error) {
	// Pick a log file for GC
	lf := vlog.pickLog(head)
	if lf == nil {
		return ErrNoRewrite
	}

	// Update stats before exiting
	defer func() {
		if err == nil {
			vlog.lfDiscardStats.Lock()
			delete(vlog.lfDiscardStats.m, lf.fid)
			vlog.lfDiscardStats.Unlock()
		}
	}()

	type reason struct {
		total   float64
		keep    float64
		discard float64
	}

	var r reason
	var window = 100.0
	count := 0

	// Pick a random start point for the log.
	skipFirstM := float64(rand.Intn(int(vlog.opt.ValueLogFileSize/mi))) - window
	var skipped float64

	start := time.Now()
	y.AssertTrue(vlog.kv != nil)
	s := new(y.Slice)
	err = vlog.iterate(lf, 0, func(e Entry, vp valuePointer) error {
		esz := float64(vp.Len) / (1 << 20) // in MBs. +4 for the CAS stuff.
		skipped += esz
		if skipped < skipFirstM {
			return nil
		}

		count++
		if count%100 == 0 {
			time.Sleep(time.Millisecond)
		}
		r.total += esz
		if r.total > window {
			return errStop
		}
		if time.Since(start) > 10*time.Second {
			return errStop
		}

		vs, err := vlog.kv.get(e.Key)
		if err != nil {
			return err
		}
		if discardEntry(e, vs) {
			r.discard += esz
			return nil
		}

		// Value is still present in value log.
		y.AssertTrue(len(vs.Value) > 0)
		vp.Decode(vs.Value)

		if vp.Fid > lf.fid {
			// Value is present in a later log. Discard.
			r.discard += esz
			return nil
		}
		if vp.Offset > e.offset {
			// Value is present in a later offset, but in the same log.
			r.discard += esz
			return nil
		}
		if vp.Fid == lf.fid && vp.Offset == e.offset {
			// This is still the active entry. This would need to be rewritten.
			r.keep += esz

		} else {
			vlog.elog.Printf("Reason=%+v\n", r)

			buf, cb, err := vlog.readValueBytes(vp, s)
			if err != nil {
				return errStop
			}
			ne := valueBytesToEntry(buf)
			ne.offset = vp.Offset
			ne.print("Latest Entry Header in LSM")
			e.print("Latest Entry in Log")
			runCallback(cb)
			return errors.Errorf("This shouldn't happen. Latest Pointer:%+v. Meta:%v.",
				vp, vs.Meta)
		}
		return nil
	})

	if err != nil {
		vlog.elog.Errorf("Error while iterating for RunGC: %v", err)
		return err
	}
	vlog.elog.Printf("Fid: %d Data status=%+v\n", lf.fid, r)

	if r.total < 10.0 || r.discard < gcThreshold*r.total {
		vlog.elog.Printf("Skipping GC on fid: %d\n\n", lf.fid)
		return ErrNoRewrite
	}

	vlog.elog.Printf("REWRITING VLOG %d\n", lf.fid)
	if err = vlog.rewrite(lf); err != nil {
		return err
	}
	vlog.elog.Printf("Done rewriting.")
	return nil
}

func (vlog *valueLog) waitOnGC(lc *y.Closer) {
	defer lc.Done()

	<-lc.HasBeenClosed() // Wait for lc to be closed.

	// Block any GC in progress to finish, and don't allow any more writes to runGC by filling up
	// the channel of size 1.
	vlog.garbageCh <- struct{}{}
}

func (vlog *valueLog) runGC(gcThreshold float64, head valuePointer) error {
	select {
	case vlog.garbageCh <- struct{}{}:
		// Run GC
		var (
			err   error
			count int
		)
		for {
			err = vlog.doRunGC(gcThreshold, head)
			if err != nil {
				break
			}
			count++
		}
		<-vlog.garbageCh
		if err == ErrNoRewrite && count > 0 {
			return nil
		}
		return err
	default:
		return ErrRejected
	}
}

func (vlog *valueLog) updateGCStats(item *Item) {
	if item.meta&bitValuePointer > 0 {
		var vp valuePointer
		vp.Decode(item.vptr)
		vlog.lfDiscardStats.Lock()
		vlog.lfDiscardStats.m[vp.Fid] += int64(vp.Len)
		vlog.lfDiscardStats.Unlock()
	}
}
