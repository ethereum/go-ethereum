package duktape

const (
	CompileEval uint = 1 << iota
	CompileFunction
	CompileStrict
	CompileSafe
	CompileNoResult
	CompileNoSource
	CompileStrlen
)

const (
	TypeNone Type = iota
	TypeUndefined
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeObject
	TypeBuffer
	TypePointer
	TypeLightFunc
)

const (
	TypeMaskNone uint = 1 << iota
	TypeMaskUndefined
	TypeMaskNull
	TypeMaskBoolean
	TypeMaskNumber
	TypeMaskString
	TypeMaskObject
	TypeMaskBuffer
	TypeMaskPointer
	TypeMaskLightFunc
)

const (
	EnumIncludeNonenumerable uint = 1 << iota
	EnumIncludeInternal
	EnumOwnPropertiesOnly
	EnumArrayIndicesOnly
	EnumSortArrayIndices
	NoProxyBehavior
)

const (
	ErrNone int = 0

	// Internal to Duktape
	ErrUnimplemented int = 50 + iota
	ErrUnsupported
	ErrInternal
	ErrAlloc
	ErrAssertion
	ErrAPI
	ErrUncaughtError
)

const (
	// Common prototypes
	ErrError int = 1 + iota
	ErrEval
	ErrRange
	ErrReference
	ErrSyntax
	ErrType
	ErrURI
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
	ErrRetError int = -(ErrError + iota)
	ErrRetEval
	ErrRetRange
	ErrRetReference
	ErrRetSyntax
	ErrRetType
	ErrRetURI
)

const (
	ExecSuccess = iota
	ExecError
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
	BufobjDuktapeAuffer     = 0
	BufobjNodejsAuffer      = 1
	BufobjArraybuffer       = 2
	BufobjDataview          = 3
	BufobjInt8array         = 4
	BufobjUint8array        = 5
	BufobjUint8clampedarray = 6
	BufobjInt16array        = 7
	BufobjUint16array       = 8
	BufobjInt32array        = 9
	BufobjUint32array       = 10
	BufobjFloat32array      = 11
	BufobjFloat64array      = 12
)
