//  +build !amd64 appengine gccgo

package bn256

func gfpNeg(c, a *gfP) {
	panic("unsupported architecture")
}

func gfpAdd(c, a, b *gfP) {
	panic("unsupported architecture")
}

func gfpSub(c, a, b *gfP) {
	panic("unsupported architecture")
}

func gfpMul(c, a, b *gfP) {
	panic("unsupported architecture")
}
