package log

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-stack/stack"
	"github.com/golang/glog"
)

const timeKey = "t"
const lvlKey = "lvl"
const msgKey = "msg"
const ctxKey = "ctx"
const errorKey = "LOG15_ERROR"
const skipLevel = 2

type Lvl int

const (
	LvlCrit Lvl = iota
	LvlError
	LvlWarn
	LvlInfo
	LvlDebug
	LvlTrace
)

// AlignedString returns a 5-character string containing the name of a Lvl.
func (l Lvl) AlignedString() string {
	switch l {
	case LvlTrace:
		return "TRACE"
	case LvlDebug:
		return "DEBUG"
	case LvlInfo:
		return "INFO "
	case LvlWarn:
		return "WARN "
	case LvlError:
		return "ERROR"
	case LvlCrit:
		return "CRIT "
	default:
		panic("bad level")
	}
}

// Strings returns the name of a Lvl.
func (l Lvl) String() string {
	switch l {
	case LvlTrace:
		return "trce"
	case LvlDebug:
		return "dbug"
	case LvlInfo:
		return "info"
	case LvlWarn:
		return "warn"
	case LvlError:
		return "eror"
	case LvlCrit:
		return "crit"
	default:
		panic("bad level")
	}
}

// LvlFromString returns the appropriate Lvl from a string name.
// Useful for parsing command line args and configuration files.
func LvlFromString(lvlString string) (Lvl, error) {
	switch lvlString {
	case "trace", "trce":
		return LvlTrace, nil
	case "debug", "dbug":
		return LvlDebug, nil
	case "info":
		return LvlInfo, nil
	case "warn":
		return LvlWarn, nil
	case "error", "eror":
		return LvlError, nil
	case "crit":
		return LvlCrit, nil
	default:
		return LvlDebug, fmt.Errorf("Unknown level: %v", lvlString)
	}
}

// A Record is what a Logger asks its handler to write
type Record struct {
	Time     time.Time
	Lvl      Lvl
	Msg      string
	Ctx      []interface{}
	Call     stack.Call
	KeyNames RecordKeyNames
}

// RecordKeyNames gets stored in a Record when the write function is executed.
type RecordKeyNames struct {
	Time string
	Msg  string
	Lvl  string
	Ctx  string
}

// A Logger writes key/value pairs to a Handler
type Logger interface {
	// New returns a new Logger that has this logger's context plus the given context
	New(ctx ...interface{}) Logger

	// GetHandler gets the handler associated with the logger.
	GetHandler() Handler

	// SetHandler updates the logger to write records to the specified handler.
	SetHandler(h Handler)

	// Log a message at the given level with context key/value pairs
	Trace(msg string, ctx ...interface{})
	Debug(msg string, ctx ...interface{})
	Info(msg string, ctx ...interface{})
	Warn(msg string, ctx ...interface{})
	Error(msg string, ctx ...interface{})
	Crit(msg string, ctx ...interface{})
}

type logger struct {
	ctx []interface{}
	h   *swapHandler
}

