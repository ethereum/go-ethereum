package goja

import (
	"fmt"
	"math"
	"time"
)

const (
	maxTime = 8.64e15
)

func timeFromMsec(msec int64) time.Time {
	sec := msec / 1000
	nsec := (msec % 1000) * 1e6
	return time.Unix(sec, nsec)
}

func timeToMsec(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond())/1e6
}

func (r *Runtime) makeDate(args []Value, loc *time.Location) (t time.Time, valid bool) {
	pick := func(index int, default_ int64) (int64, bool) {
		if index >= len(args) {
			return default_, true
		}
		value := args[index]
		if valueInt, ok := value.assertInt(); ok {
			return valueInt, true
		}
		valueFloat := value.ToFloat()
		if math.IsNaN(valueFloat) || math.IsInf(valueFloat, 0) {
			return 0, false
		}
		return int64(valueFloat), true
	}

	switch {
	case len(args) >= 2:
		var year, month, day, hour, minute, second, millisecond int64
		if year, valid = pick(0, 1900); !valid {
			return
		}
		if month, valid = pick(1, 0); !valid {
			return
		}
		if day, valid = pick(2, 1); !valid {
			return
		}
		if hour, valid = pick(3, 0); !valid {
			return
		}
		if minute, valid = pick(4, 0); !valid {
			return
		}
		if second, valid = pick(5, 0); !valid {
			return
		}
		if millisecond, valid = pick(6, 0); !valid {
			return
		}

		if year >= 0 && year <= 99 {
			year += 1900
		}

		t = time.Date(int(year), time.Month(int(month)+1), int(day), int(hour), int(minute), int(second), int(millisecond)*1e6, loc)
	case len(args) == 0:
		t = r.now()
		valid = true
	default: // one argument
		pv := toPrimitiveNumber(args[0])
		if val, ok := pv.assertString(); ok {
			return dateParse(val.String())
		}

		var n int64
		if i, ok := pv.assertInt(); ok {
			n = i
		} else if f, ok := pv.assertFloat(); ok {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return
			}
			if math.Abs(f) > maxTime {
				return
			}
			n = int64(f)
		} else {
			n = pv.ToInteger()
		}
		t = timeFromMsec(n)
		valid = true
	}
	msec := t.Unix()*1000 + int64(t.Nanosecond()/1e6)
	if msec < 0 {
		msec = -msec
	}
	if msec > maxTime {
		valid = false
	}
	return
}

func (r *Runtime) newDateTime(args []Value, loc *time.Location) *Object {
	t, isSet := r.makeDate(args, loc)
	return r.newDateObject(t, isSet)
}

func (r *Runtime) builtin_newDate(args []Value) *Object {
	return r.newDateTime(args, time.Local)
}

func (r *Runtime) builtin_date(call FunctionCall) Value {
	return asciiString(dateFormat(r.now()))
}

func (r *Runtime) date_parse(call FunctionCall) Value {
	t, set := dateParse(call.Argument(0).String())
	if set {
		return intToValue(timeToMsec(t))
	}
	return _NaN
}

func (r *Runtime) date_UTC(call FunctionCall) Value {
	t, valid := r.makeDate(call.Arguments, time.UTC)
	if !valid {
		return _NaN
	}
	return intToValue(timeToMsec(t))
}

func (r *Runtime) date_now(call FunctionCall) Value {
	return intToValue(timeToMsec(r.now()))
}

