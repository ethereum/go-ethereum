# Test libsecp256k1' group operation implementations using prover.sage

import sys

load("group_prover.sage")
load("weierstrass_prover.sage")

def formula_secp256k1_gej_double_var(a):
  """libsecp256k1's secp256k1_gej_double_var, used by various addition functions"""
  rz = a.Z * a.Y
  rz = rz * 2
  t1 = a.X^2
  t1 = t1 * 3
  t2 = t1^2
  t3 = a.Y^2
  t3 = t3 * 2
  t4 = t3^2
  t4 = t4 * 2
  t3 = t3 * a.X
  rx = t3
  rx = rx * 4
  rx = -rx
  rx = rx + t2
  t2 = -t2
  t3 = t3 * 6
  t3 = t3 + t2
  ry = t1 * t3
  t2 = -t4
  ry = ry + t2
  return jacobianpoint(rx, ry, rz)

def formula_secp256k1_gej_add_var(branch, a, b):
  """libsecp256k1's secp256k1_gej_add_var"""
  if branch == 0:
    return (constraints(), constraints(nonzero={a.Infinity : 'a_infinite'}), b)
  if branch == 1:
    return (constraints(), constraints(zero={a.Infinity : 'a_finite'}, nonzero={b.Infinity : 'b_infinite'}), a)
  z22 = b.Z^2
  z12 = a.Z^2
  u1 = a.X * z22
  u2 = b.X * z12
  s1 = a.Y * z22
  s1 = s1 * b.Z
  s2 = b.Y * z12
  s2 = s2 * a.Z
  h = -u1
  h = h + u2
  i = -s1
  i = i + s2
  if branch == 2:
    r = formula_secp256k1_gej_double_var(a)
    return (constraints(), constraints(zero={h : 'h=0', i : 'i=0', a.Infinity : 'a_finite', b.Infinity : 'b_finite'}), r)
  if branch == 3:
    return (constraints(), constraints(zero={h : 'h=0', a.Infinity : 'a_finite', b.Infinity : 'b_finite'}, nonzero={i : 'i!=0'}), point_at_infinity())
  i2 = i^2
  h2 = h^2
  h3 = h2 * h
  h = h * b.Z
  rz = a.Z * h
  t = u1 * h2
  rx = t
  rx = rx * 2
  rx = rx + h3
  rx = -rx
  rx = rx + i2
  ry = -rx
  ry = ry + t
  ry = ry * i
  h3 = h3 * s1
  h3 = -h3
  ry = ry + h3
  return (constraints(), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite'}, nonzero={h : 'h!=0'}), jacobianpoint(rx, ry, rz))

def formula_secp256k1_gej_add_ge_var(branch, a, b):
  """libsecp256k1's secp256k1_gej_add_ge_var, which assume bz==1"""
  if branch == 0:
    return (constraints(zero={b.Z - 1 : 'b.z=1'}), constraints(nonzero={a.Infinity : 'a_infinite'}), b)
  if branch == 1:
    return (constraints(zero={b.Z - 1 : 'b.z=1'}), constraints(zero={a.Infinity : 'a_finite'}, nonzero={b.Infinity : 'b_infinite'}), a)
  z12 = a.Z^2
  u1 = a.X
  u2 = b.X * z12
  s1 = a.Y
  s2 = b.Y * z12
  s2 = s2 * a.Z
  h = -u1
  h = h + u2
  i = -s1
  i = i + s2
  if (branch == 2):
    r = formula_secp256k1_gej_double_var(a)
    return (constraints(zero={b.Z - 1 : 'b.z=1'}), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite', h : 'h=0', i : 'i=0'}), r)
  if (branch == 3):
    return (constraints(zero={b.Z - 1 : 'b.z=1'}), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite', h : 'h=0'}, nonzero={i : 'i!=0'}), point_at_infinity())
  i2 = i^2
  h2 = h^2
  h3 = h * h2
  rz = a.Z * h
  t = u1 * h2
  rx = t
  rx = rx * 2
  rx = rx + h3
  rx = -rx
  rx = rx + i2
  ry = -rx
  ry = ry + t
  ry = ry * i
  h3 = h3 * s1
  h3 = -h3
  ry = ry + h3
  return (constraints(zero={b.Z - 1 : 'b.z=1'}), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite'}, nonzero={h : 'h!=0'}), jacobianpoint(rx, ry, rz))

def formula_secp256k1_gej_add_zinv_var(branch, a, b):
  """libsecp256k1's secp256k1_gej_add_zinv_var"""
  bzinv = b.Z^(-1)
  if branch == 0:
    return (constraints(), constraints(nonzero={b.Infinity : 'b_infinite'}), a)
  if branch == 1:
    bzinv2 = bzinv^2
    bzinv3 = bzinv2 * bzinv
    rx = b.X * bzinv2
    ry = b.Y * bzinv3
    rz = 1
    return (constraints(), constraints(zero={b.Infinity : 'b_finite'}, nonzero={a.Infinity : 'a_infinite'}), jacobianpoint(rx, ry, rz))
  azz = a.Z * bzinv
  z12 = azz^2
  u1 = a.X
  u2 = b.X * z12
  s1 = a.Y
  s2 = b.Y * z12
  s2 = s2 * azz
  h = -u1
  h = h + u2
  i = -s1
  i = i + s2
  if branch == 2:
    r = formula_secp256k1_gej_double_var(a)
    return (constraints(), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite', h : 'h=0', i : 'i=0'}), r)
  if branch == 3:
    return (constraints(), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite', h : 'h=0'}, nonzero={i : 'i!=0'}), point_at_infinity())
  i2 = i^2
  h2 = h^2
  h3 = h * h2
  rz = a.Z
  rz = rz * h
  t = u1 * h2
  rx = t
  rx = rx * 2
  rx = rx + h3
  rx = -rx
  rx = rx + i2
  ry = -rx
  ry = ry + t
  ry = ry * i
  h3 = h3 * s1
  h3 = -h3
  ry = ry + h3
  return (constraints(), constraints(zero={a.Infinity : 'a_finite', b.Infinity : 'b_finite'}, nonzero={h : 'h!=0'}), jacobianpoint(rx, ry, rz))

def formula_secp256k1_gej_add_ge(branch, a, b):
  """libsecp256k1's secp256k1_gej_add_ge"""
  zeroes = {}
  nonzeroes = {}
  a_infinity = False
  if (branch & 4) != 0:
    nonzeroes.update({a.Infinity : 'a_infinite'})
    a_infinity = True
  else:
    zeroes.update({a.Infinity : 'a_finite'})
  zz = a.Z^2
  u1 = a.X
  u2 = b.X * zz
  s1 = a.Y
  s2 = b.Y * zz
  s2 = s2 * a.Z
  t = u1
  t = t + u2
  m = s1
  m = m + s2
  rr = t^2
  m_alt = -u2
  tt = u1 * m_alt
  rr = rr + tt
  degenerate = (branch & 3) == 3
  if (branch & 1) != 0:
    zeroes.update({m : 'm_zero'})
  else:
    nonzeroes.update({m : 'm_nonzero'})
  if (branch & 2) != 0:
    zeroes.update({rr : 'rr_zero'})
  else:
    nonzeroes.update({rr : 'rr_nonzero'})
  rr_alt = s1
  rr_alt = rr_alt * 2
  m_alt = m_alt + u1
  if not degenerate:
    rr_alt = rr
    m_alt = m
  n = m_alt^2
  q = n * t
  n = n^2
  if degenerate:
    n = m
  t = rr_alt^2
  rz = a.Z * m_alt
  infinity = False
  if (branch & 8) != 0:
    if not a_infinity:
      infinity = True
    zeroes.update({rz : 'r.z=0'})
  else:
    nonzeroes.update({rz : 'r.z!=0'})
  rz = rz * 2
  q = -q
  t = t + q
  rx = t
  t = t * 2
  t = t + q
  t = t * rr_alt
  t = t + n
  ry = -t
  rx = rx * 4
  ry = ry * 4
  if a_infinity:
    rx = b.X
    ry = b.Y
    rz = 1
  if infinity:
    return (constraints(zero={b.Z - 1 : 'b.z=1', b.Infinity : 'b_finite'}), constraints(zero=zeroes, nonzero=nonzeroes), point_at_infinity())
  return (constraints(zero={b.Z - 1 : 'b.z=1', b.Infinity : 'b_finite'}), constraints(zero=zeroes, nonzero=nonzeroes), jacobianpoint(rx, ry, rz))

def formula_secp256k1_gej_add_ge_old(branch, a, b):
  """libsecp256k1's old secp256k1_gej_add_ge, which fails when ay+by=0 but ax!=bx"""
  a_infinity = (branch & 1) != 0
  zero = {}
  nonzero = {}
  if a_infinity:
    nonzero.update({a.Infinity : 'a_infinite'})
  else:
    zero.update({a.Infinity : 'a_finite'})
  zz = a.Z^2
  u1 = a.X
  u2 = b.X * zz
  s1 = a.Y
  s2 = b.Y * zz
  s2 = s2 * a.Z
  z = a.Z
  t = u1
  t = t + u2
  m = s1
  m = m + s2
  n = m^2
  q = n * t
  n = n^2
  rr = t^2
  t = u1 * u2
  t = -t
  rr = rr + t
  t = rr^2
  rz = m * z
  infinity = False
  if (branch & 2) != 0:
    if not a_infinity:
      infinity = True
    else:
      return (constraints(zero={b.Z - 1 : 'b.z=1', b.Infinity : 'b_finite'}), constraints(nonzero={z : 'conflict_a'}, zero={z : 'conflict_b'}), point_at_infinity())
    zero.update({rz : 'r.z=0'})
  else:
    nonzero.update({rz : 'r.z!=0'})
  rz = rz * (0 if a_infinity else 2)
  rx = t
  q = -q
  rx = rx + q
  q = q * 3
  t = t * 2
  t = t + q
  t = t * rr
  t = t + n
  ry = -t
  rx = rx * (0 if a_infinity else 4)
  ry = ry * (0 if a_infinity else 4)
  t = b.X
  t = t * (1 if a_infinity else 0)
  rx = rx + t
  t = b.Y
  t = t * (1 if a_infinity else 0)
  ry = ry + t
  t = (1 if a_infinity else 0)
  rz = rz + t
  if infinity:
    return (constraints(zero={b.Z - 1 : 'b.z=1', b.Infinity : 'b_finite'}), constraints(zero=zero, nonzero=nonzero), point_at_infinity())
  return (constraints(zero={b.Z - 1 : 'b.z=1', b.Infinity : 'b_finite'}), constraints(zero=zero, nonzero=nonzero), jacobianpoint(rx, ry, rz))

if __name__ == "__main__":
  check_symbolic_jacobian_weierstrass("secp256k1_gej_add_var", 0, 7, 5, formula_secp256k1_gej_add_var)
  check_symbolic_jacobian_weierstrass("secp256k1_gej_add_ge_var", 0, 7, 5, formula_secp256k1_gej_add_ge_var)
  check_symbolic_jacobian_weierstrass("secp256k1_gej_add_zinv_var", 0, 7, 5, formula_secp256k1_gej_add_zinv_var)
  check_symbolic_jacobian_weierstrass("secp256k1_gej_add_ge", 0, 7, 16, formula_secp256k1_gej_add_ge)
  check_symbolic_jacobian_weierstrass("secp256k1_gej_add_ge_old [should fail]", 0, 7, 4, formula_secp256k1_gej_add_ge_old)

  if len(sys.argv) >= 2 and sys.argv[1] == "--exhaustive":
    check_exhaustive_jacobian_weierstrass("secp256k1_gej_add_var", 0, 7, 5, formula_secp256k1_gej_add_var, 43)
    check_exhaustive_jacobian_weierstrass("secp256k1_gej_add_ge_var", 0, 7, 5, formula_secp256k1_gej_add_ge_var, 43)
    check_exhaustive_jacobian_weierstrass("secp256k1_gej_add_zinv_var", 0, 7, 5, formula_secp256k1_gej_add_zinv_var, 43)
    check_exhaustive_jacobian_weierstrass("secp256k1_gej_add_ge", 0, 7, 16, formula_secp256k1_gej_add_ge, 43)
    check_exhaustive_jacobian_weierstrass("secp256k1_gej_add_ge_old [should fail]", 0, 7, 4, formula_secp256k1_gej_add_ge_old, 43)
