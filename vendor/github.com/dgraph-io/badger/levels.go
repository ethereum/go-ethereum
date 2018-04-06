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
	"fmt"
	"math/rand"
	"os"
	"sort"
	"time"

	"golang.org/x/net/trace"

	"github.com/dgraph-io/badger/protos"
	"github.com/dgraph-io/badger/table"
	"github.com/dgraph-io/badger/y"
	"github.com/pkg/errors"
)

type levelsController struct {
	nextFileID uint64 // Atomic
	elog       trace.EventLog

	// The following are initialized once and const.
	levels []*levelHandler
	kv     *DB

	cstatus compactStatus
}

var (
	// This is for getting timings between stalls.
	lastUnstalled time.Time
)

// revertToManifest checks that all necessary table files exist and removes all table files not
// referenced by the manifest.  idMap is a set of table file id's that were read from the directory
// listing.
func revertToManifest(kv *DB, mf *Manifest, idMap map[uint64]struct{}) error {
	// 1. Check all files in manifest exist.
	for id := range mf.Tables {
		if _, ok := idMap[id]; !ok {
			return fmt.Errorf("file does not exist for table %d", id)
		}
	}

	// 2. Delete files that shouldn't exist.
	for id := range idMap {
		if _, ok := mf.Tables[id]; !ok {
			kv.elog.Printf("Table file %d not referenced in MANIFEST\n", id)
			filename := table.NewFilename(id, kv.opt.Dir)
			if err := os.Remove(filename); err != nil {
				return y.Wrapf(err, "While removing table %d", id)
			}
		}
	}

	return nil
}

func newLevelsController(kv *DB, mf *Manifest) (*levelsController, error) {
	y.AssertTrue(kv.opt.NumLevelZeroTablesStall > kv.opt.NumLevelZeroTables)
	s := &levelsController{
		kv:     kv,
		elog:   kv.elog,
		levels: make([]*levelHandler, kv.opt.MaxLevels),
	}
	s.cstatus.levels = make([]*levelCompactStatus, kv.opt.MaxLevels)

	for i := 0; i < kv.opt.MaxLevels; i++ {
		s.levels[i] = newLevelHandler(kv, i)
		if i == 0 {
			// Do nothing.
		} else if i == 1 {
			// Level 1 probably shouldn't be too much bigger than level 0.
			s.levels[i].maxTotalSize = kv.opt.LevelOneSize
		} else {
			s.levels[i].maxTotalSize = s.levels[i-1].maxTotalSize * int64(kv.opt.LevelSizeMultiplier)
		}
		s.cstatus.levels[i] = new(levelCompactStatus)
	}

	// Compare manifest against directory, check for existent/non-existent files, and remove.
	if err := revertToManifest(kv, mf, getIDMap(kv.opt.Dir)); err != nil {
		return nil, err
	}

	// Some files may be deleted. Let's reload.
	tables := make([][]*table.Table, kv.opt.MaxLevels)
	var maxFileID uint64
	for fileID, tableManifest := range mf.Tables {
		fname := table.NewFilename(fileID, kv.opt.Dir)
		fd, err := y.OpenExistingSyncedFile(fname, true)
		if err != nil {
			closeAllTables(tables)
			return nil, errors.Wrapf(err, "Opening file: %q", fname)
		}

		t, err := table.OpenTable(fd, kv.opt.TableLoadingMode)
		if err != nil {
			closeAllTables(tables)
			return nil, errors.Wrapf(err, "Opening table: %q", fname)
		}

		level := tableManifest.Level
		tables[level] = append(tables[level], t)

		if fileID > maxFileID {
			maxFileID = fileID
		}
	}
	s.nextFileID = maxFileID + 1
	for i, tbls := range tables {
		s.levels[i].initTables(tbls)
	}

	// Make sure key ranges do not overlap etc.
	if err := s.validate(); err != nil {
		_ = s.cleanupLevels()
		return nil, errors.Wrap(err, "Level validation")
	}

	// Sync directory (because we have at least removed some files, or previously created the
	// manifest file).
	if err := syncDir(kv.opt.Dir); err != nil {
		_ = s.close()
		return nil, err
	}

	return s, nil
}

// Closes the tables, for cleanup in newLevelsController.  (We Close() instead of using DecrRef()
// because that would delete the underlying files.)  We ignore errors, which is OK because tables
// are read-only.
func closeAllTables(tables [][]*table.Table) {
	for _, tableSlice := range tables {
		for _, table := range tableSlice {
			_ = table.Close()
		}
	}
}