func (l *logger) write(msg string, lvl Lvl, ctx []interface{}, skip int) {
	switch lvl {
	case LvlTrace:
		glog.V(3).Info(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	case LvlDebug:
		glog.V(2).Info(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	case LvlInfo:
		glog.Info(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	case LvlWarn:
		glog.Warning(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	case LvlError:
		glog.Error(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	case LvlCrit:
		glog.Fatal(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	default:
		glog.Info(getLogMsg(msg, newContext(l.ctx, ctx), skip))
	}
}

func (l *logger) New(ctx ...interface{}) Logger {
	child := &logger{newContext(l.ctx, ctx), new(swapHandler)}
	child.SetHandler(l.h)
	return child
}

func newContext(prefix []interface{}, suffix []interface{}) []interface{} {
	normalizedSuffix := normalize(suffix)
	newCtx := make([]interface{}, len(prefix)+len(normalizedSuffix))
	n := copy(newCtx, prefix)
	copy(newCtx[n:], normalizedSuffix)
	return newCtx
}

func (l *logger) Trace(msg string, ctx ...interface{}) {
	l.write(msg, LvlTrace, ctx, skipLevel)
}

func (l *logger) Debug(msg string, ctx ...interface{}) {
	l.write(msg, LvlDebug, ctx, skipLevel)
}

func (l *logger) Info(msg string, ctx ...interface{}) {
	l.write(msg, LvlInfo, ctx, skipLevel)
}

func (l *logger) Warn(msg string, ctx ...interface{}) {
	l.write(msg, LvlWarn, ctx, skipLevel)
}

func (l *logger) Error(msg string, ctx ...interface{}) {
	l.write(msg, LvlError, ctx, skipLevel)
}

func (l *logger) Crit(msg string, ctx ...interface{}) {
	l.write(msg, LvlCrit, ctx, skipLevel)
	os.Exit(1)
}

func (l *logger) GetHandler() Handler {
	return l.h.Get()
}

func (l *logger) SetHandler(h Handler) {
	l.h.Swap(h)
}

// getLogMsg method returns the log message in the following format:
// <Full path of origin> <padding> <Log message> <padding> <Context key & value>
func getLogMsg(msg string, ctx []interface{}, skip int) string {
	// locationEnabled is an atomic flag controlling whether the formatter should
	// append the log locations too when printing entries.
	atomic.StoreUint32(&locationEnabled, 0)
	// Format the location path and line number.
	location := fmt.Sprintf("%+v", stack.Caller(skip))
	align := int(atomic.LoadUint32(&locationLength))
	// Maintain the maximum location length for fancyer alignment.
	if align < len(location) {
		align = len(location)
		atomic.StoreUint32(&locationLength, uint32(align))
	}
	// Specifying padding based on the maximum length of the string representing
	// the location of the origin of the log.
	padding := strings.Repeat(" ", align-len(location))
	buf := &bytes.Buffer{}
	buf.WriteString(location)
	buf.WriteString(padding)
	buf.WriteString(msg)
	// Maintain the maximum log message length for fancyer alignment.
	if align < len(msg) {
		align = len(msg)
		atomic.StoreUint32(&locationLength, uint32(align))
	}
	// Specifying padding based on the maximum length of the string representing
	// the logged message.
	padding = strings.Repeat(" ", align-len(msg))
	buf.WriteString(padding)
	// Writing key-value pairs of the context into the buffer.
	logfmt(buf, ctx, 0, false)
	return string(buf.Bytes()[:])
}

func normalize(ctx []interface{}) []interface{} {
	// if the caller passed a Ctx object, then expand it
	if len(ctx) == 1 {
		if ctxMap, ok := ctx[0].(Ctx); ok {
			ctx = ctxMap.toArray()
		}
	}

	// ctx needs to be even because it's a series of key/value pairs
	// no one wants to check for errors on logging functions,
	// so instead of erroring on bad input, we'll just make sure
	// that things are the right length and users can fix bugs
	// when they see the output looks wrong
	if len(ctx)%2 != 0 {
		ctx = append(ctx, nil, errorKey, "Normalized odd number of arguments by adding nil")
	}

	return ctx
}

// Lazy allows you to defer calculation of a logged value that is expensive
// to compute until it is certain that it must be evaluated with the given filters.
//
// Lazy may also be used in conjunction with a Logger's New() function
// to generate a child logger which always reports the current value of changing
// state.
//
// You may wrap any function which takes no arguments to Lazy. It may return any
// number of values of any type.
type Lazy struct {
	Fn interface{}
}

// Ctx is a map of key/value pairs to pass as context to a log function
// Use this only if you really need greater safety around the arguments you pass
// to the logging functions.
type Ctx map[string]interface{}

func (c Ctx) toArray() []interface{} {
	arr := make([]interface{}, len(c)*2)

	i := 0
	for k, v := range c {
		arr[i] = k
		arr[i+1] = v
		i += 2
	}

	return arr
}