func (r *Runtime) dateproto_toString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(dateTimeLayout))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toUTCString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.In(time.UTC).Format(dateTimeLayout))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toUTCString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toISOString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			utc := d.time.In(time.UTC)
			year := utc.Year()
			if year >= -9999 && year <= 9999 {
				return asciiString(utc.Format(isoDateTimeLayout))
			}
			// extended year
			return asciiString(fmt.Sprintf("%+06d-", year) + utc.Format(isoDateTimeLayout[5:]))
		} else {
			panic(r.newError(r.global.RangeError, "Invalid time value"))
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toISOString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toJSON(call FunctionCall) Value {
	obj := r.toObject(call.This)
	tv := obj.self.toPrimitiveNumber()
	if f, ok := tv.assertFloat(); ok {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return _null
		}
	} else if _, ok := tv.assertInt(); !ok {
		return _null
	}

	if toISO, ok := obj.self.getStr("toISOString").(*Object); ok {
		if toISO, ok := toISO.self.assertCallable(); ok {
			return toISO(FunctionCall{
				This: obj,
			})
		}
	}

	r.typeErrorResult(true, "toISOString is not a function")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toDateString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(dateLayout))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toDateString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toTimeString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(timeLayout))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toTimeString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toLocaleString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(datetimeLayout_en_GB))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toLocaleString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toLocaleDateString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(dateLayout_en_GB))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toLocaleDateString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_toLocaleTimeString(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return asciiString(d.time.Format(timeLayout_en_GB))
		} else {
			return stringInvalidDate
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.toLocaleTimeString is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_valueOf(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(d.time.Unix()*1000 + int64(d.time.Nanosecond()/1e6))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.valueOf is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getTime(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getTime is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getFullYear(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Year()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getFullYear is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCFullYear(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Year()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCFullYear is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Month()) - 1)
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getMonth is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Month()) - 1)
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCMonth is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Hour()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getHours is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Hour()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCHours is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Day()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getDate is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Day()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCDate is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getDay(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Weekday()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getDay is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCDay(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Weekday()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCDay is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Minute()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getMinutes is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Minute()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCMinutes is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Second()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getSeconds is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Second()))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCSeconds is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.Nanosecond() / 1e6))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getMilliseconds is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getUTCMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			return intToValue(int64(d.time.In(time.UTC).Nanosecond() / 1e6))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getUTCMilliseconds is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_getTimezoneOffset(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			_, offset := d.time.Zone()
			return floatToValue(float64(-offset) / 60)
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.getTimezoneOffset is called on incompatible receiver")
	return nil
}

