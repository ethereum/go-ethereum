package goja

import (
	"bytes"
	"github.com/dop251/goja/parser"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
	"math"
	"strings"
	"unicode/utf8"
)

func (r *Runtime) collator() *collate.Collator {
	collator := r._collator
	if collator == nil {
		collator = collate.New(language.Und)
		r._collator = collator
	}
	return collator
}

func (r *Runtime) builtin_String(call FunctionCall) Value {
	if len(call.Arguments) > 0 {
		arg := call.Arguments[0]
		if _, ok := arg.assertString(); ok {
			return arg
		}
		return arg.ToString()
	} else {
		return newStringValue("")
	}
}

func (r *Runtime) _newString(s valueString) *Object {
	v := &Object{runtime: r}

	o := &stringObject{}
	o.class = classString
	o.val = v
	o.extensible = true
	v.self = o
	o.prototype = r.global.StringPrototype
	if s != nil {
		o.value = s
	}
	o.init()
	return v
}

func (r *Runtime) builtin_newString(args []Value) *Object {
	var s valueString
	if len(args) > 0 {
		s = args[0].ToString()
	} else {
		s = stringEmpty
	}
	return r._newString(s)
}

func searchSubstringUTF8(str, search string) (ret [][]int) {
	searchPos := 0
	l := len(str)
	if searchPos < l {
		p := strings.Index(str[searchPos:], search)
		if p != -1 {
			p += searchPos
			searchPos = p + len(search)
			ret = append(ret, []int{p, searchPos})
		}
	}
	return
}

func (r *Runtime) stringproto_toStringValueOf(this Value, funcName string) Value {
	if str, ok := this.assertString(); ok {
		return str
	}
	if obj, ok := this.(*Object); ok {
		if strObj, ok := obj.self.(*stringObject); ok {
			return strObj.value
		}
	}
	r.typeErrorResult(true, "String.prototype.%s is called on incompatible receiver", funcName)
	return nil
}

func (r *Runtime) stringproto_toString(call FunctionCall) Value {
	return r.stringproto_toStringValueOf(call.This, "toString")
}

func (r *Runtime) stringproto_valueOf(call FunctionCall) Value {
	return r.stringproto_toStringValueOf(call.This, "valueOf")
}

func (r *Runtime) string_fromcharcode(call FunctionCall) Value {
	b := make([]byte, len(call.Arguments))
	for i, arg := range call.Arguments {
		chr := toUInt16(arg)
		if chr >= utf8.RuneSelf {
			bb := make([]uint16, len(call.Arguments))
			for j := 0; j < i; j++ {
				bb[j] = uint16(b[j])
			}
			bb[i] = chr
			i++
			for j, arg := range call.Arguments[i:] {
				bb[i+j] = toUInt16(arg)
			}
			return unicodeString(bb)
		}
		b[i] = byte(chr)
	}

	return asciiString(b)
}

func (r *Runtime) stringproto_charAt(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()
	pos := call.Argument(0).ToInteger()
	if pos < 0 || pos >= s.length() {
		return stringEmpty
	}
	return newStringValue(string(s.charAt(pos)))
}

func (r *Runtime) stringproto_charCodeAt(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()
	pos := call.Argument(0).ToInteger()
	if pos < 0 || pos >= s.length() {
		return _NaN
	}
	return intToValue(int64(s.charAt(pos) & 0xFFFF))
}

func (r *Runtime) stringproto_concat(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	strs := make([]valueString, len(call.Arguments)+1)
	strs[0] = call.This.ToString()
	_, allAscii := strs[0].(asciiString)
	totalLen := strs[0].length()
	for i, arg := range call.Arguments {
		s := arg.ToString()
		if allAscii {
			_, allAscii = s.(asciiString)
		}
		strs[i+1] = s
		totalLen += s.length()
	}

	if allAscii {
		buf := bytes.NewBuffer(make([]byte, 0, totalLen))
		for _, s := range strs {
			buf.WriteString(s.String())
		}
		return asciiString(buf.String())
	} else {
		buf := make([]uint16, totalLen)
		pos := int64(0)
		for _, s := range strs {
			switch s := s.(type) {
			case asciiString:
				for i := 0; i < len(s); i++ {
					buf[pos] = uint16(s[i])
					pos++
				}
			case unicodeString:
				copy(buf[pos:], s)
				pos += s.length()
			}
		}
		return unicodeString(buf)
	}
}

func (r *Runtime) stringproto_indexOf(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	value := call.This.ToString()
	target := call.Argument(0).ToString()
	pos := call.Argument(1).ToInteger()

	if pos < 0 {
		pos = 0
	} else {
		l := value.length()
		if pos > l {
			pos = l
		}
	}

	return intToValue(value.index(target, pos))
}

