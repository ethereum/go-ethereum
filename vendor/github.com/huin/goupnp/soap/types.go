package soap

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// localLoc acts like time.Local for this package, but is faked out by the
	// unit tests to ensure that things stay constant (especially when running
	// this test in a place where local time is UTC which might mask bugs).
	localLoc = time.Local
)

func MarshalUi1(v uint8) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi1(s string) (uint8, error) {
	v, err := strconv.ParseUint(s, 10, 8)
	return uint8(v), err
}

func MarshalUi2(v uint16) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi2(s string) (uint16, error) {
	v, err := strconv.ParseUint(s, 10, 16)
	return uint16(v), err
}

func MarshalUi4(v uint32) (string, error) {
	return strconv.FormatUint(uint64(v), 10), nil
}

func UnmarshalUi4(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	return uint32(v), err
}

func MarshalUi8(v uint64) (string, error) {
	return strconv.FormatUint(v, 10), nil
}

func UnmarshalUi8(s string) (uint64, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	return uint64(v), err
}

func MarshalI1(v int8) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI1(s string) (int8, error) {
	v, err := strconv.ParseInt(s, 10, 8)
	return int8(v), err
}

func MarshalI2(v int16) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI2(s string) (int16, error) {
	v, err := strconv.ParseInt(s, 10, 16)
	return int16(v), err
}

func MarshalI4(v int32) (string, error) {
	return strconv.FormatInt(int64(v), 10), nil
}

func UnmarshalI4(s string) (int32, error) {
	v, err := strconv.ParseInt(s, 10, 32)
	return int32(v), err
}

func MarshalInt(v int64) (string, error) {
	return strconv.FormatInt(v, 10), nil
}

func UnmarshalInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func MarshalR4(v float32) (string, error) {
	return strconv.FormatFloat(float64(v), 'G', -1, 32), nil
}

func UnmarshalR4(s string) (float32, error) {
	v, err := strconv.ParseFloat(s, 32)
	return float32(v), err
}

func MarshalR8(v float64) (string, error) {
	return strconv.FormatFloat(v, 'G', -1, 64), nil
}

func UnmarshalR8(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	return float64(v), err
}

// MarshalFixed14_4 marshals float64 to SOAP "fixed.14.4" type.
func MarshalFixed14_4(v float64) (string, error) {
	if v >= 1e14 || v <= -1e14 {
		return "", fmt.Errorf("soap fixed14.4: value %v out of bounds", v)
	}
	return strconv.FormatFloat(v, 'f', 4, 64), nil
}

// UnmarshalFixed14_4 unmarshals float64 from SOAP "fixed.14.4" type.
func UnmarshalFixed14_4(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if v >= 1e14 || v <= -1e14 {
		return 0, fmt.Errorf("soap fixed14.4: value %q out of bounds", s)
	}
	return v, nil
}

// MarshalChar marshals rune to SOAP "char" type.
func MarshalChar(v rune) (string, error) {
	if v == 0 {
		return "", errors.New("soap char: rune 0 is not allowed")
	}
	return string(v), nil
}

// UnmarshalChar unmarshals rune from SOAP "char" type.
func UnmarshalChar(s string) (rune, error) {
	if len(s) == 0 {
		return 0, errors.New("soap char: got empty string")
	}
	r, n := utf8.DecodeRune([]byte(s))
	if n != len(s) {
		return 0, fmt.Errorf("soap char: value %q is not a single rune", s)
	}
	return r, nil
}

func MarshalString(v string) (string, error) {
	return v, nil
}

func UnmarshalString(v string) (string, error) {
	return v, nil
}

func parseInt(s string, err *error) int {
	v, parseErr := strconv.ParseInt(s, 10, 64)
	if parseErr != nil {
		*err = parseErr
	}
	return int(v)
}

var dateRegexps = []*regexp.Regexp{
	// yyyy[-mm[-dd]]
	regexp.MustCompile(`^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?$`),
	// yyyy[mm[dd]]
	regexp.MustCompile(`^(\d{4})(?:(\d{2})(?:(\d{2}))?)?$`),
}

func parseDateParts(s string) (year, month, day int, err error) {
	var parts []string
	for _, re := range dateRegexps {
		parts = re.FindStringSubmatch(s)
		if parts != nil {
			break
		}
	}
	if parts == nil {
		err = fmt.Errorf("soap date: value %q is not in a recognized ISO8601 date format", s)
		return
	}

	year = parseInt(parts[1], &err)
	month = 1
	day = 1
	if len(parts[2]) != 0 {
		month = parseInt(parts[2], &err)
		if len(parts[3]) != 0 {
			day = parseInt(parts[3], &err)
		}
	}

	if err != nil {
		err = fmt.Errorf("soap date: %q: %v", s, err)
	}

	return
}

var timeRegexps = []*regexp.Regexp{
	// hh[:mm[:ss]]
	regexp.MustCompile(`^(\d{2})(?::(\d{2})(?::(\d{2}))?)?$`),
	// hh[mm[ss]]
	regexp.MustCompile(`^(\d{2})(?:(\d{2})(?:(\d{2}))?)?$`),
}

