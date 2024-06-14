const P1600_RHO_OFFSETS = [0, 1, 62, 28, 27, 36, 44, 6, 55, 20, 3, 10, 43, 25, 39, 41, 45, 15, 21, 8, 18, 2, 61, 56, 14]
const P1600_ROUND_CONSTANTS = [
  0x00000001, 0x00000000,
  0x00008082, 0x00000000,
  0x0000808a, 0x80000000,
  0x80008000, 0x80000000,
  0x0000808b, 0x00000000,
  0x80000001, 0x00000000,
  0x80008081, 0x80000000,
  0x00008009, 0x80000000,
  0x0000008a, 0x00000000,
  0x00000088, 0x00000000,
  0x80008009, 0x00000000,
  0x8000000a, 0x00000000,
  0x8000808b, 0x00000000,
  0x0000008b, 0x80000000,
  0x00008089, 0x80000000,
  0x00008003, 0x80000000,
  0x00008002, 0x80000000,
  0x00000080, 0x80000000,
  0x0000800a, 0x00000000,
  0x8000000a, 0x80000000,
  0x80008081, 0x80000000,
  0x00008080, 0x80000000,
  0x80000001, 0x00000000,
  0x80008008, 0x80000000
]

function p1600 (state) {
  for (let round = 0; round < 24; ++round) {
    theta(state)
    rho(state)
    pi(state)
    chi(state)
    iota(state, round)
  }
}

// steps
function theta (s) {
  const clo = [0, 0, 0, 0, 0]
  const chi = [0, 0, 0, 0, 0]

  for (let x = 0; x < 5; ++x) {
    for (let y = 0; y < 5; ++y) {
      clo[x] ^= s[ilo(x, y)]
      chi[x] ^= s[ihi(x, y)]
    }
  }

  for (let x = 0; x < 5; ++x) {
    const next = (x + 1) % 5
    const prev = (x + 4) % 5
    const dlo = rol64lo(clo[next], chi[next], 1) ^ clo[prev]
    const dhi = rol64hi(clo[next], chi[next], 1) ^ chi[prev]

    for (let y = 0; y < 5; ++y) {
      s[ilo(x, y)] ^= dlo
      s[ihi(x, y)] ^= dhi
    }
  }
}

function rho (s) {
  for (let x = 0; x < 5; ++x) {
    for (let y = 0; y < 5; ++y) {
      const lo = rol64lo(s[ilo(x, y)], s[ihi(x, y)], P1600_RHO_OFFSETS[index(x, y)])
      const hi = rol64hi(s[ilo(x, y)], s[ihi(x, y)], P1600_RHO_OFFSETS[index(x, y)])
      s[ilo(x, y)] = lo
      s[ihi(x, y)] = hi
    }
  }
}

function pi (s) {
  const ts = s.slice()

  for (let x = 0; x < 5; ++x) {
    for (let y = 0; y < 5; ++y) {
      const nx = (0 * x + 1 * y) % 5
      const ny = (2 * x + 3 * y) % 5
      s[ilo(nx, ny)] = ts[ilo(x, y)]
      s[ihi(nx, ny)] = ts[ihi(x, y)]
    }
  }
}

function chi (s) {
  const clo = [0, 0, 0, 0, 0]
  const chi = [0, 0, 0, 0, 0]

  for (let y = 0; y < 5; ++y) {
    for (let x = 0; x < 5; ++x) {
      clo[x] = s[ilo(x, y)] ^ (~s[ilo((x + 1) % 5, y)] & s[ilo((x + 2) % 5, y)])
      chi[x] = s[ihi(x, y)] ^ (~s[ihi((x + 1) % 5, y)] & s[ihi((x + 2) % 5, y)])
    }

    for (let x = 0; x < 5; ++x) {
      s[ilo(x, y)] = clo[x]
      s[ihi(x, y)] = chi[x]
    }
  }
}

function iota (s, i) {
  s[ilo(0, 0)] ^= P1600_ROUND_CONSTANTS[i * 2]
  s[ihi(0, 0)] ^= P1600_ROUND_CONSTANTS[i * 2 + 1]
}

// shortcuts
function index (x, y) { return x + 5 * y }
function ilo (x, y) { return index(x, y) * 2 }
function ihi (x, y) { return index(x, y) * 2 + 1 }

function rol64lo (lo, hi, shift) {
  if (shift >= 32) {
    const t = lo
    lo = hi
    hi = t
    shift -= 32
  }

  return shift === 0 ? lo : (lo << shift) | (hi >>> (32 - shift))
}

function rol64hi (lo, hi, shift) {
  if (shift >= 32) {
    const t = lo
    lo = hi
    hi = t
    shift -= 32
  }

  return shift === 0 ? hi : (hi << shift) | (lo >>> (32 - shift))
}

module.exports = {
  p1600: p1600,

  // for tests
  _rol64lo: rol64lo,
  _rol64hi: rol64hi
}
