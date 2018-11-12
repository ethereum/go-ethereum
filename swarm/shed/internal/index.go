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

package internal

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type IndexItem struct {
	Hash            []byte
	Data            []byte
	AccessTimestamp int64
	StoreTimestamp  int64
}

func (i IndexItem) Join(i2 IndexItem) (new IndexItem) {
	if i.Hash == nil {
		i.Hash = i2.Hash
	}
	if i.Data == nil {
		i.Data = i2.Data
	}
	if i.AccessTimestamp == 0 {
		i.AccessTimestamp = i2.AccessTimestamp
	}
	if i.StoreTimestamp == 0 {
		i.StoreTimestamp = i2.StoreTimestamp
	}
	return i
}

type Index struct {
	db              *DB
	prefix          []byte
	encodeKeyFunc   func(fields IndexItem) (key []byte, err error)
	decodeKeyFunc   func(key []byte) (e IndexItem, err error)
	encodeValueFunc func(fields IndexItem) (value []byte, err error)
	decodeValueFunc func(value []byte) (e IndexItem, err error)
}

type IndexFuncs struct {
	EncodeKey   func(fields IndexItem) (key []byte, err error)
	DecodeKey   func(key []byte) (e IndexItem, err error)
	EncodeValue func(fields IndexItem) (value []byte, err error)
	DecodeValue func(value []byte) (e IndexItem, err error)
}

func (db *DB) NewIndex(name string, funcs IndexFuncs) (f Index, err error) {
	id, err := db.schemaIndexID(name)
	if err != nil {
		return f, err
	}
	prefix := []byte{id}
	return Index{
		db:     db,
		prefix: prefix,
		encodeKeyFunc: func(e IndexItem) (key []byte, err error) {
			key, err = funcs.EncodeKey(e)
			if err != nil {
				return nil, err
			}
			return append(append(make([]byte, 0, len(key)+1), prefix...), key...), nil
		},
		decodeKeyFunc: func(key []byte) (e IndexItem, err error) {
			return funcs.DecodeKey(key[1:])
		},
		encodeValueFunc: funcs.EncodeValue,
		decodeValueFunc: funcs.DecodeValue,
	}, nil
}

func (f Index) Get(keyFields IndexItem) (out IndexItem, err error) {
	key, err := f.encodeKeyFunc(keyFields)
	if err != nil {
		return out, err
	}
	value, err := f.db.Get(key)
	if err != nil {
		return out, err
	}
	out, err = f.decodeValueFunc(value)
	if err != nil {
		return out, err
	}
	return out.Join(keyFields), nil
}

func (f Index) Put(i IndexItem) (err error) {
	key, err := f.encodeKeyFunc(i)
	if err != nil {
		return err
	}
	value, err := f.encodeValueFunc(i)
	if err != nil {
		return err
	}
	return f.db.Put(key, value)
}

func (f Index) PutInBatch(batch *leveldb.Batch, i IndexItem) (err error) {
	key, err := f.encodeKeyFunc(i)
	if err != nil {
		return err
	}
	value, err := f.encodeValueFunc(i)
	if err != nil {
		return err
	}
	batch.Put(key, value)
	return nil
}

func (f Index) Delete(keyFields IndexItem) (err error) {
	key, err := f.encodeKeyFunc(keyFields)
	if err != nil {
		return err
	}
	return f.db.Delete(key)
}

func (f Index) DeleteInBatch(batch *leveldb.Batch, keyFields IndexItem) (err error) {
	key, err := f.encodeKeyFunc(keyFields)
	if err != nil {
		return err
	}
	batch.Delete(key)
	return nil
}

type IterFunc func(item IndexItem) (stop bool, err error)

func (f Index) IterateAll(fn IterFunc) (err error) {
	it := f.db.NewIterator()
	defer it.Release()

	for ok := it.Seek(f.prefix); ok; ok = it.Next() {
		key := it.Key()
		if key[0] != f.prefix[0] {
			break
		}
		keyIndexItem, err := f.decodeKeyFunc(key)
		if err != nil {
			return err
		}
		valueIndexItem, err := f.decodeValueFunc(it.Value())
		if err != nil {
			return err
		}
		stop, err := fn(keyIndexItem.Join(valueIndexItem))
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}
	return it.Error()
}

func (f Index) IterateFrom(start IndexItem, fn IterFunc) (err error) {
	startKey, err := f.encodeKeyFunc(start)
	if err != nil {
		return err
	}
	it := f.db.NewIterator()
	defer it.Release()

	for ok := it.Seek(startKey); ok; ok = it.Next() {
		key := it.Key()
		if key[0] != f.prefix[0] {
			break
		}
		keyIndexItem, err := f.decodeKeyFunc(key)
		if err != nil {
			return err
		}
		valueIndexItem, err := f.decodeValueFunc(it.Value())
		if err != nil {
			return err
		}
		stop, err := fn(keyIndexItem.Join(valueIndexItem))
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}
	return it.Error()
}
