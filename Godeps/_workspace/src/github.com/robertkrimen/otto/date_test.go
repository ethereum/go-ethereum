package otto

import (
	"math"
	"testing"
	"time"
)

func mockTimeLocal(location *time.Location) func() {
	local := time.Local
	time.Local = location
	return func() {
		time.Local = local
	}
}

// Passing or failing should not be dependent on what time zone we're in
func mockUTC() func() {
	return mockTimeLocal(time.UTC)
}

func TestDate(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		time0 := time.Unix(1348616313, 47*1000*1000).Local()

		test(`Date`, "function Date() { [native code] }")
		test(`new Date(0).toUTCString()`, "Thu, 01 Jan 1970 00:00:00 UTC")
		test(`new Date(0).toGMTString()`, "Thu, 01 Jan 1970 00:00:00 GMT")
		if false {
			// TODO toLocale{Date,Time}String
			test(`new Date(0).toLocaleString()`, "")
			test(`new Date(0).toLocaleDateString()`, "")
			test(`new Date(0).toLocaleTimeString()`, "")
		}
		test(`new Date(1348616313).getTime()`, 1348616313)
		test(`new Date(1348616313).toUTCString()`, "Fri, 16 Jan 1970 14:36:56 UTC")
		test(`abc = new Date(1348616313047); abc.toUTCString()`, "Tue, 25 Sep 2012 23:38:33 UTC")
		test(`abc.getYear()`, time0.Year()-1900)
		test(`abc.getFullYear()`, time0.Year())
		test(`abc.getUTCFullYear()`, 2012)
		test(`abc.getMonth()`, int(time0.Month())-1) // Remember, the JavaScript month is 0-based
		test(`abc.getUTCMonth()`, 8)
		test(`abc.getDate()`, time0.Day())
		test(`abc.getUTCDate()`, 25)
		test(`abc.getDay()`, int(time0.Weekday()))
		test(`abc.getUTCDay()`, 2)
		test(`abc.getHours()`, time0.Hour())
		test(`abc.getUTCHours()`, 23)
		test(`abc.getMinutes()`, time0.Minute())
		test(`abc.getUTCMinutes()`, 38)
		test(`abc.getSeconds()`, time0.Second())
		test(`abc.getUTCSeconds()`, 33)
		test(`abc.getMilliseconds()`, time0.Nanosecond()/(1000*1000)) // In honor of the 47%
		test(`abc.getUTCMilliseconds()`, 47)
		_, offset := time0.Zone()
		test(`abc.getTimezoneOffset()`, offset/-60)

		test(`new Date("Xyzzy").getTime()`, math.NaN())

		test(`abc.setFullYear(2011); abc.toUTCString()`, "Sun, 25 Sep 2011 23:38:33 UTC")
		test(`new Date(12564504e5).toUTCString()`, "Sun, 25 Oct 2009 06:00:00 UTC")
		test(`new Date(2009, 9, 25).toUTCString()`, "Sun, 25 Oct 2009 00:00:00 UTC")
		test(`+(new Date(2009, 9, 25))`, 1256428800000)

		format := "Mon, 2 Jan 2006 15:04:05 MST"

		time1 := time.Unix(1256450400, 0)
		time0 = time.Date(time1.Year(), time1.Month(), time1.Day(), time1.Hour(), time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()

		time0 = time.Date(time1.Year(), time1.Month(), time1.Day(), time1.Hour(), time1.Minute(), time1.Second(), 2001*1000*1000, time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setMilliseconds(2001); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), time1.Month(), time1.Day(), time1.Hour(), time1.Minute(), 61, time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setSeconds("61"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), time1.Month(), time1.Day(), time1.Hour(), 61, time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setMinutes("61"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), time1.Month(), time1.Day(), 5, time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setHours("5"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), time1.Month(), 26, time1.Hour(), time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setDate("26"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), 10, time1.Day(), time1.Hour(), time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setMonth(9); abc.toUTCString()`, time0.Format(format))
		test(`abc = new Date(12564504e5); abc.setMonth("09"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(time1.Year(), 11, time1.Day(), time1.Hour(), time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setMonth("10"); abc.toUTCString()`, time0.Format(format))

		time0 = time.Date(2010, time1.Month(), time1.Day(), time1.Hour(), time1.Minute(), time1.Second(), time1.Nanosecond(), time1.Location()).UTC()
		test(`abc = new Date(12564504e5); abc.setFullYear(2010); abc.toUTCString()`, time0.Format(format))

		test(`new Date("2001-01-01T10:01:02.000").getTime()`, 978343262000)

		// Date()
		test(`typeof Date()`, "string")
		test(`typeof Date(2006, 1, 2)`, "string")

		test(`
            abc = Object.getOwnPropertyDescriptor(Date, "parse");
            [ abc.value === Date.parse, abc.writable, abc.enumerable, abc.configurable ];
        `, "true,true,false,true")

		test(`
            abc = Object.getOwnPropertyDescriptor(Date.prototype, "toTimeString");
            [ abc.value === Date.prototype.toTimeString, abc.writable, abc.enumerable, abc.configurable ];
        `, "true,true,false,true")

		test(`
            var abc = Object.getOwnPropertyDescriptor(Date, "prototype");
            [   [ typeof Date.prototype ],
                [ abc.writable, abc.enumerable, abc.configurable ] ];
        `, "object,false,false,false")
	})
}

func TestDate_parse(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`Date.parse("2001-01-01T10:01:02.000")`, 978343262000)

		test(`Date.parse("2006-01-02T15:04:05.000")`, 1136214245000)

		test(`Date.parse("2006")`, 1136073600000)

		test(`Date.parse("1970-01-16T14:36:56+00:00")`, 1348616000)

		test(`Date.parse("1970-01-16T14:36:56.313+00:00")`, 1348616313)

		test(`Date.parse("1970-01-16T14:36:56.000")`, 1348616000)

		test(`Date.parse.length`, 1)
	})
}

func TestDate_UTC(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`Date.UTC(2009, 9, 25)`, 1256428800000)

		test(`Date.UTC.length`, 7)
	})
}

func TestDate_now(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		// FIXME I think this too risky
		test(`+(""+Date.now()).substr(0, 10)`, float64(epochToInteger(timeToEpoch(time.Now()))/1000))

		test(`Date.now() - Date.now(1,2,3) < 24 * 60 * 60`, true)
	})
}

func TestDate_toISOString(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`new Date(0).toISOString()`, "1970-01-01T00:00:00.000Z")
	})
}

func TestDate_toJSON(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`new Date(0).toJSON()`, "1970-01-01T00:00:00.000Z")
	})
}

func TestDate_setYear(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`new Date(12564504e5).setYear(96)`, 846223200000)

		test(`new Date(12564504e5).setYear(1996)`, 846223200000)

		test(`new Date(12564504e5).setYear(2000)`, 972453600000)
	})
}

func TestDateDefaultValue(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
        var date = new Date();
        date + 0 === date.toString() + "0";
    `, true)
	})
}

