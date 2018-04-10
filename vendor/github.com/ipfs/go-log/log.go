// Package log is the logging library used by IPFS
// (https://github.com/ipfs/go-ipfs). It uses a modified version of
// https://godoc.org/github.com/whyrusleeping/go-logging .
package log

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	opentrace "github.com/opentracing/opentracing-go"
	otExt "github.com/opentracing/opentracing-go/ext"
	otl "github.com/opentracing/opentracing-go/log"
)

// StandardLogger provides API compatibility with standard printf loggers
// eg. go-logging
type StandardLogger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
}

// EventLogger extends the StandardLogger interface to allow for log items
// containing structured metadata
type EventLogger interface {
	StandardLogger

	// Event merges structured data from the provided inputs into a single
	// machine-readable log event.
	//
	// If the context contains metadata, a copy of this is used as the base
	// metadata accumulator.
	//
	// If one or more loggable objects are provided, these are deep-merged into base blob.
	//
	// Next, the event name is added to the blob under the key "event". If
	// the key "event" already exists, it will be over-written.
	//
	// Finally the timestamp and package name are added to the accumulator and
	// the metadata is logged.
	Event(ctx context.Context, event string, m ...Loggable)

	EventBegin(ctx context.Context, event string, m ...Loggable) *EventInProgress

	EventBeginInContext(ctx context.Context, event string, m ...Loggable) context.Context
}

// Logger retrieves an event logger by name
func Logger(system string) EventLogger {

	// TODO if we would like to adjust log levels at run-time. Store this event
	// logger in a map (just like the util.Logger impl)
	if len(system) == 0 {
		setuplog := getLogger("setup-logger")
		setuplog.Warning("Missing name parameter")
		system = "undefined"
	}

	logger := getLogger(system)

	return &eventLogger{system: system, StandardLogger: logger}
}

// eventLogger implements the EventLogger and wraps a go-logging Logger
type eventLogger struct {
	StandardLogger

	system string
	// TODO add log-level
}

func (el *eventLogger) EventBegin(ctx context.Context, event string, metadata ...Loggable) *EventInProgress {
	start := time.Now()
	el.Event(ctx, fmt.Sprintf("%sBegin", event), metadata...)

	span, ctx := opentrace.StartSpanFromContext(ctx, event)

	eip := &EventInProgress{}
	eip.doneFunc = func(additional []Loggable) {
		metadata = append(metadata, additional...)                      // anything added during the operation
		metadata = append(metadata, LoggableMap(map[string]interface{}{ // finally, duration of event
			"duration": time.Now().Sub(start),
		}))

		el.Event(ctx, event, metadata...)
		if traceingDisabled() {
			return
		}
		otExt.Component.Set(span, el.system)
		for _, m := range metadata {
			for l, v := range m.Loggable() {
				if l == "error" {
					otExt.Error.Set(span, true)
				}
				f := getOpentracingField(l, v)
				span.LogFields(f)
			}
		}
		span.Finish()
	}
	return eip
}

type activeEventKeyType struct{}

var activeEventKey = activeEventKeyType{}

// Stores an event in a context to be finished at a later point
func (el *eventLogger) EventBeginInContext(ctx context.Context, event string, metadata ...Loggable) context.Context {
	eip := el.EventBegin(ctx, event, metadata...)
	return context.WithValue(ctx, activeEventKey, eip)
}

// Will complete an event if there is one in ctx
func MaybeFinishEvent(ctx context.Context) {
	val := ctx.Value(activeEventKey)
	if eip, ok := val.(*EventInProgress); ok {
		eip.Done()
	}
}

func (el *eventLogger) Event(ctx context.Context, event string, metadata ...Loggable) {

	// short circuit if theres nothing to write to
	if !WriterGroup.Active() {
		return
	}

	// Collect loggables for later logging
	var loggables []Loggable

	// get any existing metadata from the context
	existing, err := MetadataFromContext(ctx)
	if err != nil {
		existing = Metadata{}
	}
	loggables = append(loggables, existing)

	for _, datum := range metadata {
		loggables = append(loggables, datum)
	}

	e := entry{
		loggables: loggables,
		system:    el.system,
		event:     event,
	}

	accum := Metadata{}
	for _, loggable := range e.loggables {
		accum = DeepMerge(accum, loggable.Loggable())
	}

	// apply final attributes to reserved keys
	// TODO accum["level"] = level
	accum["event"] = e.event
	accum["system"] = e.system
	accum["time"] = FormatRFC3339(time.Now())

	out, err := json.Marshal(accum)
	if err != nil {
		el.Errorf("ERROR FORMATTING EVENT ENTRY: %s", err)
		return
	}

	WriterGroup.Write(append(out, '\n'))
}

// EventInProgress represent and event which is happening
type EventInProgress struct {
	loggables []Loggable
	doneFunc  func([]Loggable)
}

// Append adds loggables to be included in the call to Done
func (eip *EventInProgress) Append(l Loggable) {
	eip.loggables = append(eip.loggables, l)
}

// SetError includes the provided error
func (eip *EventInProgress) SetError(err error) {
	eip.loggables = append(eip.loggables, LoggableMap{
		"error": err.Error(),
	})
}

// Done creates a new Event entry that includes the duration and appended
// loggables.
func (eip *EventInProgress) Done() {
	eip.doneFunc(eip.loggables) // create final event with extra data
}

// DoneWithErr creates a new Event entry that includes the duration and appended
// loggables. DoneWithErr accepts an error, if err is non-nil, it is set on
// the EventInProgress. Otherwise the logic is the same as the `Done()` method
func (eip *EventInProgress) DoneWithErr(err error) {
	if err != nil {
		eip.SetError(err)
	}
	eip.doneFunc(eip.loggables)
}

// Close is an alias for done
func (eip *EventInProgress) Close() error {
	eip.Done()
	return nil
}

// FormatRFC3339 returns the given time in UTC with RFC3999Nano format.
func FormatRFC3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func getOpentracingField(l string, v interface{}) otl.Field {
	var f otl.Field
	switch v.(type) {
	case bool:
		f = otl.Bool(l, v.(bool))
	case string:
		f = otl.String(l, v.(string))
	case float32:
		f = otl.Float32(l, v.(float32))
	case float64:
		f = otl.Float64(l, v.(float64))
	case int:
		f = otl.Int(l, v.(int))
	case int32:
		f = otl.Int32(l, v.(int32))
	case int64:
		f = otl.Int64(l, v.(int64))
	case uint32:
		f = otl.Uint32(l, v.(uint32))
	case uint64:
		f = otl.Uint64(l, v.(uint64))
	default:
		f = otl.Object(l, v)
	}
	return f
}

func traceingDisabled() bool {
	maybeTracer := opentrace.GlobalTracer()
	switch maybeTracer.(type) {
	case opentrace.NoopTracer:
		return true
	default:
		return false
	}
}