func (r *Runtime) dateproto_setTime(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		msec := call.Argument(0).ToInteger()
		d.time = timeFromMsec(msec)
		return intToValue(msec)
	}
	r.typeErrorResult(true, "Method Date.prototype.setTime is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			msec := call.Argument(0).ToInteger()
			m := timeToMsec(d.time) - int64(d.time.Nanosecond())/1e6 + msec
			d.time = timeFromMsec(m)
			return intToValue(m)
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setMilliseconds is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCMilliseconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			msec := call.Argument(0).ToInteger()
			m := timeToMsec(d.time) - int64(d.time.Nanosecond())/1e6 + msec
			d.time = timeFromMsec(m)
			return intToValue(m)
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCMilliseconds is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			sec := int(call.Argument(0).ToInteger())
			var nsec int
			if len(call.Arguments) > 1 {
				nsec = int(call.Arguments[1].ToInteger() * 1e6)
			} else {
				nsec = d.time.Nanosecond()
			}
			d.time = time.Date(d.time.Year(), d.time.Month(), d.time.Day(), d.time.Hour(), d.time.Minute(), sec, nsec, time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setSeconds is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCSeconds(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			sec := int(call.Argument(0).ToInteger())
			var nsec int
			t := d.time.In(time.UTC)
			if len(call.Arguments) > 1 {
				nsec = int(call.Arguments[1].ToInteger() * 1e6)
			} else {
				nsec = t.Nanosecond()
			}
			d.time = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), sec, nsec, time.UTC).In(time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCSeconds is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			min := int(call.Argument(0).ToInteger())
			var sec, nsec int
			if len(call.Arguments) > 1 {
				sec = int(call.Arguments[1].ToInteger())
			} else {
				sec = d.time.Second()
			}
			if len(call.Arguments) > 2 {
				nsec = int(call.Arguments[2].ToInteger() * 1e6)
			} else {
				nsec = d.time.Nanosecond()
			}
			d.time = time.Date(d.time.Year(), d.time.Month(), d.time.Day(), d.time.Hour(), min, sec, nsec, time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setMinutes is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCMinutes(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			min := int(call.Argument(0).ToInteger())
			var sec, nsec int
			t := d.time.In(time.UTC)
			if len(call.Arguments) > 1 {
				sec = int(call.Arguments[1].ToInteger())
			} else {
				sec = t.Second()
			}
			if len(call.Arguments) > 2 {
				nsec = int(call.Arguments[2].ToInteger() * 1e6)
			} else {
				nsec = t.Nanosecond()
			}
			d.time = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), min, sec, nsec, time.UTC).In(time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCMinutes is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			hour := int(call.Argument(0).ToInteger())
			var min, sec, nsec int
			if len(call.Arguments) > 1 {
				min = int(call.Arguments[1].ToInteger())
			} else {
				min = d.time.Minute()
			}
			if len(call.Arguments) > 2 {
				sec = int(call.Arguments[2].ToInteger())
			} else {
				sec = d.time.Second()
			}
			if len(call.Arguments) > 3 {
				nsec = int(call.Arguments[3].ToInteger() * 1e6)
			} else {
				nsec = d.time.Nanosecond()
			}
			d.time = time.Date(d.time.Year(), d.time.Month(), d.time.Day(), hour, min, sec, nsec, time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setHours is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCHours(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			hour := int(call.Argument(0).ToInteger())
			var min, sec, nsec int
			t := d.time.In(time.UTC)
			if len(call.Arguments) > 1 {
				min = int(call.Arguments[1].ToInteger())
			} else {
				min = t.Minute()
			}
			if len(call.Arguments) > 2 {
				sec = int(call.Arguments[2].ToInteger())
			} else {
				sec = t.Second()
			}
			if len(call.Arguments) > 3 {
				nsec = int(call.Arguments[3].ToInteger() * 1e6)
			} else {
				nsec = t.Nanosecond()
			}
			d.time = time.Date(d.time.Year(), d.time.Month(), d.time.Day(), hour, min, sec, nsec, time.UTC).In(time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCHours is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			d.time = time.Date(d.time.Year(), d.time.Month(), int(call.Argument(0).ToInteger()), d.time.Hour(), d.time.Minute(), d.time.Second(), d.time.Nanosecond(), time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setDate is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCDate(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			t := d.time.In(time.UTC)
			d.time = time.Date(t.Year(), t.Month(), int(call.Argument(0).ToInteger()), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC).In(time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCDate is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			month := time.Month(int(call.Argument(0).ToInteger()) + 1)
			var day int
			if len(call.Arguments) > 1 {
				day = int(call.Arguments[1].ToInteger())
			} else {
				day = d.time.Day()
			}
			d.time = time.Date(d.time.Year(), month, day, d.time.Hour(), d.time.Minute(), d.time.Second(), d.time.Nanosecond(), time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setMonth is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCMonth(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if d.isSet {
			month := time.Month(int(call.Argument(0).ToInteger()) + 1)
			var day int
			t := d.time.In(time.UTC)
			if len(call.Arguments) > 1 {
				day = int(call.Arguments[1].ToInteger())
			} else {
				day = t.Day()
			}
			d.time = time.Date(t.Year(), month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC).In(time.Local)
			return intToValue(timeToMsec(d.time))
		} else {
			return _NaN
		}
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCMonth is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setFullYear(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if !d.isSet {
			d.time = time.Unix(0, 0)
		}
		year := int(call.Argument(0).ToInteger())
		var month time.Month
		var day int
		if len(call.Arguments) > 1 {
			month = time.Month(call.Arguments[1].ToInteger() + 1)
		} else {
			month = d.time.Month()
		}
		if len(call.Arguments) > 2 {
			day = int(call.Arguments[2].ToInteger())
		} else {
			day = d.time.Day()
		}
		d.time = time.Date(year, month, day, d.time.Hour(), d.time.Minute(), d.time.Second(), d.time.Nanosecond(), time.Local)
		return intToValue(timeToMsec(d.time))
	}
	r.typeErrorResult(true, "Method Date.prototype.setFullYear is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) dateproto_setUTCFullYear(call FunctionCall) Value {
	obj := r.toObject(call.This)
	if d, ok := obj.self.(*dateObject); ok {
		if !d.isSet {
			d.time = time.Unix(0, 0)
		}
		year := int(call.Argument(0).ToInteger())
		var month time.Month
		var day int
		t := d.time.In(time.UTC)
		if len(call.Arguments) > 1 {
			month = time.Month(call.Arguments[1].ToInteger() + 1)
		} else {
			month = t.Month()
		}
		if len(call.Arguments) > 2 {
			day = int(call.Arguments[2].ToInteger())
		} else {
			day = t.Day()
		}
		d.time = time.Date(year, month, day, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC).In(time.Local)
		return intToValue(timeToMsec(d.time))
	}
	r.typeErrorResult(true, "Method Date.prototype.setUTCFullYear is called on incompatible receiver")
	panic("Unreachable")
}

func (r *Runtime) createDateProto(val *Object) objectImpl {
	o := &baseObject{
		class:      classObject,
		val:        val,
		extensible: true,
		prototype:  r.global.ObjectPrototype,
	}
	o.init()

	o._putProp("constructor", r.global.Date, true, false, true)
	o._putProp("toString", r.newNativeFunc(r.dateproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("toDateString", r.newNativeFunc(r.dateproto_toDateString, nil, "toDateString", nil, 0), true, false, true)
	o._putProp("toTimeString", r.newNativeFunc(r.dateproto_toTimeString, nil, "toTimeString", nil, 0), true, false, true)
	o._putProp("toLocaleString", r.newNativeFunc(r.dateproto_toLocaleString, nil, "toLocaleString", nil, 0), true, false, true)
	o._putProp("toLocaleDateString", r.newNativeFunc(r.dateproto_toLocaleDateString, nil, "toLocaleDateString", nil, 0), true, false, true)
	o._putProp("toLocaleTimeString", r.newNativeFunc(r.dateproto_toLocaleTimeString, nil, "toLocaleTimeString", nil, 0), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.dateproto_valueOf, nil, "valueOf", nil, 0), true, false, true)
	o._putProp("getTime", r.newNativeFunc(r.dateproto_getTime, nil, "getTime", nil, 0), true, false, true)
	o._putProp("getFullYear", r.newNativeFunc(r.dateproto_getFullYear, nil, "getFullYear", nil, 0), true, false, true)
	o._putProp("getUTCFullYear", r.newNativeFunc(r.dateproto_getUTCFullYear, nil, "getUTCFullYear", nil, 0), true, false, true)
	o._putProp("getMonth", r.newNativeFunc(r.dateproto_getMonth, nil, "getMonth", nil, 0), true, false, true)
	o._putProp("getUTCMonth", r.newNativeFunc(r.dateproto_getUTCMonth, nil, "getUTCMonth", nil, 0), true, false, true)
	o._putProp("getDate", r.newNativeFunc(r.dateproto_getDate, nil, "getDate", nil, 0), true, false, true)
	o._putProp("getUTCDate", r.newNativeFunc(r.dateproto_getUTCDate, nil, "getUTCDate", nil, 0), true, false, true)
	o._putProp("getDay", r.newNativeFunc(r.dateproto_getDay, nil, "getDay", nil, 0), true, false, true)
	o._putProp("getUTCDay", r.newNativeFunc(r.dateproto_getUTCDay, nil, "getUTCDay", nil, 0), true, false, true)
	o._putProp("getHours", r.newNativeFunc(r.dateproto_getHours, nil, "getHours", nil, 0), true, false, true)
	o._putProp("getUTCHours", r.newNativeFunc(r.dateproto_getUTCHours, nil, "getUTCHours", nil, 0), true, false, true)
	o._putProp("getMinutes", r.newNativeFunc(r.dateproto_getMinutes, nil, "getMinutes", nil, 0), true, false, true)
	o._putProp("getUTCMinutes", r.newNativeFunc(r.dateproto_getUTCMinutes, nil, "getUTCMinutes", nil, 0), true, false, true)
	o._putProp("getSeconds", r.newNativeFunc(r.dateproto_getSeconds, nil, "getSeconds", nil, 0), true, false, true)
	o._putProp("getUTCSeconds", r.newNativeFunc(r.dateproto_getUTCSeconds, nil, "getUTCSeconds", nil, 0), true, false, true)
	o._putProp("getMilliseconds", r.newNativeFunc(r.dateproto_getMilliseconds, nil, "getMilliseconds", nil, 0), true, false, true)
	o._putProp("getUTCMilliseconds", r.newNativeFunc(r.dateproto_getUTCMilliseconds, nil, "getUTCMilliseconds", nil, 0), true, false, true)
	o._putProp("getTimezoneOffset", r.newNativeFunc(r.dateproto_getTimezoneOffset, nil, "getTimezoneOffset", nil, 0), true, false, true)
	o._putProp("setTime", r.newNativeFunc(r.dateproto_setTime, nil, "setTime", nil, 1), true, false, true)
	o._putProp("setMilliseconds", r.newNativeFunc(r.dateproto_setMilliseconds, nil, "setMilliseconds", nil, 1), true, false, true)
	o._putProp("setUTCMilliseconds", r.newNativeFunc(r.dateproto_setUTCMilliseconds, nil, "setUTCMilliseconds", nil, 1), true, false, true)
	o._putProp("setSeconds", r.newNativeFunc(r.dateproto_setSeconds, nil, "setSeconds", nil, 2), true, false, true)
	o._putProp("setUTCSeconds", r.newNativeFunc(r.dateproto_setUTCSeconds, nil, "setUTCSeconds", nil, 2), true, false, true)
	o._putProp("setMinutes", r.newNativeFunc(r.dateproto_setMinutes, nil, "setMinutes", nil, 3), true, false, true)
	o._putProp("setUTCMinutes", r.newNativeFunc(r.dateproto_setUTCMinutes, nil, "setUTCMinutes", nil, 3), true, false, true)
	o._putProp("setHours", r.newNativeFunc(r.dateproto_setHours, nil, "setHours", nil, 4), true, false, true)
	o._putProp("setUTCHours", r.newNativeFunc(r.dateproto_setUTCHours, nil, "setUTCHours", nil, 4), true, false, true)
	o._putProp("setDate", r.newNativeFunc(r.dateproto_setDate, nil, "setDate", nil, 1), true, false, true)
	o._putProp("setUTCDate", r.newNativeFunc(r.dateproto_setUTCDate, nil, "setUTCDate", nil, 1), true, false, true)
	o._putProp("setMonth", r.newNativeFunc(r.dateproto_setMonth, nil, "setMonth", nil, 2), true, false, true)
	o._putProp("setUTCMonth", r.newNativeFunc(r.dateproto_setUTCMonth, nil, "setUTCMonth", nil, 2), true, false, true)
	o._putProp("setFullYear", r.newNativeFunc(r.dateproto_setFullYear, nil, "setFullYear", nil, 3), true, false, true)
	o._putProp("setUTCFullYear", r.newNativeFunc(r.dateproto_setUTCFullYear, nil, "setUTCFullYear", nil, 3), true, false, true)
	o._putProp("toUTCString", r.newNativeFunc(r.dateproto_toUTCString, nil, "toUTCString", nil, 0), true, false, true)
	o._putProp("toISOString", r.newNativeFunc(r.dateproto_toISOString, nil, "toISOString", nil, 0), true, false, true)
	o._putProp("toJSON", r.newNativeFunc(r.dateproto_toJSON, nil, "toJSON", nil, 1), true, false, true)

	return o
}

func (r *Runtime) createDate(val *Object) objectImpl {
	o := r.newNativeFuncObj(val, r.builtin_date, r.builtin_newDate, "Date", r.global.DatePrototype, 7)

	o._putProp("parse", r.newNativeFunc(r.date_parse, nil, "parse", nil, 1), true, false, true)
	o._putProp("UTC", r.newNativeFunc(r.date_UTC, nil, "UTC", nil, 7), true, false, true)
	o._putProp("now", r.newNativeFunc(r.date_now, nil, "now", nil, 0), true, false, true)

	return o
}

func (r *Runtime) newLazyObject(create func(*Object) objectImpl) *Object {
	val := &Object{runtime: r}
	o := &lazyObject{
		val:    val,
		create: create,
	}
	val.self = o
	return val
}

func (r *Runtime) initDate() {
	//r.global.DatePrototype = r.newObject()
	//o := r.global.DatePrototype.self
	r.global.DatePrototype = r.newLazyObject(r.createDateProto)

	//r.global.Date = r.newNativeFunc(r.builtin_date, r.builtin_newDate, "Date", r.global.DatePrototype, 7)
	//o := r.global.Date.self
	r.global.Date = r.newLazyObject(r.createDate)

	r.addToGlobal("Date", r.global.Date)
}