func TestDate_April1978(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            var abc = new Date(1978,3);
            [ abc.getYear(), abc.getMonth(), abc.valueOf() ];
        `, "78,3,260236800000")
	})
}

func TestDate_setMilliseconds(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date();
            def = abc.setMilliseconds();
            [ abc, def ];
        `, "Invalid Date,NaN")
	})
}

func TestDate_new(t *testing.T) {
	// FIXME?
	// This is probably incorrect, due to differences in Go date/time handling
	// versus ECMA date/time handling, but we'll leave this here for
	// future reference

	if true {
		return
	}

	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            [
                new Date(1899, 11).valueOf(),
                new Date(1899, 12).valueOf(),
                new Date(1900, 0).valueOf()
            ]
        `, "-2211638400000,-2208960000000,-2208960000000")
	})
}

func TestDateComparison(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            var now0 = Date.now();
            var now1 = (new Date()).toString();
            [ now0 === now1, Math.abs(now0 - Date.parse(now1)) <= 1000 ];
        `, "false,true")
	})
}

func TestDate_setSeconds(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setSeconds(10, 12);

            def.setSeconds(10);
            def.setMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setUTCSeconds(10, 12);

            def.setUTCSeconds(10);
            def.setUTCMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`Date.prototype.setSeconds.length`, 2)
		test(`Date.prototype.setUTCSeconds.length`, 2)
	})
}

func TestDate_setMinutes(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setMinutes(8, 10, 12);

            def.setMinutes(8);
            def.setSeconds(10);
            def.setMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setUTCMinutes(8, 10, 12);

            def.setUTCMinutes(8);
            def.setUTCSeconds(10);
            def.setUTCMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`Date.prototype.setMinutes.length`, 3)
		test(`Date.prototype.setUTCMinutes.length`, 3)
	})
}

func TestDate_setHours(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setHours(6, 8, 10, 12);

            def.setHours(6);
            def.setMinutes(8);
            def.setSeconds(10);
            def.setMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setUTCHours(6, 8, 10, 12);

            def.setUTCHours(6);
            def.setUTCMinutes(8);
            def.setUTCSeconds(10);
            def.setUTCMilliseconds(12);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`Date.prototype.setHours.length`, 4)
		test(`Date.prototype.setUTCHours.length`, 4)
	})
}

func TestDate_setMonth(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setMonth(6, 8);

            def.setMonth(6);
            def.setDate(8);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setUTCMonth(6, 8);

            def.setUTCMonth(6);
            def.setUTCDate(8);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`Date.prototype.setMonth.length`, 2)
		test(`Date.prototype.setUTCMonth.length`, 2)
	})
}

func TestDate_setFullYear(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setFullYear(1981, 6, 8);

            def.setFullYear(1981);
            def.setMonth(6);
            def.setDate(8);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`
            abc = new Date(1980, 10);
            def = new Date(abc);

            abc.setUTCFullYear(1981, 6, 8);

            def.setUTCFullYear(1981);
            def.setUTCMonth(6);
            def.setUTCDate(8);

            abc.valueOf() === def.valueOf();
        `, true)

		test(`Date.prototype.setFullYear.length`, 3)
		test(`Date.prototype.setUTCFullYear.length`, 3)
	})
}

func TestDate_setTime(t *testing.T) {
	tt(t, func() {
		test, _ := test()

		defer mockUTC()()

		test(`
            var abc = new Date(1999, 6, 1);
            var def = new Date();
            def.setTime(abc.getTime());
            [ def, abc.valueOf() == def.valueOf() ];
        `, "Thu, 01 Jul 1999 00:00:00 UTC,true")

		test(`Date.prototype.setTime.length`, 1)
	})
}
