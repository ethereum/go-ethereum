package hook

import "sync"

type Record struct {
	ancientDbReadCnt          int // ancient DB 접근 횟수
	levelDbReadCnt            int // level DB 접근 횟수
	levelDbSnapshotReadCnt    int // level DB 스냅샷 접근 횟수
	duplicatedReadCnt         int // 중복 디비 접근 횟수
	duplicatedSnapshotReadCnt int // 중복 스냅샷 접근 횟수
	readDbKeySet              *map[string]bool
	readDbKeySetLock          sync.RWMutex
	readSnapshotDbKeySet      *map[string]bool
	readSnapshotDbKeySetLock  sync.RWMutex
	rpcLock                   sync.Mutex
}

var Gr = Record{
	ancientDbReadCnt:          0,
	levelDbReadCnt:            0,
	levelDbSnapshotReadCnt:    0,
	duplicatedReadCnt:         0,
	duplicatedSnapshotReadCnt: 0,
	readDbKeySet:              &map[string]bool{},
	readDbKeySetLock:          sync.RWMutex{},
	readSnapshotDbKeySet:      &map[string]bool{},
	readSnapshotDbKeySetLock:  sync.RWMutex{},
	rpcLock:                   sync.Mutex{},
}

func (gr *Record) Lock() {
	gr.rpcLock.Lock()
}

func (gr *Record) Unlock() {
	gr.rpcLock.Unlock()
}

func (gr *Record) Reset() {
	gr.ancientDbReadCnt = 0
	gr.levelDbReadCnt = 0
	gr.levelDbSnapshotReadCnt = 0
	gr.duplicatedReadCnt = 0
	gr.duplicatedSnapshotReadCnt = 0
	gr.readDbKeySet = &map[string]bool{}
	gr.readSnapshotDbKeySet = &map[string]bool{}
}

func (gr *Record) addReadKeySet(key string) {
	gr.readDbKeySetLock.Lock()
	defer gr.readDbKeySetLock.Unlock()

	(*gr.readDbKeySet)[key] = true
}

func (gr *Record) addReadSnapshotKeySet(key string) {
	gr.readSnapshotDbKeySetLock.Lock()
	defer gr.readSnapshotDbKeySetLock.Unlock()

	(*gr.readSnapshotDbKeySet)[key] = true
}

func (gr *Record) isAlreadyRead(key string) bool {
	gr.readDbKeySetLock.RLock()
	defer gr.readDbKeySetLock.RUnlock()
	if (*gr.readDbKeySet)[key] == true {
		return true
	}
	return false
}

func (gr *Record) countDuplicatedKey(key []byte) {
	keyStr := string(key[:])
	if gr.isAlreadyRead(keyStr) {
		gr.duplicatedReadCnt++
	} else {
		gr.addReadKeySet(keyStr)
	}
}

func (gr *Record) isAlreadyReadSnapshot(key string) bool {
	gr.readSnapshotDbKeySetLock.RLock()
	defer gr.readSnapshotDbKeySetLock.RUnlock()
	if (*gr.readSnapshotDbKeySet)[key] == true {
		return true
	}
	return false
}

func (gr *Record) countDuplicatedSnapshotKey(key []byte) {
	keyStr := string(key[:])
	if gr.isAlreadyReadSnapshot(keyStr) {
		gr.duplicatedSnapshotReadCnt++
	} else {
		gr.addReadSnapshotKeySet(keyStr)
	}
}

func (gr *Record) CountAncientDbRead(key []byte) {
	gr.countDuplicatedKey(key)
	gr.ancientDbReadCnt++
}

func (gr *Record) CountLevelDbRead(key []byte) {
	gr.countDuplicatedKey(key)
	gr.levelDbReadCnt++
}

func (gr *Record) CountLevelDbSnapshotRead(key []byte) {
	gr.countDuplicatedSnapshotKey(key)
	gr.levelDbSnapshotReadCnt++
}

func (gr *Record) AncientDbReadCnt() int {
	return gr.ancientDbReadCnt
}

func (gr *Record) LevelDbReadCnt() int {
	return gr.levelDbReadCnt
}

func (gr *Record) LevelDbSnapshotReadCnt() int {
	return gr.levelDbSnapshotReadCnt
}

func (gr *Record) DuplicatedReadCnt() int {
	return gr.duplicatedReadCnt
}

func (gr *Record) DuplicatedSnapshotReadCnt() int {
	return gr.duplicatedSnapshotReadCnt
}