func (s *levelsController) cleanupLevels() error {
	var firstErr error
	for _, l := range s.levels {
		if err := l.close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *levelsController) startCompact(lc *y.Closer) {
	n := s.kv.opt.NumCompactors
	lc.AddRunning(n - 1)
	for i := 0; i < n; i++ {
		go s.runWorker(lc)
	}
}

func (s *levelsController) runWorker(lc *y.Closer) {
	defer lc.Done()
	if s.kv.opt.DoNotCompact {
		return
	}

	time.Sleep(time.Duration(rand.Int31n(1000)) * time.Millisecond)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		// Can add a done channel or other stuff.
		case <-ticker.C:
			prios := s.pickCompactLevels()
			for _, p := range prios {
				// TODO: Handle error.
				didCompact, _ := s.doCompact(p)
				if didCompact {
					break
				}
			}
		case <-lc.HasBeenClosed():
			return
		}
	}
}

// Returns true if level zero may be compacted, without accounting for compactions that already
// might be happening.
func (s *levelsController) isLevel0Compactable() bool {
	return s.levels[0].numTables() >= s.kv.opt.NumLevelZeroTables
}

// Returns true if the non-zero level may be compacted.  delSize provides the size of the tables
// which are currently being compacted so that we treat them as already having started being
// compacted (because they have been, yet their size is already counted in getTotalSize).
func (l *levelHandler) isCompactable(delSize int64) bool {
	return l.getTotalSize()-delSize >= l.maxTotalSize
}

type compactionPriority struct {
	level int
	score float64
}

// pickCompactLevel determines which level to compact.
// Based on: https://github.com/facebook/rocksdb/wiki/Leveled-Compaction
func (s *levelsController) pickCompactLevels() (prios []compactionPriority) {
	// This function must use identical criteria for guaranteeing compaction's progress that
	// addLevel0Table uses.

	// cstatus is checked to see if level 0's tables are already being compacted
	if !s.cstatus.overlapsWith(0, infRange) && s.isLevel0Compactable() {
		pri := compactionPriority{
			level: 0,
			score: float64(s.levels[0].numTables()) / float64(s.kv.opt.NumLevelZeroTables),
		}
		prios = append(prios, pri)
	}

	for i, l := range s.levels[1:] {
		// Don't consider those tables that are already being compacted right now.
		delSize := s.cstatus.delSize(i + 1)

		if l.isCompactable(delSize) {
			pri := compactionPriority{
				level: i + 1,
				score: float64(l.getTotalSize()-delSize) / float64(l.maxTotalSize),
			}
			prios = append(prios, pri)
		}
	}
	sort.Slice(prios, func(i, j int) bool {
		return prios[i].score > prios[j].score
	})
	return prios
}

// compactBuildTables merge topTables and botTables to form a list of new tables.
func (s *levelsController) compactBuildTables(
	l int, cd compactDef) ([]*table.Table, func() error, error) {
	topTables := cd.top
	botTables := cd.bot

	// Create iterators across all the tables involved first.
	var iters []y.Iterator
	if l == 0 {
		iters = appendIteratorsReversed(iters, topTables, false)
	} else {
		y.AssertTrue(len(topTables) == 1)
		iters = []y.Iterator{topTables[0].NewIterator(false)}
	}

	// Next level has level>=1 and we can use ConcatIterator as key ranges do not overlap.
	iters = append(iters, table.NewConcatIterator(botTables, false))
	it := y.NewMergeIterator(iters, false)
	defer it.Close() // Important to close the iterator to do ref counting.

	it.Rewind()

	// Start generating new tables.
	type newTableResult struct {
		table *table.Table
		err   error
	}
	resultCh := make(chan newTableResult)
	var i int
	for ; it.Valid(); i++ {
		timeStart := time.Now()
		builder := table.NewTableBuilder()
		for ; it.Valid(); it.Next() {
			if builder.ReachedCapacity(s.kv.opt.MaxTableSize) {
				break
			}
			y.Check(builder.Add(it.Key(), it.Value()))
		}
		// It was true that it.Valid() at least once in the loop above, which means we
		// called Add() at least once, and builder is not Empty().
		y.AssertTrue(!builder.Empty())

		cd.elog.LazyPrintf("LOG Compact. Iteration to generate one table took: %v\n", time.Since(timeStart))

		fileID := s.reserveFileID()
		go func(builder *table.Builder) {
			defer builder.Close()

			fd, err := y.CreateSyncedFile(table.NewFilename(fileID, s.kv.opt.Dir), true)
			if err != nil {
				resultCh <- newTableResult{nil, errors.Wrapf(err, "While opening new table: %d", fileID)}
				return
			}

			if _, err := fd.Write(builder.Finish()); err != nil {
				resultCh <- newTableResult{nil, errors.Wrapf(err, "Unable to write to file: %d", fileID)}
				return
			}

			tbl, err := table.OpenTable(fd, s.kv.opt.TableLoadingMode)
			// decrRef is added below.
			resultCh <- newTableResult{tbl, errors.Wrapf(err, "Unable to open table: %q", fd.Name())}
		}(builder)
	}

	newTables := make([]*table.Table, 0, 20)

	// Wait for all table builders to finish.
	var firstErr error
	for x := 0; x < i; x++ {
		res := <-resultCh
		newTables = append(newTables, res.table)
		if firstErr == nil {
			firstErr = res.err
		}
	}

	if firstErr == nil {
		// Ensure created files' directory entries are visible.  We don't mind the extra latency
		// from not doing this ASAP after all file creation has finished because this is a
		// background operation.
		firstErr = syncDir(s.kv.opt.Dir)
	}

	if firstErr != nil {
		// An error happened.  Delete all the newly created table files (by calling DecrRef
		// -- we're the only holders of a ref).
		for j := 0; j < i; j++ {
			if newTables[j] != nil {
				newTables[j].DecrRef()
			}
		}
		errorReturn := errors.Wrapf(firstErr, "While running compaction for: %+v", cd)
		return nil, nil, errorReturn
	}

	sort.Slice(newTables, func(i, j int) bool {
		return y.CompareKeys(newTables[i].Biggest(), newTables[j].Biggest()) < 0
	})

	return newTables, func() error { return decrRefs(newTables) }, nil
}

