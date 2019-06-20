package goja

import (
	"time"
)

const (
	dateTimeLayout       = "Mon Jan 02 2006 15:04:05 GMT-0700 (MST)"
	isoDateTimeLayout    = "2006-01-02T15:04:05.000Z"
	dateLayout           = "Mon Jan 02 2006"
	timeLayout           = "15:04:05 GMT-0700 (MST)"
	datetimeLayout_en_GB = "01/02/2006, 15:04:05"
	dateLayout_en_GB     = "01/02/2006"
	timeLayout_en_GB     = "15:04:05"
)

type dateObject struct {
	baseObject
	time  time.Time
	isSet bool
}

var (
	dateLayoutList = []string{
		"2006-01-02T15:04:05.000Z0700",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05Z0700",
		"2006-01-02T15:04:05",
		"2006-01-02",
		time.RFC1123,
		time.RFC1123Z,
		dateTimeLayout,
		time.UnixDate,
		time.ANSIC,
		time.RubyDate,
		"Mon, 02 Jan 2006 15:04:05 GMT-0700 (MST)",
		"Mon, 02 Jan 2006 15:04:05 -0700 (MST)",

		"2006",
		"2006-01",

		"2006T15:04",
		"2006-01T15:04",
		"2006-01-02T15:04",

		"2006T15:04:05",
		"2006-01T15:04:05",

		"2006T15:04:05.000",
		"2006-01T15:04:05.000",

		"2006T15:04Z0700",
		"2006-01T15:04Z0700",
		"2006-01-02T15:04Z0700",

		"2006T15:04:05Z0700",
		"2006-01T15:04:05Z0700",

		"2006T15:04:05.000Z0700",
		"2006-01T15:04:05.000Z0700",
	}
)

func dateParse(date string) (time.Time, bool) {
	var t time.Time
	var err error
	for _, layout := range dateLayoutList {
		t, err = parseDate(layout, date, time.UTC)
		if err == nil {
			break
		}
	}
	unix := timeToMsec(t)
	return t, err == nil && unix >= -8640000000000000 && unix <= 8640000000000000
}

func (r *Runtime) newDateObject(t time.Time, isSet bool) *Object {
	v := &Object{runtime: r}
	d := &dateObject{}
	v.self = d
	d.val = v
	d.class = classDate
	d.prototype = r.global.DatePrototype
	d.extensible = true
	d.init()
	d.time = t.In(time.Local)
	d.isSet = isSet
	return v
}

func dateFormat(t time.Time) string {
	return t.Local().Format(dateTimeLayout)
}

func (d *dateObject) toPrimitive() Value {
	return d.toPrimitiveString()
}

func (d *dateObject) export() interface{} {
	if d.isSet {
		return d.time
	}
	return nil
}
