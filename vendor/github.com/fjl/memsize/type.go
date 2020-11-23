package memsize

import (
	"fmt"
	"reflect"
)

// address is a memory location.
//
// Code dealing with uintptr is oblivious to the zero address.
// Code dealing with address is not: it treats the zero address
// as invalid. Offsetting an invalid address doesn't do anything.
//
// This distinction is useful because there are objects that we can't
// get the pointer to.
type address uintptr

const invalidAddr = address(0)

func (a address) valid() bool {
	return a != 0
}

func (a address) addOffset(off uintptr) address {
	if !a.valid() {
		return invalidAddr
	}
	return a + address(off)
}

func (a address) String() string {
	if uintptrBits == 32 {
		return fmt.Sprintf("%#0.8x", uintptr(a))
	}
	return fmt.Sprintf("%#0.16x", uintptr(a))
}

type typCache map[reflect.Type]typInfo

type typInfo struct {
	isPointer bool
	needScan  bool
}

// isPointer returns true for pointer-ish values. The notion of
// pointer includes everything but plain values, i.e. slices, maps
// channels, interfaces are 'pointer', too.
func (tc *typCache) isPointer(typ reflect.Type) bool {
	return tc.info(typ).isPointer
}

// needScan reports whether a value of the type needs to be scanned
// recursively because it may contain pointers.
func (tc *typCache) needScan(typ reflect.Type) bool {
	return tc.info(typ).needScan
}

func (tc *typCache) info(typ reflect.Type) typInfo {
	info, found := (*tc)[typ]
	switch {
	case found:
		return info
	case isPointer(typ):
		info = typInfo{true, true}
	default:
		info = typInfo{false, tc.checkNeedScan(typ)}
	}
	(*tc)[typ] = info
	return info
}

func (tc *typCache) checkNeedScan(typ reflect.Type) bool {
	switch k := typ.Kind(); k {
	case reflect.Struct:
		// Structs don't need scan if none of their fields need it.
		for i := 0; i < typ.NumField(); i++ {
			if tc.needScan(typ.Field(i).Type) {
				return true
			}
		}
	case reflect.Array:
		// Arrays don't need scan if their element type doesn't.
		return tc.needScan(typ.Elem())
	}
	return false
}

func isPointer(typ reflect.Type) bool {
	k := typ.Kind()
	switch {
	case k <= reflect.Complex128:
		return false
	case k == reflect.Array:
		return false
	case k >= reflect.Chan && k <= reflect.String:
		return true
	case k == reflect.Struct || k == reflect.UnsafePointer:
		return false
	default:
		unhandledKind(k)
		return false
	}
}

func unhandledKind(k reflect.Kind) {
	panic("unhandled kind " + k.String())
}

// HumanSize formats the given number of bytes as a readable string.
func HumanSize(bytes uintptr) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.3f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.3f MB", float64(bytes)/1024/1024)
	}
}
