const EC = require('elliptic').ec

const ec = new EC('secp256k1')
const ecparams = ec.curve

// Hack, we can not use bn.js@5, while elliptic uses bn.js@4
// See https://github.com/indutny/elliptic/issues/191#issuecomment-569888758
const BN = ecparams.n.constructor

function loadCompressedPublicKey (first, xbuf) {
  let x = new BN(xbuf)

  // overflow
  if (x.cmp(ecparams.p) >= 0) return null
  x = x.toRed(ecparams.red)

  // compute corresponding Y
  let y = x.redSqr().redIMul(x).redIAdd(ecparams.b).redSqrt()
  if ((first === 0x03) !== y.isOdd()) y = y.redNeg()

  return ec.keyPair({ pub: { x: x, y: y } })
}

function loadUncompressedPublicKey (first, xbuf, ybuf) {
  let x = new BN(xbuf)
  let y = new BN(ybuf)

  // overflow
  if (x.cmp(ecparams.p) >= 0 || y.cmp(ecparams.p) >= 0) return null

  x = x.toRed(ecparams.red)
  y = y.toRed(ecparams.red)

  // is odd flag
  if ((first === 0x06 || first === 0x07) && y.isOdd() !== (first === 0x07)) return null

  // x*x*x + b = y*y
  const x3 = x.redSqr().redIMul(x)
  if (!y.redSqr().redISub(x3.redIAdd(ecparams.b)).isZero()) return null

  return ec.keyPair({ pub: { x: x, y: y } })
}

function loadPublicKey (pubkey) {
  // length should be validated in interface
  const first = pubkey[0]
  switch (first) {
    case 0x02:
    case 0x03:
      if (pubkey.length !== 33) return null
      return loadCompressedPublicKey(first, pubkey.subarray(1, 33))
    case 0x04:
    case 0x06:
    case 0x07:
      if (pubkey.length !== 65) return null
      return loadUncompressedPublicKey(first, pubkey.subarray(1, 33), pubkey.subarray(33, 65))
    default:
      return null
  }
}

function savePublicKey (output, point) {
  const pubkey = point.encode(null, output.length === 33)
  // Loop should be faster because we do not need create extra Uint8Array
  // output.set(new Uint8Array(pubkey))
  for (let i = 0; i < output.length; ++i) output[i] = pubkey[i]
}

