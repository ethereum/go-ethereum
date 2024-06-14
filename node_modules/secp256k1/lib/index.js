const errors = {
  IMPOSSIBLE_CASE: 'Impossible case. Please create issue.',
  TWEAK_ADD:
    'The tweak was out of range or the resulted private key is invalid',
  TWEAK_MUL: 'The tweak was out of range or equal to zero',
  CONTEXT_RANDOMIZE_UNKNOW: 'Unknow error on context randomization',
  SECKEY_INVALID: 'Private Key is invalid',
  PUBKEY_PARSE: 'Public Key could not be parsed',
  PUBKEY_SERIALIZE: 'Public Key serialization error',
  PUBKEY_COMBINE: 'The sum of the public keys is not valid',
  SIG_PARSE: 'Signature could not be parsed',
  SIGN: 'The nonce generation function failed, or the private key was invalid',
  RECOVER: 'Public key could not be recover',
  ECDH: 'Scalar was invalid (zero or overflow)'
}

function assert (cond, msg) {
  if (!cond) throw new Error(msg)
}

function isUint8Array (name, value, length) {
  assert(value instanceof Uint8Array, `Expected ${name} to be an Uint8Array`)

  if (length !== undefined) {
    if (Array.isArray(length)) {
      const numbers = length.join(', ')
      const msg = `Expected ${name} to be an Uint8Array with length [${numbers}]`
      assert(length.includes(value.length), msg)
    } else {
      const msg = `Expected ${name} to be an Uint8Array with length ${length}`
      assert(value.length === length, msg)
    }
  }
}

function isCompressed (value) {
  assert(toTypeString(value) === 'Boolean', 'Expected compressed to be a Boolean')
}

function getAssertedOutput (output = (len) => new Uint8Array(len), length) {
  if (typeof output === 'function') output = output(length)
  isUint8Array('output', output, length)
  return output
}

function toTypeString (value) {
  return Object.prototype.toString.call(value).slice(8, -1)
}

