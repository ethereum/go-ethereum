// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wasm

import (
	"errors"
	"fmt"
	"io"

	"github.com/go-interpreter/wagon/wasm/leb128"
)

// Import is an interface implemented by types that can be imported by a WebAssembly module.
type Import interface {
	Kind() External
	Marshaler
	isImport()
}

// ImportEntry describes an import statement in a Wasm module.
type ImportEntry struct {
	ModuleName string // module name string
	FieldName  string // field name string

	// If Kind is Function, Type is a FuncImport containing the type index of the function signature
	// If Kind is Table, Type is a TableImport containing the type of the imported table
	// If Kind is Memory, Type is a MemoryImport containing the type of the imported memory
	// If the Kind is Global, Type is a GlobalVarImport
	Type Import
}

type FuncImport struct {
	Type uint32
}

func (FuncImport) isImport() {}
func (FuncImport) Kind() External {
	return ExternalFunction
}
func (f FuncImport) MarshalWASM(w io.Writer) error {
	_, err := leb128.WriteVarUint32(w, uint32(f.Type))
	return err
}

type TableImport struct {
	Type Table
}

func (TableImport) isImport() {}
func (TableImport) Kind() External {
	return ExternalTable
}
func (t TableImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}

type MemoryImport struct {
	Type Memory
}

func (MemoryImport) isImport() {}
func (MemoryImport) Kind() External {
	return ExternalMemory
}
func (t MemoryImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}

type GlobalVarImport struct {
	Type GlobalVar
}

func (GlobalVarImport) isImport() {}
func (GlobalVarImport) Kind() External {
	return ExternalGlobal
}
func (t GlobalVarImport) MarshalWASM(w io.Writer) error {
	return t.Type.MarshalWASM(w)
}

var (
	ErrImportMutGlobal           = errors.New("wasm: cannot import global mutable variable")
	ErrNoExportsInImportedModule = errors.New("wasm: imported module has no exports")
)

type InvalidExternalError uint8

func (e InvalidExternalError) Error() string {
	return fmt.Sprintf("wasm: invalid external_kind value %d", uint8(e))
}

type ExportNotFoundError struct {
	ModuleName string
	FieldName  string
}

type KindMismatchError struct {
	ModuleName string
	FieldName  string
	Import     External
	Export     External
}

func (e KindMismatchError) Error() string {
	return fmt.Sprintf("wasm: Mismatching import and export external kind values for %s.%s (%v, %v)", e.FieldName, e.ModuleName, e.Import, e.Export)
}

func (e ExportNotFoundError) Error() string {
	return fmt.Sprintf("wasm: couldn't find export with name %s in module %s", e.FieldName, e.ModuleName)
}

type InvalidFunctionIndexError uint32

func (e InvalidFunctionIndexError) Error() string {
	return fmt.Sprintf("wasm: Invalid index to function index space: %#x", uint32(e))
}

func (module *Module) resolveImports(resolve ResolveFunc) error {
	if module.Import == nil {
		return nil
	}

	modules := make(map[string]*Module)

	var funcs uint32
	for _, importEntry := range module.Import.Entries {
		importedModule, ok := modules[importEntry.ModuleName]
		if !ok {
			var err error
			importedModule, err = resolve(importEntry.ModuleName)
			if err != nil {
				return err
			}

			modules[importEntry.ModuleName] = importedModule
		}

		if importedModule.Export == nil {
			return ErrNoExportsInImportedModule
		}

		exportEntry, ok := importedModule.Export.Entries[importEntry.FieldName]
		if !ok {
			return ExportNotFoundError{importEntry.ModuleName, importEntry.FieldName}
		}

		if exportEntry.Kind != importEntry.Type.Kind() {
			return KindMismatchError{
				FieldName:  importEntry.FieldName,
				ModuleName: importEntry.ModuleName,
				Import:     importEntry.Type.Kind(),
				Export:     exportEntry.Kind,
			}
		}

		index := exportEntry.Index

		switch exportEntry.Kind {
		case ExternalFunction:
			fn := importedModule.GetFunction(int(index))
			if fn == nil {
				return InvalidFunctionIndexError(index)
			}
			module.FunctionIndexSpace = append(module.FunctionIndexSpace, *fn)
			module.Code.Bodies = append(module.Code.Bodies, *fn.Body)
			module.imports.Funcs = append(module.imports.Funcs, funcs)
			funcs++
		case ExternalGlobal:
			glb := importedModule.GetGlobal(int(index))
			if glb == nil {
				return InvalidGlobalIndexError(index)
			}
			if glb.Type.Mutable {
				return ErrImportMutGlobal
			}
			module.GlobalIndexSpace = append(module.GlobalIndexSpace, *glb)
			module.imports.Globals++

			// In both cases below, index should be always 0 (according to the MVP)
			// We check it against the length of the index space anyway.
		case ExternalTable:
			if int(index) >= len(importedModule.TableIndexSpace) {
				return InvalidTableIndexError(index)
			}
			module.TableIndexSpace[0] = importedModule.TableIndexSpace[0]
			module.imports.Tables++
		case ExternalMemory:
			if int(index) >= len(importedModule.LinearMemoryIndexSpace) {
				return InvalidLinearMemoryIndexError(index)
			}
			module.LinearMemoryIndexSpace[0] = importedModule.LinearMemoryIndexSpace[0]
			module.imports.Memories++
		default:
			return InvalidExternalError(exportEntry.Kind)
		}
	}
	return nil
}