func (r *Runtime) stringproto_lastIndexOf(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	value := call.This.ToString()
	target := call.Argument(0).ToString()
	numPos := call.Argument(1).ToNumber()

	var pos int64
	if f, ok := numPos.assertFloat(); ok && math.IsNaN(f) {
		pos = value.length()
	} else {
		pos = numPos.ToInteger()
		if pos < 0 {
			pos = 0
		} else {
			l := value.length()
			if pos > l {
				pos = l
			}
		}
	}

	return intToValue(value.lastIndex(target, pos))
}

func (r *Runtime) stringproto_localeCompare(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	this := norm.NFD.String(call.This.String())
	that := norm.NFD.String(call.Argument(0).String())
	return intToValue(int64(r.collator().CompareString(this, that)))
}

func (r *Runtime) stringproto_match(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()
	regexp := call.Argument(0)
	var rx *regexpObject
	if regexp, ok := regexp.(*Object); ok {
		rx, _ = regexp.self.(*regexpObject)
	}

	if rx == nil {
		rx = r.builtin_newRegExp([]Value{regexp}).self.(*regexpObject)
	}

	if rx.global {
		rx.putStr("lastIndex", intToValue(0), false)
		var a []Value
		var previousLastIndex int64
		for {
			match, result := rx.execRegexp(s)
			if !match {
				break
			}
			thisIndex := rx.getStr("lastIndex").ToInteger()
			if thisIndex == previousLastIndex {
				previousLastIndex++
				rx.putStr("lastIndex", intToValue(previousLastIndex), false)
			} else {
				previousLastIndex = thisIndex
			}
			a = append(a, s.substring(int64(result[0]), int64(result[1])))
		}
		if len(a) == 0 {
			return _null
		}
		return r.newArrayValues(a)
	} else {
		return rx.exec(s)
	}
}

func (r *Runtime) stringproto_replace(call FunctionCall) Value {
	s := call.This.ToString()
	var str string
	var isASCII bool
	if astr, ok := s.(asciiString); ok {
		str = string(astr)
		isASCII = true
	} else {
		str = s.String()
	}
	searchValue := call.Argument(0)
	replaceValue := call.Argument(1)

	var found [][]int

	if searchValue, ok := searchValue.(*Object); ok {
		if regexp, ok := searchValue.self.(*regexpObject); ok {
			find := 1
			if regexp.global {
				find = -1
			}
			if isASCII {
				found = regexp.pattern.FindAllSubmatchIndexASCII(str, find)
			} else {
				found = regexp.pattern.FindAllSubmatchIndexUTF8(str, find)
			}
			if found == nil {
				return s
			}
		}
	}

	if found == nil {
		found = searchSubstringUTF8(str, searchValue.String())
	}

	if len(found) == 0 {
		return s
	}

	var buf bytes.Buffer
	lastIndex := 0

	var rcall func(FunctionCall) Value

	if replaceValue, ok := replaceValue.(*Object); ok {
		if c, ok := replaceValue.self.assertCallable(); ok {
			rcall = c
		}
	}

	if rcall != nil {
		for _, item := range found {
			if item[0] != lastIndex {
				buf.WriteString(str[lastIndex:item[0]])
			}
			matchCount := len(item) / 2
			argumentList := make([]Value, matchCount+2)
			for index := 0; index < matchCount; index++ {
				offset := 2 * index
				if item[offset] != -1 {
					if isASCII {
						argumentList[index] = asciiString(str[item[offset]:item[offset+1]])
					} else {
						argumentList[index] = newStringValue(str[item[offset]:item[offset+1]])
					}
				} else {
					argumentList[index] = _undefined
				}
			}
			argumentList[matchCount] = valueInt(item[0])
			argumentList[matchCount+1] = s
			replacement := rcall(FunctionCall{
				This:      _undefined,
				Arguments: argumentList,
			}).String()
			buf.WriteString(replacement)
			lastIndex = item[1]
		}
	} else {
		newstring := replaceValue.String()

		for _, item := range found {
			if item[0] != lastIndex {
				buf.WriteString(str[lastIndex:item[0]])
			}
			matches := len(item) / 2
			for i := 0; i < len(newstring); i++ {
				if newstring[i] == '$' && i < len(newstring)-1 {
					ch := newstring[i+1]
					switch ch {
					case '$':
						buf.WriteByte('$')
					case '`':
						buf.WriteString(str[0:item[0]])
					case '\'':
						buf.WriteString(str[item[1]:])
					case '&':
						buf.WriteString(str[item[0]:item[1]])
					default:
						matchNumber := 0
						l := 0
						for _, ch := range newstring[i+1:] {
							if ch >= '0' && ch <= '9' {
								m := matchNumber*10 + int(ch-'0')
								if m >= matches {
									break
								}
								matchNumber = m
								l++
							} else {
								break
							}
						}
						if l > 0 {
							offset := 2 * matchNumber
							if offset < len(item) && item[offset] != -1 {
								buf.WriteString(str[item[offset]:item[offset+1]])
							}
							i += l - 1
						} else {
							buf.WriteByte('$')
							buf.WriteByte(ch)
						}

					}
					i++
				} else {
					buf.WriteByte(newstring[i])
				}
			}
			lastIndex = item[1]
		}
	}

	if lastIndex != len(str) {
		buf.WriteString(str[lastIndex:])
	}

	return newStringValue(buf.String())
}

