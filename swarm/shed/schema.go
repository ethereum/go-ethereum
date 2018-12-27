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

package shed

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	// LevelDB key value for storing the schema.
	keySchema = []byte{0}
	// LevelDB key prefix for all field type.
	// LevelDB keys will be constructed by appending name values to this prefix.
	keyPrefixFields byte = 1
	// LevelDB key prefix from which indexing keys start.
	// Every index has its own key prefix and this value defines the first one.
	keyPrefixIndexStart byte = 2 // Q: or maybe a higher number like 7, to have more space for potential specific perfixes
)

// schema is used to serialize known database structure information.
type schema struct {
	Fields  map[string]fieldSpec `json:"fields"`  // keys are field names
	Indexes map[byte]indexSpec   `json:"indexes"` // keys are index prefix bytes
}

// fieldSpec holds information about a particular field.
// It does not need Name field as it is contained in the
// schema.Field map key.
type fieldSpec struct {
	Type string `json:"type"`
}

// indxSpec holds information about a particular index.
// It does not contain index type, as indexes do not have type.
type indexSpec struct {
	Name string `json:"name"`
}

// schemaFieldKey retrives the complete LevelDB key for
// a particular field form the schema definition.
func (db *DB) schemaFieldKey(name, fieldType string) (key []byte, err error) {
	if name == "" {
		return nil, errors.New("field name can not be blank")
	}
	if fieldType == "" {
		return nil, errors.New("field type can not be blank")
	}
	s, err := db.getSchema()
	if err != nil {
		return nil, err
	}
	var found bool
	for n, f := range s.Fields {
		if n == name {
			if f.Type != fieldType {
				return nil, fmt.Errorf("field %q of type %q stored as %q in db", name, fieldType, f.Type)
			}
			break
		}
	}
	if !found {
		s.Fields[name] = fieldSpec{
			Type: fieldType,
		}
		err := db.putSchema(s)
		if err != nil {
			return nil, err
		}
	}
	return append([]byte{keyPrefixFields}, []byte(name)...), nil
}

// schemaIndexID retrieves the complete LevelDB prefix for
// a particular index.
func (db *DB) schemaIndexPrefix(name string) (id byte, err error) {
	if name == "" {
		return 0, errors.New("index name can not be blank")
	}
	s, err := db.getSchema()
	if err != nil {
		return 0, err
	}
	nextID := keyPrefixIndexStart
	for i, f := range s.Indexes {
		if i >= nextID {
			nextID = i + 1
		}
		if f.Name == name {
			return i, nil
		}
	}
	id = nextID
	s.Indexes[id] = indexSpec{
		Name: name,
	}
	return id, db.putSchema(s)
}

// getSchema retrieves the complete schema from
// the database.
func (db *DB) getSchema() (s schema, err error) {
	b, err := db.Get(keySchema)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(b, &s)
	return s, err
}

// putSchema stores the complete schema to
// the database.
func (db *DB) putSchema(s schema) (err error) {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(keySchema, b)
}
