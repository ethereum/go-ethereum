// +build amd64,!generic

package bls12381

import (
	"golang.org/x/sys/cpu"
)

func init() {
	if !cpu.X86.HasADX || !cpu.X86.HasBMI2 {
		mul = mulNoADX
		wmul = wmulNoADX
		fromWide = montRedNoADX
		mulFR = mulNoADXFR
		wmulFR = wmulNoADXFR
		wfp2Mul = wfp2MulGeneric
		wfp2Square = wfp2SquareGeneric
	}
}

var mul func(c, a, b *fe) = mulADX
var wmul func(c *wfe, a, b *fe) = wmulADX
var fromWide func(c *fe, w *wfe) = montRedADX
var wfp2Mul func(c *wfe2, a, b *fe2) = wfp2MulADX
var wfp2Square func(c *wfe2, b *fe2) = wfp2SquareADX

func square(c, a *fe) {
	mul(c, a, a)
}

func neg(c, a *fe) {
	if a.isZero() {
		c.set(a)
	} else {
		_neg(c, a)
	}
}

//go:noescape
func add(c, a, b *fe)

//go:noescape
func addAssign(a, b *fe)

//go:noescape
func ladd(c, a, b *fe)

//go:noescape
func laddAssign(a, b *fe)

//go:noescape
func double(c, a *fe)

//go:noescape
func doubleAssign(a *fe)

//go:noescape
func ldouble(c, a *fe)

//go:noescape
func ldoubleAssign(a *fe)

//go:noescape
func sub(c, a, b *fe)

//go:noescape
func subAssign(a, b *fe)

//go:noescape
func lsubAssign(a, b *fe)

//go:noescape
func _neg(c, a *fe)

//go:noescape
func mulNoADX(c, a, b *fe)

//go:noescape
func mulADX(c, a, b *fe)

//go:noescape
func wmulNoADX(c *wfe, a, b *fe)

//go:noescape
func wmulADX(c *wfe, a, b *fe)

//go:noescape
func montRedNoADX(a *fe, w *wfe)

//go:noescape
func montRedADX(a *fe, w *wfe)

//go:noescape
func lwadd(c, a, b *wfe)

//go:noescape
func lwaddAssign(a, b *wfe)

//go:noescape
func wadd(c, a, b *wfe)

//go:noescape
func lwdouble(c, a *wfe)

//go:noescape
func wdouble(c, a *wfe)

//go:noescape
func lwsub(c, a, b *wfe)

//go:noescape
func lwsubAssign(a, b *wfe)

//go:noescape
func wsub(c, a, b *wfe)

//go:noescape
func fp2Add(c, a, b *fe2)

//go:noescape
func fp2AddAssign(a, b *fe2)

//go:noescape
func fp2Ladd(c, a, b *fe2)

//go:noescape
func fp2LaddAssign(a, b *fe2)

//go:noescape
func fp2DoubleAssign(a *fe2)

//go:noescape
func fp2Double(c, a *fe2)

//go:noescape
func fp2Sub(c, a, b *fe2)

//go:noescape
func fp2SubAssign(a, b *fe2)

//go:noescape
func mulByNonResidue(c, a *fe2)

//go:noescape
func mulByNonResidueAssign(a *fe2)

//go:noescape
func wfp2Add(c, a, b *wfe2)

//go:noescape
func wfp2AddAssign(a, b *wfe2)

//go:noescape
func wfp2Ladd(c, a, b *wfe2)

//go:noescape
func wfp2LaddAssign(a, b *wfe2)

//go:noescape
func wfp2AddMixed(c, a, b *wfe2)

//go:noescape
func wfp2AddMixedAssign(a, b *wfe2)

//go:noescape
func wfp2Sub(c, a, b *wfe2)

//go:noescape
func wfp2SubAssign(a, b *wfe2)

//go:noescape
func wfp2SubMixed(c, a, b *wfe2)

//go:noescape
func wfp2SubMixedAssign(a, b *wfe2)

//go:noescape
func wfp2Double(c, a *wfe2)

//go:noescape
func wfp2DoubleAssign(a *wfe2)

//go:noescape
func wfp2MulByNonResidue(c, a *wfe2)

//go:noescape
func wfp2MulByNonResidueAssign(a *wfe2)

//go:noescape
func wfp2SquareADX(c *wfe2, a *fe2)

//go:noescape
func wfp2MulADX(c *wfe2, a, b *fe2)

var mulFR func(c, a, b *Fr) = mulADXFR
var wmulFR func(c *wideFr, a, b *Fr) = wmulADXFR

func squareFR(c, a *Fr) {
	mulFR(c, a, a)
}

func negFR(c, a *Fr) {
	if a.IsZero() {
		c.Set(a)
	} else {
		_negFR(c, a)
	}
}

//go:noescape
func addFR(c, a, b *Fr)

//go:noescape
func laddAssignFR(a, b *Fr)

//go:noescape
func doubleFR(c, a *Fr)

//go:noescape
func subFR(c, a, b *Fr)

//go:noescape
func lsubAssignFR(a, b *Fr)

//go:noescape
func _negFR(c, a *Fr)

//go:noescape
func mulNoADXFR(c, a, b *Fr)

//go:noescape
func mulADXFR(c, a, b *Fr)

//go:noescape
func wmulADXFR(c *wideFr, a, b *Fr)

//go:noescape
func wmulNoADXFR(c *wideFr, a, b *Fr)

//go:noescape
func waddFR(a, b *wideFr)