func buildChangeSet(cd *compactDef, newTables []*table.Table) protos.ManifestChangeSet {
	changes := []*protos.ManifestChange{}
	for _, table := range newTables {
		changes = append(changes, makeTableCreateChange(table.ID(), cd.nextLevel.level))
	}
	for _, table := range cd.top {
		changes = append(changes, makeTableDeleteChange(table.ID()))
	}
	for _, table := range cd.bot {
		changes = append(changes, makeTableDeleteChange(table.ID()))
	}
	return protos.ManifestChangeSet{Changes: changes}
}

type compactDef struct {
	elog trace.Trace

	thisLevel *levelHandler
	nextLevel *levelHandler

	top []*table.Table
	bot []*table.Table

	thisRange keyRange
	nextRange keyRange

	thisSize int64
}

func (cd *compactDef) lockLevels() {
	cd.thisLevel.RLock()
	cd.nextLevel.RLock()
}

func (cd *compactDef) unlockLevels() {
	cd.nextLevel.RUnlock()
	cd.thisLevel.RUnlock()
}

func (s *levelsController) fillTablesL0(cd *compactDef) bool {
	cd.lockLevels()
	defer cd.unlockLevels()

	cd.top = make([]*table.Table, len(cd.thisLevel.tables))
	copy(cd.top, cd.thisLevel.tables)
	if len(cd.top) == 0 {
		return false
	}
	cd.thisRange = infRange

	kr := getKeyRange(cd.top)
	left, right := cd.nextLevel.overlappingTables(levelHandlerRLocked{}, kr)
	cd.bot = make([]*table.Table, right-left)
	copy(cd.bot, cd.nextLevel.tables[left:right])

	if len(cd.bot) == 0 {
		cd.nextRange = kr
	} else {
		cd.nextRange = getKeyRange(cd.bot)
	}

	if !s.cstatus.compareAndAdd(thisAndNextLevelRLocked{}, *cd) {
		return false
	}

	return true
}