module.exports = {
  contextRandomize () {
    return 0
  },

  privateKeyVerify (seckey) {
    const bn = new BN(seckey)
    return bn.cmp(ecparams.n) < 0 && !bn.isZero() ? 0 : 1
  },

  privateKeyNegate (seckey) {
    const bn = new BN(seckey)
    const negate = ecparams.n.sub(bn).umod(ecparams.n).toArrayLike(Uint8Array, 'be', 32)
    seckey.set(negate)
    return 0
  },

  privateKeyTweakAdd (seckey, tweak) {
    const bn = new BN(tweak)
    if (bn.cmp(ecparams.n) >= 0) return 1

    bn.iadd(new BN(seckey))
    if (bn.cmp(ecparams.n) >= 0) bn.isub(ecparams.n)
    if (bn.isZero()) return 1

    const tweaked = bn.toArrayLike(Uint8Array, 'be', 32)
    seckey.set(tweaked)

    return 0
  },

  privateKeyTweakMul (seckey, tweak) {
    let bn = new BN(tweak)
    if (bn.cmp(ecparams.n) >= 0 || bn.isZero()) return 1

    bn.imul(new BN(seckey))
    if (bn.cmp(ecparams.n) >= 0) bn = bn.umod(ecparams.n)

    const tweaked = bn.toArrayLike(Uint8Array, 'be', 32)
    seckey.set(tweaked)

    return 0
  },

  publicKeyVerify (pubkey) {
    const pair = loadPublicKey(pubkey)
    return pair === null ? 1 : 0
  },

  publicKeyCreate (output, seckey) {
    const bn = new BN(seckey)
    if (bn.cmp(ecparams.n) >= 0 || bn.isZero()) return 1

    const point = ec.keyFromPrivate(seckey).getPublic()
    savePublicKey(output, point)

    return 0
  },

  publicKeyConvert (output, pubkey) {
    const pair = loadPublicKey(pubkey)
    if (pair === null) return 1

    const point = pair.getPublic()
    savePublicKey(output, point)

    return 0
  },

  publicKeyNegate (output, pubkey) {
    const pair = loadPublicKey(pubkey)
    if (pair === null) return 1

    const point = pair.getPublic()
    point.y = point.y.redNeg()
    savePublicKey(output, point)

    return 0
  },

  publicKeyCombine (output, pubkeys) {
    const pairs = new Array(pubkeys.length)
    for (let i = 0; i < pubkeys.length; ++i) {
      pairs[i] = loadPublicKey(pubkeys[i])
      if (pairs[i] === null) return 1
    }

    let point = pairs[0].getPublic()
    for (let i = 1; i < pairs.length; ++i) point = point.add(pairs[i].pub)
    if (point.isInfinity()) return 2

    savePublicKey(output, point)

    return 0
  },

  publicKeyTweakAdd (output, pubkey, tweak) {
    const pair = loadPublicKey(pubkey)
    if (pair === null) return 1

    tweak = new BN(tweak)
    if (tweak.cmp(ecparams.n) >= 0) return 2

    const point = pair.getPublic().add(ecparams.g.mul(tweak))
    if (point.isInfinity()) return 2

    savePublicKey(output, point)

    return 0
  },

  publicKeyTweakMul (output, pubkey, tweak) {
    const pair = loadPublicKey(pubkey)
    if (pair === null) return 1

    tweak = new BN(tweak)
    if (tweak.cmp(ecparams.n) >= 0 || tweak.isZero()) return 2

    const point = pair.getPublic().mul(tweak)
    savePublicKey(output, point)

    return 0
  },

  signatureNormalize (sig) {
    const r = new BN(sig.subarray(0, 32))
    const s = new BN(sig.subarray(32, 64))
    if (r.cmp(ecparams.n) >= 0 || s.cmp(ecparams.n) >= 0) return 1

    if (s.cmp(ec.nh) === 1) {
      sig.set(ecparams.n.sub(s).toArrayLike(Uint8Array, 'be', 32), 32)
    }

    return 0
  },

  // Copied 1-to-1 from https://github.com/bitcoinjs/bip66/blob/master/index.js
  // Adapted for Uint8Array instead Buffer
  signatureExport (obj, sig) {
    const sigR = sig.subarray(0, 32)
    const sigS = sig.subarray(32, 64)
    if (new BN(sigR).cmp(ecparams.n) >= 0) return 1
    if (new BN(sigS).cmp(ecparams.n) >= 0) return 1

    const { output } = obj

    // Prepare R
    let r = output.subarray(4, 4 + 33)
    r[0] = 0x00
    r.set(sigR, 1)

    let lenR = 33
    let posR = 0
    for (; lenR > 1 && r[posR] === 0x00 && !(r[posR + 1] & 0x80); --lenR, ++posR);

    r = r.subarray(posR)
    if (r[0] & 0x80) return 1
    if (lenR > 1 && (r[0] === 0x00) && !(r[1] & 0x80)) return 1

    // Prepare S
    let s = output.subarray(6 + 33, 6 + 33 + 33)
    s[0] = 0x00
    s.set(sigS, 1)

    let lenS = 33
    let posS = 0
    for (; lenS > 1 && s[posS] === 0x00 && !(s[posS + 1] & 0x80); --lenS, ++posS);

    s = s.subarray(posS)
    if (s[0] & 0x80) return 1
    if (lenS > 1 && (s[0] === 0x00) && !(s[1] & 0x80)) return 1

    // Set output length for return
    obj.outputlen = 6 + lenR + lenS

    // Output in specified format
    // 0x30 [total-length] 0x02 [R-length] [R] 0x02 [S-length] [S]
    output[0] = 0x30
    output[1] = obj.outputlen - 2
    output[2] = 0x02
    output[3] = r.length
    output.set(r, 4)
    output[4 + lenR] = 0x02
    output[5 + lenR] = s.length
    output.set(s, 6 + lenR)

    return 0
  },

  // Copied 1-to-1 from https://github.com/bitcoinjs/bip66/blob/master/index.js
  // Adapted for Uint8Array instead Buffer
  signatureImport (output, sig) {
    if (sig.length < 8) return 1
    if (sig.length > 72) return 1
    if (sig[0] !== 0x30) return 1
    if (sig[1] !== sig.length - 2) return 1
    if (sig[2] !== 0x02) return 1

    const lenR = sig[3]
    if (lenR === 0) return 1
    if (5 + lenR >= sig.length) return 1
    if (sig[4 + lenR] !== 0x02) return 1

    const lenS = sig[5 + lenR]
    if (lenS === 0) return 1
    if ((6 + lenR + lenS) !== sig.length) return 1

    if (sig[4] & 0x80) return 1
    if (lenR > 1 && (sig[4] === 0x00) && !(sig[5] & 0x80)) return 1

    if (sig[lenR + 6] & 0x80) return 1
    if (lenS > 1 && (sig[lenR + 6] === 0x00) && !(sig[lenR + 7] & 0x80)) return 1

    let sigR = sig.subarray(4, 4 + lenR)
    if (sigR.length === 33 && sigR[0] === 0x00) sigR = sigR.subarray(1)
    if (sigR.length > 32) return 1

    let sigS = sig.subarray(6 + lenR)
    if (sigS.length === 33 && sigS[0] === 0x00) sigS = sigS.slice(1)
    if (sigS.length > 32) throw new Error('S length is too long')

    let r = new BN(sigR)
    if (r.cmp(ecparams.n) >= 0) r = new BN(0)

    let s = new BN(sig.subarray(6 + lenR))
    if (s.cmp(ecparams.n) >= 0) s = new BN(0)

    output.set(r.toArrayLike(Uint8Array, 'be', 32), 0)
    output.set(s.toArrayLike(Uint8Array, 'be', 32), 32)

    return 0
  },

  ecdsaSign (obj, message, seckey, data, noncefn) {
    if (noncefn) {
      const _noncefn = noncefn
      noncefn = (counter) => {
        const nonce = _noncefn(message, seckey, null, data, counter)

        const isValid = nonce instanceof Uint8Array && nonce.length === 32
        if (!isValid) throw new Error('This is the way')

        return new BN(nonce)
      }
    }

    const d = new BN(seckey)
    if (d.cmp(ecparams.n) >= 0 || d.isZero()) return 1

    let sig
    try {
      sig = ec.sign(message, seckey, { canonical: true, k: noncefn, pers: data })
    } catch (err) {
      return 1
    }

    obj.signature.set(sig.r.toArrayLike(Uint8Array, 'be', 32), 0)
    obj.signature.set(sig.s.toArrayLike(Uint8Array, 'be', 32), 32)
    obj.recid = sig.recoveryParam

    return 0
  },

  ecdsaVerify (sig, msg32, pubkey) {
    const sigObj = { r: sig.subarray(0, 32), s: sig.subarray(32, 64) }

    const sigr = new BN(sigObj.r)
    const sigs = new BN(sigObj.s)
    if (sigr.cmp(ecparams.n) >= 0 || sigs.cmp(ecparams.n) >= 0) return 1
    if (sigs.cmp(ec.nh) === 1 || sigr.isZero() || sigs.isZero()) return 3

    const pair = loadPublicKey(pubkey)
    if (pair === null) return 2

    const point = pair.getPublic()
    const isValid = ec.verify(msg32, sigObj, point)
    return isValid ? 0 : 3
  },

  ecdsaRecover (output, sig, recid, msg32) {
    const sigObj = { r: sig.slice(0, 32), s: sig.slice(32, 64) }

    const sigr = new BN(sigObj.r)
    const sigs = new BN(sigObj.s)
    if (sigr.cmp(ecparams.n) >= 0 || sigs.cmp(ecparams.n) >= 0) return 1

    if (sigr.isZero() || sigs.isZero()) return 2

    // Can throw `throw new Error('Unable to find sencond key candinate');`
    let point
    try {
      point = ec.recoverPubKey(msg32, sigObj, recid)
    } catch (err) {
      return 2
    }

    savePublicKey(output, point)

    return 0
  },

  ecdh (output, pubkey, seckey, data, hashfn, xbuf, ybuf) {
    const pair = loadPublicKey(pubkey)
    if (pair === null) return 1

    const scalar = new BN(seckey)
    if (scalar.cmp(ecparams.n) >= 0 || scalar.isZero()) return 2

    const point = pair.getPublic().mul(scalar)

    if (hashfn === undefined) {
      const data = point.encode(null, true)
      const sha256 = ec.hash().update(data).digest()
      for (let i = 0; i < 32; ++i) output[i] = sha256[i]
    } else {
      if (!xbuf) xbuf = new Uint8Array(32)
      const x = point.getX().toArray('be', 32)
      for (let i = 0; i < 32; ++i) xbuf[i] = x[i]

      if (!ybuf) ybuf = new Uint8Array(32)
      const y = point.getY().toArray('be', 32)
      for (let i = 0; i < 32; ++i) ybuf[i] = y[i]

      const hash = hashfn(xbuf, ybuf, data)

      const isValid = hash instanceof Uint8Array && hash.length === output.length
      if (!isValid) return 2

      output.set(hash)
    }

    return 0
  }
}
