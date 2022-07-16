package blockstm

import (
	"fmt"
	"sync"

	"github.com/emirpasic/gods/maps/treemap"

	"github.com/ethereum/go-ethereum/log"
)

const FlagDone = 0
const FlagEstimate = 1

type MVHashMap struct {
	rw sync.RWMutex
	m  map[string]*TxnIndexCells // TODO: might want a more efficient key representation
}

func MakeMVHashMap() *MVHashMap {
	return &MVHashMap{
		rw: sync.RWMutex{},
		m:  make(map[string]*TxnIndexCells),
	}
}

type WriteCell struct {
	flag        uint
	incarnation int
	data        interface{}
}

type TxnIndexCells struct {
	rw sync.RWMutex
	tm *treemap.Map
}

type Version struct {
	TxnIndex    int
	Incarnation int
}

func (mv *MVHashMap) getKeyCells(k []byte, fNoKey func(kenc string) *TxnIndexCells) (cells *TxnIndexCells) {
	kenc := string(k)

	var ok bool

	mv.rw.RLock()
	cells, ok = mv.m[kenc]
	mv.rw.RUnlock()

	if !ok {
		cells = fNoKey(kenc)
	}

	return
}

func (mv *MVHashMap) Write(k []byte, v Version, data interface{}) {
	cells := mv.getKeyCells(k, func(kenc string) (cells *TxnIndexCells) {
		n := &TxnIndexCells{
			rw: sync.RWMutex{},
			tm: treemap.NewWithIntComparator(),
		}
		var ok bool
		mv.rw.Lock()
		if cells, ok = mv.m[kenc]; !ok {
			mv.m[kenc] = n
			cells = n
		}
		mv.rw.Unlock()
		return
	})

	// TODO: could probably have a scheme where this only generally requires a read lock since any given transaction transaction
	//  should only have one incarnation executing at a time...
	cells.rw.Lock()
	defer cells.rw.Unlock()
	ci, ok := cells.tm.Get(v.TxnIndex)

	if ok {
		if ci.(*WriteCell).incarnation > v.Incarnation {
			panic(fmt.Errorf("existing transaction value does not have lower incarnation: %v, %v",
				string(k), v.TxnIndex))
		} else if ci.(*WriteCell).flag == FlagEstimate {
			log.Debug("mvhashmap marking previous estimate as done", "tx index", v.TxnIndex, "incarnation", v.Incarnation)
		}

		ci.(*WriteCell).flag = FlagDone
		ci.(*WriteCell).incarnation = v.Incarnation
		ci.(*WriteCell).data = data
	} else {
		cells.tm.Put(v.TxnIndex, &WriteCell{
			flag:        FlagDone,
			incarnation: v.Incarnation,
			data:        data,
		})
	}
}

func (mv *MVHashMap) MarkEstimate(k []byte, txIdx int) {
	cells := mv.getKeyCells(k, func(_ string) *TxnIndexCells {
		panic(fmt.Errorf("path must already exist"))
	})

	cells.rw.RLock()
	if ci, ok := cells.tm.Get(txIdx); !ok {
		panic("should not happen - cell should be present for path")
	} else {
		ci.(*WriteCell).flag = FlagEstimate
	}
	cells.rw.RUnlock()
}

func (mv *MVHashMap) Delete(k []byte, txIdx int) {
	cells := mv.getKeyCells(k, func(_ string) *TxnIndexCells {
		panic(fmt.Errorf("path must already exist"))
	})

	cells.rw.Lock()
	defer cells.rw.Unlock()
	cells.tm.Remove(txIdx)
}

const (
	MVReadResultDone       = 0
	MVReadResultDependency = 1
	MVReadResultNone       = 2
)

type MVReadResult struct {
	depIdx      int
	incarnation int
	value       interface{}
}

func (res *MVReadResult) DepIdx() int {
	return res.depIdx
}

func (res *MVReadResult) Incarnation() int {
	return res.incarnation
}

func (res *MVReadResult) Value() interface{} {
	return res.value
}

func (mvr MVReadResult) Status() int {
	if mvr.depIdx != -1 {
		if mvr.incarnation == -1 {
			return MVReadResultDependency
		} else {
			return MVReadResultDone
		}
	}

	return MVReadResultNone
}

func (mv *MVHashMap) Read(k []byte, txIdx int) (res MVReadResult) {
	res.depIdx = -1
	res.incarnation = -1

	cells := mv.getKeyCells(k, func(_ string) *TxnIndexCells {
		return nil
	})
	if cells == nil {
		return
	}

	cells.rw.RLock()
	defer cells.rw.RUnlock()

	if fk, fv := cells.tm.Floor(txIdx - 1); fk != nil && fv != nil {
		c := fv.(*WriteCell)
		switch c.flag {
		case FlagEstimate:
			res.depIdx = fk.(int)
			res.value = c.data
		case FlagDone:
			{
				res.depIdx = fk.(int)
				res.incarnation = c.incarnation
				res.value = c.data
			}
		default:
			panic(fmt.Errorf("should not happen - unknown flag value"))
		}
	}

	return
}

func (mv *MVHashMap) FlushMVWriteSet(writes []WriteDescriptor) {
	for _, v := range writes {
		mv.Write(v.Path, v.V, v.Val)
	}
}

func ValidateVersion(txIdx int, lastInputOutput *TxnInputOutput, versionedData *MVHashMap) (valid bool) {
	valid = true

	for _, rd := range lastInputOutput.ReadSet(txIdx) {
		mvResult := versionedData.Read(rd.Path, txIdx)
		switch mvResult.Status() {
		case MVReadResultDone:
			valid = rd.Kind == ReadKindMap && rd.V == Version{
				TxnIndex:    mvResult.depIdx,
				Incarnation: mvResult.incarnation,
			}
		case MVReadResultDependency:
			valid = false
		case MVReadResultNone:
			valid = rd.Kind == ReadKindStorage // feels like an assertion?
		default:
			panic(fmt.Errorf("should not happen - undefined mv read status: %ver", mvResult.Status()))
		}

		if !valid {
			break
		}
	}

	return
}