module.exports = (secp256k1) => {
  return {
    contextRandomize (seed) {
      assert(
        seed === null || seed instanceof Uint8Array,
        'Expected seed to be an Uint8Array or null'
      )
      if (seed !== null) isUint8Array('seed', seed, 32)

      switch (secp256k1.contextRandomize(seed)) {
        case 1:
          throw new Error(errors.CONTEXT_RANDOMIZE_UNKNOW)
      }
    },

    privateKeyVerify (seckey) {
      isUint8Array('private key', seckey, 32)

      return secp256k1.privateKeyVerify(seckey) === 0
    },

    privateKeyNegate (seckey) {
      isUint8Array('private key', seckey, 32)

      switch (secp256k1.privateKeyNegate(seckey)) {
        case 0:
          return seckey
        case 1:
          throw new Error(errors.IMPOSSIBLE_CASE)
      }
    },

    privateKeyTweakAdd (seckey, tweak) {
      isUint8Array('private key', seckey, 32)
      isUint8Array('tweak', tweak, 32)

      switch (secp256k1.privateKeyTweakAdd(seckey, tweak)) {
        case 0:
          return seckey
        case 1:
          throw new Error(errors.TWEAK_ADD)
      }
    },

    privateKeyTweakMul (seckey, tweak) {
      isUint8Array('private key', seckey, 32)
      isUint8Array('tweak', tweak, 32)

      switch (secp256k1.privateKeyTweakMul(seckey, tweak)) {
        case 0:
          return seckey
        case 1:
          throw new Error(errors.TWEAK_MUL)
      }
    },

    publicKeyVerify (pubkey) {
      isUint8Array('public key', pubkey, [33, 65])

      return secp256k1.publicKeyVerify(pubkey) === 0
    },

    publicKeyCreate (seckey, compressed = true, output) {
      isUint8Array('private key', seckey, 32)
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyCreate(output, seckey)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.SECKEY_INVALID)
        case 2:
          throw new Error(errors.PUBKEY_SERIALIZE)
      }
    },

    publicKeyConvert (pubkey, compressed = true, output) {
      isUint8Array('public key', pubkey, [33, 65])
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyConvert(output, pubkey)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.PUBKEY_SERIALIZE)
      }
    },

    publicKeyNegate (pubkey, compressed = true, output) {
      isUint8Array('public key', pubkey, [33, 65])
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyNegate(output, pubkey)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.IMPOSSIBLE_CASE)
        case 3:
          throw new Error(errors.PUBKEY_SERIALIZE)
      }
    },

    publicKeyCombine (pubkeys, compressed = true, output) {
      assert(Array.isArray(pubkeys), 'Expected public keys to be an Array')
      assert(pubkeys.length > 0, 'Expected public keys array will have more than zero items')
      for (const pubkey of pubkeys) {
        isUint8Array('public key', pubkey, [33, 65])
      }
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyCombine(output, pubkeys)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.PUBKEY_COMBINE)
        case 3:
          throw new Error(errors.PUBKEY_SERIALIZE)
      }
    },

    publicKeyTweakAdd (pubkey, tweak, compressed = true, output) {
      isUint8Array('public key', pubkey, [33, 65])
      isUint8Array('tweak', tweak, 32)
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyTweakAdd(output, pubkey, tweak)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.TWEAK_ADD)
      }
    },

    publicKeyTweakMul (pubkey, tweak, compressed = true, output) {
      isUint8Array('public key', pubkey, [33, 65])
      isUint8Array('tweak', tweak, 32)
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.publicKeyTweakMul(output, pubkey, tweak)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.TWEAK_MUL)
      }
    },

    signatureNormalize (sig) {
      isUint8Array('signature', sig, 64)

      switch (secp256k1.signatureNormalize(sig)) {
        case 0:
          return sig
        case 1:
          throw new Error(errors.SIG_PARSE)
      }
    },

    signatureExport (sig, output) {
      isUint8Array('signature', sig, 64)
      output = getAssertedOutput(output, 72)

      const obj = { output, outputlen: 72 }
      switch (secp256k1.signatureExport(obj, sig)) {
        case 0:
          return output.slice(0, obj.outputlen)
        case 1:
          throw new Error(errors.SIG_PARSE)
        case 2:
          throw new Error(errors.IMPOSSIBLE_CASE)
      }
    },

    signatureImport (sig, output) {
      isUint8Array('signature', sig)
      output = getAssertedOutput(output, 64)

      switch (secp256k1.signatureImport(output, sig)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.SIG_PARSE)
        case 2:
          throw new Error(errors.IMPOSSIBLE_CASE)
      }
    },

    ecdsaSign (msg32, seckey, options = {}, output) {
      isUint8Array('message', msg32, 32)
      isUint8Array('private key', seckey, 32)
      assert(toTypeString(options) === 'Object', 'Expected options to be an Object')
      if (options.data !== undefined) isUint8Array('options.data', options.data)
      if (options.noncefn !== undefined) assert(toTypeString(options.noncefn) === 'Function', 'Expected options.noncefn to be a Function')
      output = getAssertedOutput(output, 64)

      const obj = { signature: output, recid: null }
      switch (secp256k1.ecdsaSign(obj, msg32, seckey, options.data, options.noncefn)) {
        case 0:
          return obj
        case 1:
          throw new Error(errors.SIGN)
        case 2:
          throw new Error(errors.IMPOSSIBLE_CASE)
      }
    },

    ecdsaVerify (sig, msg32, pubkey) {
      isUint8Array('signature', sig, 64)
      isUint8Array('message', msg32, 32)
      isUint8Array('public key', pubkey, [33, 65])

      switch (secp256k1.ecdsaVerify(sig, msg32, pubkey)) {
        case 0:
          return true
        case 3:
          return false
        case 1:
          throw new Error(errors.SIG_PARSE)
        case 2:
          throw new Error(errors.PUBKEY_PARSE)
      }
    },

    ecdsaRecover (sig, recid, msg32, compressed = true, output) {
      isUint8Array('signature', sig, 64)
      assert(
        toTypeString(recid) === 'Number' &&
          recid >= 0 &&
          recid <= 3,
        'Expected recovery id to be a Number within interval [0, 3]'
      )
      isUint8Array('message', msg32, 32)
      isCompressed(compressed)
      output = getAssertedOutput(output, compressed ? 33 : 65)

      switch (secp256k1.ecdsaRecover(output, sig, recid, msg32)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.SIG_PARSE)
        case 2:
          throw new Error(errors.RECOVER)
        case 3:
          throw new Error(errors.IMPOSSIBLE_CASE)
      }
    },

    ecdh (pubkey, seckey, options = {}, output) {
      isUint8Array('public key', pubkey, [33, 65])
      isUint8Array('private key', seckey, 32)
      assert(toTypeString(options) === 'Object', 'Expected options to be an Object')
      if (options.data !== undefined) isUint8Array('options.data', options.data)
      if (options.hashfn !== undefined) {
        assert(toTypeString(options.hashfn) === 'Function', 'Expected options.hashfn to be a Function')
        if (options.xbuf !== undefined) isUint8Array('options.xbuf', options.xbuf, 32)
        if (options.ybuf !== undefined) isUint8Array('options.ybuf', options.ybuf, 32)
        isUint8Array('output', output)
      } else {
        output = getAssertedOutput(output, 32)
      }

      switch (secp256k1.ecdh(output, pubkey, seckey, options.data, options.hashfn, options.xbuf, options.ybuf)) {
        case 0:
          return output
        case 1:
          throw new Error(errors.PUBKEY_PARSE)
        case 2:
          throw new Error(errors.ECDH)
      }
    }
  }
}
