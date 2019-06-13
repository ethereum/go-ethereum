package duktape

/*
#cgo !windows CFLAGS: -std=c99 -O3 -Wall -Wno-unused-value -fomit-frame-pointer -fstrict-aliasing
#cgo windows CFLAGS: -O3 -Wall -Wno-unused-value -fomit-frame-pointer -fstrict-aliasing

#include "duktape.h"
*/
import "C"

const (
	CompileEval       uint = C.DUK_COMPILE_EVAL
	CompileFunction   uint = C.DUK_COMPILE_FUNCTION
	CompileStrict     uint = C.DUK_COMPILE_STRICT
	CompileShebang    uint = C.DUK_COMPILE_SHEBANG
	CompileSafe       uint = C.DUK_COMPILE_SAFE
	CompileNoResult   uint = C.DUK_COMPILE_NORESULT
	CompileNoSource   uint = C.DUK_COMPILE_NOSOURCE
	CompileStrlen     uint = C.DUK_COMPILE_STRLEN
	CompileNoFileName uint = C.DUK_COMPILE_NOFILENAME
	CompileFuncExpr   uint = C.DUK_COMPILE_FUNCEXPR
)

const (
	TypeNone      Type = C.DUK_TYPE_NONE
	TypeUndefined Type = C.DUK_TYPE_UNDEFINED
	TypeNull      Type = C.DUK_TYPE_NULL
	TypeBoolean   Type = C.DUK_TYPE_BOOLEAN
	TypeNumber    Type = C.DUK_TYPE_NUMBER
	TypeString    Type = C.DUK_TYPE_STRING
	TypeObject    Type = C.DUK_TYPE_OBJECT
	TypeBuffer    Type = C.DUK_TYPE_BUFFER
	TypePointer   Type = C.DUK_TYPE_POINTER
	TypeLightFunc Type = C.DUK_TYPE_LIGHTFUNC
)

const (
	TypeMaskNone      uint = C.DUK_TYPE_MASK_NONE
	TypeMaskUndefined uint = C.DUK_TYPE_MASK_UNDEFINED
	TypeMaskNull      uint = C.DUK_TYPE_MASK_NULL
	TypeMaskBoolean   uint = C.DUK_TYPE_MASK_BOOLEAN
	TypeMaskNumber    uint = C.DUK_TYPE_MASK_NUMBER
	TypeMaskString    uint = C.DUK_TYPE_MASK_STRING
	TypeMaskObject    uint = C.DUK_TYPE_MASK_OBJECT
	TypeMaskBuffer    uint = C.DUK_TYPE_MASK_BUFFER
	TypeMaskPointer   uint = C.DUK_TYPE_MASK_POINTER
	TypeMaskLightFunc uint = C.DUK_TYPE_MASK_LIGHTFUNC
)

const (
	EnumIncludeNonenumerable uint = C.DUK_ENUM_INCLUDE_NONENUMERABLE
	EnumIncludeHidden        uint = C.DUK_ENUM_INCLUDE_HIDDEN
	EnumIncludeSymbols       uint = C.DUK_ENUM_INCLUDE_SYMBOLS
	EnumExcludeStrings       uint = C.DUK_ENUM_EXCLUDE_STRINGS
	EnumOwnPropertiesOnly    uint = C.DUK_ENUM_OWN_PROPERTIES_ONLY
	EnumArrayIndicesOnly     uint = C.DUK_ENUM_ARRAY_INDICES_ONLY
	EnumSortArrayIndices     uint = C.DUK_ENUM_SORT_ARRAY_INDICES
	NoProxyBehavior          uint = C.DUK_ENUM_NO_PROXY_BEHAVIOR
)

const (
	ErrUnimplemented int = 50 + iota
	ErrUnsupported

	ErrNone      int = C.DUK_ERR_NONE
	ErrError     int = C.DUK_ERR_ERROR
	ErrEval      int = C.DUK_ERR_EVAL_ERROR
	ErrRange     int = C.DUK_ERR_RANGE_ERROR
	ErrReference int = C.DUK_ERR_REFERENCE_ERROR
	ErrSyntax    int = C.DUK_ERR_SYNTAX_ERROR
	ErrType      int = C.DUK_ERR_TYPE_ERROR
	ErrURI       int = C.DUK_ERR_URI_ERROR
)

const (
	// Returned error values
	ErrRetUnimplemented int = -(ErrUnimplemented + iota)
	ErrRetUnsupported
	ErrRetInternal
	ErrRetAlloc
	ErrRetAssertion
	ErrRetAPI
	ErrRetUncaughtError
)

const (
	ErrRetError     int = -(ErrError)
	ErrRetEval      int = -(ErrEval)
	ErrRetRange     int = -(ErrRange)
	ErrRetReference int = -(ErrReference)
	ErrRetSyntax    int = -(ErrSyntax)
	ErrRetType      int = -(ErrType)
	ErrRetURI       int = -(ErrURI)
)

const (
	ExecSuccess int = C.DUK_EXEC_SUCCESS
	ExecError   int = C.DUK_EXEC_ERROR
)

const (
	LogTrace int = iota
	LogDebug
	LogInfo
	LogWarn
	LogError
	LogFatal
)

const (
	BufObjArrayBuffer       int = C.DUK_BUFOBJ_ARRAYBUFFER
	BufObjNodejsBuffer      int = C.DUK_BUFOBJ_NODEJS_BUFFER
	BufObjDataView          int = C.DUK_BUFOBJ_DATAVIEW
	BufobjInt8Array         int = C.DUK_BUFOBJ_INT8ARRAY
	BufobjUint8Array        int = C.DUK_BUFOBJ_UINT8ARRAY
	BufobjUint8ClampedArray int = C.DUK_BUFOBJ_UINT8CLAMPEDARRAY
	BufObjInt16Array        int = C.DUK_BUFOBJ_INT16ARRAY
	BufObjUint16Array       int = C.DUK_BUFOBJ_UINT16ARRAY
	BufObjInt32Array        int = C.DUK_BUFOBJ_INT32ARRAY
	BufObjUint32Array       int = C.DUK_BUFOBJ_UINT32ARRAY
	BufObjFloat32Array      int = C.DUK_BUFOBJ_FLOAT32ARRAY
	BufObjFloat64Array      int = C.DUK_BUFOBJ_FLOAT64ARRAY
)