func (r *Runtime) stringproto_search(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()
	regexp := call.Argument(0)
	var rx *regexpObject
	if regexp, ok := regexp.(*Object); ok {
		rx, _ = regexp.self.(*regexpObject)
	}

	if rx == nil {
		rx = r.builtin_newRegExp([]Value{regexp}).self.(*regexpObject)
	}

	match, result := rx.execRegexp(s)
	if !match {
		return intToValue(-1)
	}
	return intToValue(int64(result[0]))
}

func (r *Runtime) stringproto_slice(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	l := s.length()
	start := call.Argument(0).ToInteger()
	var end int64
	if arg1 := call.Argument(1); arg1 != _undefined {
		end = arg1.ToInteger()
	} else {
		end = l
	}

	if start < 0 {
		start += l
		if start < 0 {
			start = 0
		}
	} else {
		if start > l {
			start = l
		}
	}

	if end < 0 {
		end += l
		if end < 0 {
			end = 0
		}
	} else {
		if end > l {
			end = l
		}
	}

	if end > start {
		return s.substring(start, end)
	}
	return stringEmpty
}

func (r *Runtime) stringproto_split(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	separatorValue := call.Argument(0)
	limitValue := call.Argument(1)
	limit := -1
	if limitValue != _undefined {
		limit = int(toUInt32(limitValue))
	}

	if limit == 0 {
		return r.newArrayValues(nil)
	}

	if separatorValue == _undefined {
		return r.newArrayValues([]Value{s})
	}

	var search *regexpObject
	if o, ok := separatorValue.(*Object); ok {
		search, _ = o.self.(*regexpObject)
	}

	if search != nil {
		targetLength := s.length()
		valueArray := []Value{}
		result := search.pattern.FindAllSubmatchIndex(s, -1)
		lastIndex := 0
		found := 0

		for _, match := range result {
			if match[0] == match[1] {
				// FIXME Ugh, this is a hack
				if match[0] == 0 || int64(match[0]) == targetLength {
					continue
				}
			}

			if lastIndex != match[0] {
				valueArray = append(valueArray, s.substring(int64(lastIndex), int64(match[0])))
				found++
			} else if lastIndex == match[0] {
				if lastIndex != -1 {
					valueArray = append(valueArray, stringEmpty)
					found++
				}
			}

			lastIndex = match[1]
			if found == limit {
				goto RETURN
			}

			captureCount := len(match) / 2
			for index := 1; index < captureCount; index++ {
				offset := index * 2
				var value Value
				if match[offset] != -1 {
					value = s.substring(int64(match[offset]), int64(match[offset+1]))
				} else {
					value = _undefined
				}
				valueArray = append(valueArray, value)
				found++
				if found == limit {
					goto RETURN
				}
			}
		}

		if found != limit {
			if int64(lastIndex) != targetLength {
				valueArray = append(valueArray, s.substring(int64(lastIndex), targetLength))
			} else {
				valueArray = append(valueArray, stringEmpty)
			}
		}

	RETURN:
		return r.newArrayValues(valueArray)

	} else {
		separator := separatorValue.String()

		excess := false
		str := s.String()
		if limit > len(str) {
			limit = len(str)
		}
		splitLimit := limit
		if limit > 0 {
			splitLimit = limit + 1
			excess = true
		}

		split := strings.SplitN(str, separator, splitLimit)

		if excess && len(split) > limit {
			split = split[:limit]
		}

		valueArray := make([]Value, len(split))
		for index, value := range split {
			valueArray[index] = newStringValue(value)
		}

		return r.newArrayValues(valueArray)
	}

}

func (r *Runtime) stringproto_substring(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	l := s.length()
	intStart := call.Argument(0).ToInteger()
	var intEnd int64
	if end := call.Argument(1); end != _undefined {
		intEnd = end.ToInteger()
	} else {
		intEnd = l
	}
	if intStart < 0 {
		intStart = 0
	} else if intStart > l {
		intStart = l
	}

	if intEnd < 0 {
		intEnd = 0
	} else if intEnd > l {
		intEnd = l
	}

	if intStart > intEnd {
		intStart, intEnd = intEnd, intStart
	}

	return s.substring(intStart, intEnd)
}