func parseTimeParts(s string) (hour, minute, second int, err error) {
	var parts []string
	for _, re := range timeRegexps {
		parts = re.FindStringSubmatch(s)
		if parts != nil {
			break
		}
	}
	if parts == nil {
		err = fmt.Errorf("soap time: value %q is not in ISO8601 time format", s)
		return
	}

	hour = parseInt(parts[1], &err)
	if len(parts[2]) != 0 {
		minute = parseInt(parts[2], &err)
		if len(parts[3]) != 0 {
			second = parseInt(parts[3], &err)
		}
	}

	if err != nil {
		err = fmt.Errorf("soap time: %q: %v", s, err)
	}

	return
}

// (+|-)hh[[:]mm]
var timezoneRegexp = regexp.MustCompile(`^([+-])(\d{2})(?::?(\d{2}))?$`)

func parseTimezone(s string) (offset int, err error) {
	if s == "Z" {
		return 0, nil
	}
	parts := timezoneRegexp.FindStringSubmatch(s)
	if parts == nil {
		err = fmt.Errorf("soap timezone: value %q is not in ISO8601 timezone format", s)
		return
	}

	offset = parseInt(parts[2], &err) * 3600
	if len(parts[3]) != 0 {
		offset += parseInt(parts[3], &err) * 60
	}
	if parts[1] == "-" {
		offset = -offset
	}

	if err != nil {
		err = fmt.Errorf("soap timezone: %q: %v", s, err)
	}

	return
}

var completeDateTimeZoneRegexp = regexp.MustCompile(`^([^T]+)(?:T([^-+Z]+)(.+)?)?$`)

// splitCompleteDateTimeZone splits date, time and timezone apart from an
// ISO8601 string. It does not ensure that the contents of each part are
// correct, it merely splits on certain delimiters.
// e.g "2010-09-08T12:15:10+0700" => "2010-09-08", "12:15:10", "+0700".
// Timezone can only be present if time is also present.
func splitCompleteDateTimeZone(s string) (dateStr, timeStr, zoneStr string, err error) {
	parts := completeDateTimeZoneRegexp.FindStringSubmatch(s)
	if parts == nil {
		err = fmt.Errorf("soap date/time/zone: value %q is not in ISO8601 datetime format", s)
		return
	}
	dateStr = parts[1]
	timeStr = parts[2]
	zoneStr = parts[3]
	return
}

// MarshalDate marshals time.Time to SOAP "date" type. Note that this converts
// to local time, and discards the time-of-day components.
func MarshalDate(v time.Time) (string, error) {
	return v.In(localLoc).Format("2006-01-02"), nil
}

var dateFmts = []string{"2006-01-02", "20060102"}

// UnmarshalDate unmarshals time.Time from SOAP "date" type. This outputs the
// date as midnight in the local time zone.
func UnmarshalDate(s string) (time.Time, error) {
	year, month, day, err := parseDateParts(s)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, localLoc), nil
}

// TimeOfDay is used in cases where SOAP "time" or "time.tz" is used.
type TimeOfDay struct {
	// Duration of time since midnight.
	FromMidnight time.Duration

	// Set to true if Offset is specified. If false, then the timezone is
	// unspecified (and by ISO8601 - implies some "local" time).
	HasOffset bool

	// Offset is non-zero only if time.tz is used. It is otherwise ignored. If
	// non-zero, then it is regarded as a UTC offset in seconds. Note that the
	// sub-minutes is ignored by the marshal function.
	Offset int
}

// MarshalTimeOfDay marshals TimeOfDay to the "time" type.
func MarshalTimeOfDay(v TimeOfDay) (string, error) {
	d := int64(v.FromMidnight / time.Second)
	hour := d / 3600
	d = d % 3600
	minute := d / 60
	second := d % 60

	return fmt.Sprintf("%02d:%02d:%02d", hour, minute, second), nil
}

// UnmarshalTimeOfDay unmarshals TimeOfDay from the "time" type.
func UnmarshalTimeOfDay(s string) (TimeOfDay, error) {
	t, err := UnmarshalTimeOfDayTz(s)
	if err != nil {
		return TimeOfDay{}, err
	} else if t.HasOffset {
		return TimeOfDay{}, fmt.Errorf("soap time: value %q contains unexpected timezone", s)
	}
	return t, nil
}

// MarshalTimeOfDayTz marshals TimeOfDay to the "time.tz" type.
func MarshalTimeOfDayTz(v TimeOfDay) (string, error) {
	d := int64(v.FromMidnight / time.Second)
	hour := d / 3600
	d = d % 3600
	minute := d / 60
	second := d % 60

	tz := ""
	if v.HasOffset {
		if v.Offset == 0 {
			tz = "Z"
		} else {
			offsetMins := v.Offset / 60
			sign := '+'
			if offsetMins < 1 {
				offsetMins = -offsetMins
				sign = '-'
			}
			tz = fmt.Sprintf("%c%02d:%02d", sign, offsetMins/60, offsetMins%60)
		}
	}

	return fmt.Sprintf("%02d:%02d:%02d%s", hour, minute, second, tz), nil
}