func (s *levelsController) fillTables(cd *compactDef) bool {
	cd.lockLevels()
	defer cd.unlockLevels()

	tbls := make([]*table.Table, len(cd.thisLevel.tables))
	copy(tbls, cd.thisLevel.tables)
	if len(tbls) == 0 {
		return false
	}

	// Find the biggest table, and compact that first.
	// TODO: Try other table picking strategies.
	sort.Slice(tbls, func(i, j int) bool {
		return tbls[i].Size() > tbls[j].Size()
	})

	for _, t := range tbls {
		cd.thisSize = t.Size()
		cd.thisRange = keyRange{
			left:  t.Smallest(),
			right: t.Biggest(),
		}
		if s.cstatus.overlapsWith(cd.thisLevel.level, cd.thisRange) {
			continue
		}
		cd.top = []*table.Table{t}
		left, right := cd.nextLevel.overlappingTables(levelHandlerRLocked{}, cd.thisRange)

		cd.bot = make([]*table.Table, right-left)
		copy(cd.bot, cd.nextLevel.tables[left:right])

		if len(cd.bot) == 0 {
			cd.bot = []*table.Table{}
			cd.nextRange = cd.thisRange
			if !s.cstatus.compareAndAdd(thisAndNextLevelRLocked{}, *cd) {
				continue
			}
			return true
		}
		cd.nextRange = getKeyRange(cd.bot)

		if s.cstatus.overlapsWith(cd.nextLevel.level, cd.nextRange) {
			continue
		}

		if !s.cstatus.compareAndAdd(thisAndNextLevelRLocked{}, *cd) {
			continue
		}
		return true
	}
	return false
}

func (s *levelsController) runCompactDef(l int, cd compactDef) (err error) {
	timeStart := time.Now()

	thisLevel := cd.thisLevel
	nextLevel := cd.nextLevel

	if thisLevel.level >= 1 && len(cd.bot) == 0 {
		y.AssertTrue(len(cd.top) == 1)
		tbl := cd.top[0]

		// We write to the manifest _before_ we delete files (and after we created files).
		changes := []*protos.ManifestChange{
			// The order matters here -- you can't temporarily have two copies of the same
			// table id when reloading the manifest.
			makeTableDeleteChange(tbl.ID()),
			makeTableCreateChange(tbl.ID(), nextLevel.level),
		}
		if err := s.kv.manifest.addChanges(changes); err != nil {
			return err
		}

		// We have to add to nextLevel before we remove from thisLevel, not after.  This way, we
		// don't have a bug where reads would see keys missing from both levels.

		// Note: It's critical that we add tables (replace them) in nextLevel before deleting them
		// in thisLevel.  (We could finagle it atomically somehow.)  Also, when reading we must
		// read, or at least acquire s.RLock(), in increasing order by level, so that we don't skip
		// a compaction.

		if err := nextLevel.replaceTables(cd.top); err != nil {
			return err
		}
		if err := thisLevel.deleteTables(cd.top); err != nil {
			return err
		}

		cd.elog.LazyPrintf("\tLOG Compact-Move %d->%d smallest:%s biggest:%s took %v\n",
			l, l+1, string(tbl.Smallest()), string(tbl.Biggest()), time.Since(timeStart))
		return nil
	}

	newTables, decr, err := s.compactBuildTables(l, cd)
	if err != nil {
		return err
	}
	defer func() {
		// Only assign to err, if it's not already nil.
		if decErr := decr(); err == nil {
			err = decErr
		}
	}()
	changeSet := buildChangeSet(&cd, newTables)

	// We write to the manifest _before_ we delete files (and after we created files)
	if err := s.kv.manifest.addChanges(changeSet.Changes); err != nil {
		return err
	}

	// See comment earlier in this function about the ordering of these ops, and the order in which
	// we access levels when reading.
	if err := nextLevel.replaceTables(newTables); err != nil {
		return err
	}
	if err := thisLevel.deleteTables(cd.top); err != nil {
		return err
	}

	// Note: For level 0, while doCompact is running, it is possible that new tables are added.
	// However, the tables are added only to the end, so it is ok to just delete the first table.

	cd.elog.LazyPrintf("LOG Compact %d->%d, del %d tables, add %d tables, took %v\n",
		l, l+1, len(cd.top)+len(cd.bot), len(newTables), time.Since(timeStart))
	return nil
}

// doCompact picks some table on level l and compacts it away to the next level.
func (s *levelsController) doCompact(p compactionPriority) (bool, error) {
	l := p.level
	y.AssertTrue(l+1 < s.kv.opt.MaxLevels) // Sanity check.

	cd := compactDef{
		elog:      trace.New("Badger", "Compact"),
		thisLevel: s.levels[l],
		nextLevel: s.levels[l+1],
	}
	cd.elog.SetMaxEvents(100)
	defer cd.elog.Finish()

	cd.elog.LazyPrintf("Got compaction priority: %+v", p)

	// While picking tables to be compacted, both levels' tables are expected to
	// remain unchanged.
	if l == 0 {
		if !s.fillTablesL0(&cd) {
			cd.elog.LazyPrintf("fillTables failed for level: %d\n", l)
			return false, nil
		}

	} else {
		if !s.fillTables(&cd) {
			cd.elog.LazyPrintf("fillTables failed for level: %d\n", l)
			return false, nil
		}
	}

	cd.elog.LazyPrintf("Running for level: %d\n", cd.thisLevel.level)
	s.cstatus.toLog(cd.elog)
	if err := s.runCompactDef(l, cd); err != nil {
		// This compaction couldn't be done successfully.
		cd.elog.LazyPrintf("\tLOG Compact FAILED with error: %+v: %+v", err, cd)
		return false, err
	}

	// Done with compaction. So, remove the ranges from compaction status.
	s.cstatus.delete(cd)
	s.cstatus.toLog(cd.elog)
	cd.elog.LazyPrintf("Compaction for level: %d DONE", cd.thisLevel.level)
	return true, nil
}

