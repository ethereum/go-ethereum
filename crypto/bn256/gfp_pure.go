//  +build !amd64 appengine gccgo

package bn256

func gfpNeg(c, a *gfP) {
	// c = p - a
	c[0] = p2[0] - a[0]
	c[1] = p2[1] - a[1]
	if p2[0] < a[0] {
		c[1]--
	}
	c[2] = p2[2] - a[2]
	if p2[1] < a[1] {
		c[2]--
	}
	c[3] = p2[3] - a[3]
	if p2[2] < a[2] {
		c[3]--
	}
	// c < p ? c : c-p
	gfpCarry(c, p2[3] < a[3])
}

func gfpAdd(c, a, b *gfP) {
	// c = a + b
	c[0] = a[0] + b[0]
	c[1] = a[1] + b[1]
	if c[0] < a[0] {
		c[1]++
	}
	c[2] = a[2] + b[2]
	if c[1] < a[1] {
		c[2]++
	}
	c[3] = a[3] + b[3]
	if c[2] < a[2] {
		c[3]++
	}
	// c < p ? c : c-p
	gfpCarry(c, c[3] < a[3])
}

func gfpSub(c, a, b *gfP) {
	// c = a - b
	c[0] = a[0] - b[0]
	c[1] = a[1] - b[1]
	if c[0] > a[0] {
		c[1]--
	}
	c[2] = a[2] - b[2]
	if c[1] > a[1] {
		c[2]--
	}
	c[3] = a[3] - b[3]
	if c[2] > a[2] {
		c[3]--
	}
	// c < p ? c : c-p
	gfpCarry(c, c[3] < a[3])
}

func gfpMul(c, a, b *gfP) {

}

func gfpCarry(c *gfP, carry bool) {
	if c[3] < p2[3] {
		return
	} else if c[3] == p2[3] {
		if c[2] < p2[3] {
			return
		} else if c[2] == p2[2] {
			if c[1] < p2[1] {
				return
			} else if c[1] == p2[1] {
				if c[0] < p2[0] {
					return
				}
			}
		}
	}
	if c[0] < p2[0] {
		c[1]--
	}
	c[0] -= p2[0]

	if c[1] < p2[1] {
		c[2]--
	}
	c[1] -= p2[1]

	if c[2] < p2[2] {
		c[3]--
	}
	c[2] -= p2[2]

	c[3] -= p2[3]
}