// UnmarshalTimeOfDayTz unmarshals TimeOfDay from the "time.tz" type.
func UnmarshalTimeOfDayTz(s string) (tod TimeOfDay, err error) {
	zoneIndex := strings.IndexAny(s, "Z+-")
	var timePart string
	var hasOffset bool
	var offset int
	if zoneIndex == -1 {
		hasOffset = false
		timePart = s
	} else {
		hasOffset = true
		timePart = s[:zoneIndex]
		if offset, err = parseTimezone(s[zoneIndex:]); err != nil {
			return
		}
	}

	hour, minute, second, err := parseTimeParts(timePart)
	if err != nil {
		return
	}

	fromMidnight := time.Duration(hour*3600+minute*60+second) * time.Second

	// ISO8601 special case - values up to 24:00:00 are allowed, so using
	// strictly greater-than for the maximum value.
	if fromMidnight > 24*time.Hour || minute >= 60 || second >= 60 {
		return TimeOfDay{}, fmt.Errorf("soap time.tz: value %q has value(s) out of range", s)
	}

	return TimeOfDay{
		FromMidnight: time.Duration(hour*3600+minute*60+second) * time.Second,
		HasOffset:    hasOffset,
		Offset:       offset,
	}, nil
}

// MarshalDateTime marshals time.Time to SOAP "dateTime" type. Note that this
// converts to local time.
func MarshalDateTime(v time.Time) (string, error) {
	return v.In(localLoc).Format("2006-01-02T15:04:05"), nil
}

// UnmarshalDateTime unmarshals time.Time from the SOAP "dateTime" type. This
// returns a value in the local timezone.
func UnmarshalDateTime(s string) (result time.Time, err error) {
	dateStr, timeStr, zoneStr, err := splitCompleteDateTimeZone(s)
	if err != nil {
		return
	}

	if len(zoneStr) != 0 {
		err = fmt.Errorf("soap datetime: unexpected timezone in %q", s)
		return
	}

	year, month, day, err := parseDateParts(dateStr)
	if err != nil {
		return
	}

	var hour, minute, second int
	if len(timeStr) != 0 {
		hour, minute, second, err = parseTimeParts(timeStr)
		if err != nil {
			return
		}
	}

	result = time.Date(year, time.Month(month), day, hour, minute, second, 0, localLoc)
	return
}

// MarshalDateTimeTz marshals time.Time to SOAP "dateTime.tz" type.
func MarshalDateTimeTz(v time.Time) (string, error) {
	return v.Format("2006-01-02T15:04:05-07:00"), nil
}

// UnmarshalDateTimeTz unmarshals time.Time from the SOAP "dateTime.tz" type.
// This returns a value in the local timezone when the timezone is unspecified.
func UnmarshalDateTimeTz(s string) (result time.Time, err error) {
	dateStr, timeStr, zoneStr, err := splitCompleteDateTimeZone(s)
	if err != nil {
		return
	}

	year, month, day, err := parseDateParts(dateStr)
	if err != nil {
		return
	}

	var hour, minute, second int
	var location *time.Location = localLoc
	if len(timeStr) != 0 {
		hour, minute, second, err = parseTimeParts(timeStr)
		if err != nil {
			return
		}
		if len(zoneStr) != 0 {
			var offset int
			offset, err = parseTimezone(zoneStr)
			if offset == 0 {
				location = time.UTC
			} else {
				location = time.FixedZone("", offset)
			}
		}
	}

	result = time.Date(year, time.Month(month), day, hour, minute, second, 0, location)
	return
}

// MarshalBoolean marshals bool to SOAP "boolean" type.
func MarshalBoolean(v bool) (string, error) {
	if v {
		return "1", nil
	}
	return "0", nil
}

// UnmarshalBoolean unmarshals bool from the SOAP "boolean" type.
func UnmarshalBoolean(s string) (bool, error) {
	switch s {
	case "0", "false", "no":
		return false, nil
	case "1", "true", "yes":
		return true, nil
	}
	return false, fmt.Errorf("soap boolean: %q is not a valid boolean value", s)
}

// MarshalBinBase64 marshals []byte to SOAP "bin.base64" type.
func MarshalBinBase64(v []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(v), nil
}

// UnmarshalBinBase64 unmarshals []byte from the SOAP "bin.base64" type.
func UnmarshalBinBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// MarshalBinHex marshals []byte to SOAP "bin.hex" type.
func MarshalBinHex(v []byte) (string, error) {
	return hex.EncodeToString(v), nil
}

// UnmarshalBinHex unmarshals []byte from the SOAP "bin.hex" type.
func UnmarshalBinHex(s string) ([]byte, error) {
	return hex.DecodeString(s)
}

// MarshalURI marshals *url.URL to SOAP "uri" type.
func MarshalURI(v *url.URL) (string, error) {
	return v.String(), nil
}

// UnmarshalURI unmarshals *url.URL from the SOAP "uri" type.
func UnmarshalURI(s string) (*url.URL, error) {
	return url.Parse(s)
}