func (s *levelsController) addLevel0Table(t *table.Table) error {
	// We update the manifest _before_ the table becomes part of a levelHandler, because at that
	// point it could get used in some compaction.  This ensures the manifest file gets updated in
	// the proper order. (That means this update happens before that of some compaction which
	// deletes the table.)
	err := s.kv.manifest.addChanges([]*protos.ManifestChange{
		makeTableCreateChange(t.ID(), 0),
	})
	if err != nil {
		return err
	}

	for !s.levels[0].tryAddLevel0Table(t) {
		// Stall. Make sure all levels are healthy before we unstall.
		var timeStart time.Time
		{
			s.elog.Printf("STALLED STALLED STALLED STALLED STALLED STALLED STALLED STALLED: %v\n",
				time.Since(lastUnstalled))
			s.cstatus.RLock()
			for i := 0; i < s.kv.opt.MaxLevels; i++ {
				s.elog.Printf("level=%d. Status=%s Size=%d\n",
					i, s.cstatus.levels[i].debug(), s.levels[i].getTotalSize())
			}
			s.cstatus.RUnlock()
			timeStart = time.Now()
		}
		// Before we unstall, we need to make sure that level 0 and 1 are healthy. Otherwise, we
		// will very quickly fill up level 0 again and if the compaction strategy favors level 0,
		// then level 1 is going to super full.
		for {
			// Passing 0 for delSize to compactable means we're treating incomplete compactions as
			// not having finished -- we wait for them to finish.  Also, it's crucial this behavior
			// replicates pickCompactLevels' behavior in computing compactability in order to
			// guarantee progress.
			if !s.isLevel0Compactable() && !s.levels[1].isCompactable(0) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		{
			s.elog.Printf("UNSTALLED UNSTALLED UNSTALLED UNSTALLED UNSTALLED UNSTALLED: %v\n",
				time.Since(timeStart))
			lastUnstalled = time.Now()
		}
	}

	return nil
}

func (s *levelsController) close() error {
	err := s.cleanupLevels()
	return errors.Wrap(err, "levelsController.Close")
}

// get returns the found value if any. If not found, we return nil.
func (s *levelsController) get(key []byte, maxVs y.ValueStruct) (y.ValueStruct, error) {
	// It's important that we iterate the levels from 0 on upward.  The reason is, if we iterated
	// in opposite order, or in parallel (naively calling all the h.RLock() in some order) we could
	// read level L's tables post-compaction and level L+1's tables pre-compaction.  (If we do
	// parallelize this, we will need to call the h.RLock() function by increasing order of level
	// number.)

	version := y.ParseTs(key)
	for _, h := range s.levels {
		vs, err := h.get(key) // Calls h.RLock() and h.RUnlock().
		if err != nil {
			return y.ValueStruct{}, errors.Wrapf(err, "get key: %q", key)
		}
		if vs.Value == nil && vs.Meta == 0 {
			continue
		}
		if vs.Version == version {
			return vs, nil
		}
		if maxVs.Version < vs.Version {
			maxVs = vs
		}
	}
	return maxVs, nil
}

func appendIteratorsReversed(out []y.Iterator, th []*table.Table, reversed bool) []y.Iterator {
	for i := len(th) - 1; i >= 0; i-- {
		// This will increment the reference of the table handler.
		out = append(out, th[i].NewIterator(reversed))
	}
	return out
}

// appendIterators appends iterators to an array of iterators, for merging.
// Note: This obtains references for the table handlers. Remember to close these iterators.
func (s *levelsController) appendIterators(
	iters []y.Iterator, reversed bool) []y.Iterator {
	// Just like with get, it's important we iterate the levels from 0 on upward, to avoid missing
	// data when there's a compaction.
	for _, level := range s.levels {
		iters = level.appendIterators(iters, reversed)
	}
	return iters
}