func (r *Runtime) stringproto_toLowerCase(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	return s.toLower()
}

func (r *Runtime) stringproto_toUpperCase(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	return s.toUpper()
}

func (r *Runtime) stringproto_trim(call FunctionCall) Value {
	r.checkObjectCoercible(call.This)
	s := call.This.ToString()

	return newStringValue(strings.Trim(s.String(), parser.WhitespaceChars))
}

func (r *Runtime) stringproto_substr(call FunctionCall) Value {
	s := call.This.ToString()
	start := call.Argument(0).ToInteger()
	var length int64
	sl := int64(s.length())
	if arg := call.Argument(1); arg != _undefined {
		length = arg.ToInteger()
	} else {
		length = sl
	}

	if start < 0 {
		start = max(sl+start, 0)
	}

	length = min(max(length, 0), sl-start)
	if length <= 0 {
		return stringEmpty
	}

	return s.substring(start, start+length)
}

func (r *Runtime) initString() {
	r.global.StringPrototype = r.builtin_newString([]Value{stringEmpty})

	o := r.global.StringPrototype.self
	o.(*stringObject).prototype = r.global.ObjectPrototype
	o._putProp("toString", r.newNativeFunc(r.stringproto_toString, nil, "toString", nil, 0), true, false, true)
	o._putProp("valueOf", r.newNativeFunc(r.stringproto_valueOf, nil, "valueOf", nil, 0), true, false, true)
	o._putProp("charAt", r.newNativeFunc(r.stringproto_charAt, nil, "charAt", nil, 1), true, false, true)
	o._putProp("charCodeAt", r.newNativeFunc(r.stringproto_charCodeAt, nil, "charCodeAt", nil, 1), true, false, true)
	o._putProp("concat", r.newNativeFunc(r.stringproto_concat, nil, "concat", nil, 1), true, false, true)
	o._putProp("indexOf", r.newNativeFunc(r.stringproto_indexOf, nil, "indexOf", nil, 1), true, false, true)
	o._putProp("lastIndexOf", r.newNativeFunc(r.stringproto_lastIndexOf, nil, "lastIndexOf", nil, 1), true, false, true)
	o._putProp("localeCompare", r.newNativeFunc(r.stringproto_localeCompare, nil, "localeCompare", nil, 1), true, false, true)
	o._putProp("match", r.newNativeFunc(r.stringproto_match, nil, "match", nil, 1), true, false, true)
	o._putProp("replace", r.newNativeFunc(r.stringproto_replace, nil, "replace", nil, 2), true, false, true)
	o._putProp("search", r.newNativeFunc(r.stringproto_search, nil, "search", nil, 1), true, false, true)
	o._putProp("slice", r.newNativeFunc(r.stringproto_slice, nil, "slice", nil, 2), true, false, true)
	o._putProp("split", r.newNativeFunc(r.stringproto_split, nil, "split", nil, 2), true, false, true)
	o._putProp("substring", r.newNativeFunc(r.stringproto_substring, nil, "substring", nil, 2), true, false, true)
	o._putProp("toLowerCase", r.newNativeFunc(r.stringproto_toLowerCase, nil, "toLowerCase", nil, 0), true, false, true)
	o._putProp("toLocaleLowerCase", r.newNativeFunc(r.stringproto_toLowerCase, nil, "toLocaleLowerCase", nil, 0), true, false, true)
	o._putProp("toUpperCase", r.newNativeFunc(r.stringproto_toUpperCase, nil, "toUpperCase", nil, 0), true, false, true)
	o._putProp("toLocaleUpperCase", r.newNativeFunc(r.stringproto_toUpperCase, nil, "toLocaleUpperCase", nil, 0), true, false, true)
	o._putProp("trim", r.newNativeFunc(r.stringproto_trim, nil, "trim", nil, 0), true, false, true)

	// Annex B
	o._putProp("substr", r.newNativeFunc(r.stringproto_substr, nil, "substr", nil, 2), true, false, true)

	r.global.String = r.newNativeFunc(r.builtin_String, r.builtin_newString, "String", r.global.StringPrototype, 1)
	o = r.global.String.self
	o._putProp("fromCharCode", r.newNativeFunc(r.string_fromcharcode, nil, "fromCharCode", nil, 1), true, false, true)

	r.addToGlobal("String", r.global.String)

	r.stringSingleton = r.builtin_new(r.global.String, nil).self.(*stringObject)
}
