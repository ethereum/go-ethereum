package bn256

import (
	"fmt"

	fuzz "github.com/google/gofuzz"
)

func Fuzz(data []byte) int {
	var gfpTest gfP
	f := fuzz.NewFromGoFuzz(data)
	f.Fuzz(&gfpTest)
	zInvVar := &gfP{}
	zInvVar.InvertVariableTime(&gfpTest)
	zInv := &gfP{}
	zInv.Invert(&gfpTest)
	if !gfpEq(zInv, zInvVar) {
		panic(fmt.Sprintf("invalid invert: %v %v", zInv, zInvVar))
	}
	return 0
}

func Fuzz2(data []byte) int {
	var gfpTest gfP2
	f := fuzz.NewFromGoFuzz(data)
	f.Fuzz(&gfpTest)
	zInvVar := &gfP2{}
	zInvVar.InvertVariableTime(&gfpTest)
	zInv := &gfP2{}
	zInv.Invert(&gfpTest)
	if !gfp2Eq(zInv, zInvVar) {
		panic(fmt.Sprintf("invalid invert: %v %v", zInv, zInvVar))
	}
	return 0
}

func Fuzz3(data []byte) int {
	var gfpTest gfP6
	f := fuzz.NewFromGoFuzz(data)
	f.Fuzz(&gfpTest)
	zInvVar := &gfP6{}
	zInvVar.InvertVariableTime(&gfpTest)
	zInv := &gfP6{}
	zInv.Invert(&gfpTest)
	if !gfp6Eq(zInv, zInvVar) {
		panic(fmt.Sprintf("invalid invert: %v %v", zInv, zInvVar))
	}
	return 0
}

func Fuzz4(data []byte) int {
	var gfpTest gfP12
	f := fuzz.NewFromGoFuzz(data)
	f.Fuzz(&gfpTest)
	zInvVar := &gfP12{}
	zInvVar.InvertVariableTime(&gfpTest)
	zInv := &gfP12{}
	zInv.Invert(&gfpTest)
	if !gfp12Eq(zInv, zInvVar) {
		panic(fmt.Sprintf("invalid invert: %v %v", zInv, zInvVar))
	}
	return 0
}

func gfp12Eq(a, b *gfP12) bool {
	if gfp6Eq(&a.x, &b.x) &&
		gfp6Eq(&a.y, &b.y) {
		return true
	}
	return false
}

func gfp6Eq(a, b *gfP6) bool {
	if gfp2Eq(&a.x, &b.x) &&
		gfp2Eq(&a.y, &b.y) &&
		gfp2Eq(&a.z, &b.z) {
		return true
	}
	return false
}

func gfp2Eq(a, b *gfP2) bool {
	if gfpEq(&a.x, &b.x) &&
		gfpEq(&a.y, &b.y) {
		return true
	}
	return false
}

func gfpEq(a, b *gfP) bool {
	if a[0] == b[0] &&
		a[1] == b[1] &&
		a[2] == b[2] &&
		a[3] == b[3] {
		return true
	}
	return false
}
