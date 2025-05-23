'use strict';

function _interopDefault (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }

var assert = _interopDefault(require('assert'));
var safeBuffer = _interopDefault(require('safe-buffer'));
var bs58check = _interopDefault(require('bs58check'));

var cryptoShim = require("../shims/hdkey-crypto");

var secp256k1Shim = require("../shims/hdkey-secp256k1v3");

var Buffer = safeBuffer.Buffer;




var MASTER_SECRET = Buffer.from('Bitcoin seed', 'utf8');
var HARDENED_OFFSET = 0x80000000;
var LEN = 78;

// Bitcoin hardcoded by default, can use package `coininfo` for others
var BITCOIN_VERSIONS = {private: 0x0488ADE4, public: 0x0488B21E};

function HDKey (versions) {
  this.versions = versions || BITCOIN_VERSIONS;
  this.depth = 0;
  this.index = 0;
  this._privateKey = null;
  this._publicKey = null;
  this.chainCode = null;
  this._fingerprint = 0;
  this.parentFingerprint = 0;
}

Object.defineProperty(HDKey.prototype, 'fingerprint', { get: function () { return this._fingerprint } });
Object.defineProperty(HDKey.prototype, 'identifier', { get: function () { return this._identifier } });
Object.defineProperty(HDKey.prototype, 'pubKeyHash', { get: function () { return this.identifier } });

Object.defineProperty(HDKey.prototype, 'privateKey', {
  get: function () {
    return this._privateKey
  },
  set: function (value) {
    assert.equal(value.length, 32, 'Private key must be 32 bytes.');
    assert(secp256k1Shim.privateKeyVerify(value) === true, 'Invalid private key');

    this._privateKey = value;
    this._publicKey = secp256k1Shim.publicKeyCreate(value, true);
    this._identifier = hash160(this.publicKey);
    this._fingerprint = this._identifier.slice(0, 4).readUInt32BE(0);
  }
});

Object.defineProperty(HDKey.prototype, 'publicKey', {
  get: function () {
    return this._publicKey
  },
  set: function (value) {
    assert(value.length === 33 || value.length === 65, 'Public key must be 33 or 65 bytes.');
    assert(secp256k1Shim.publicKeyVerify(value) === true, 'Invalid public key');

    this._publicKey = secp256k1Shim.publicKeyConvert(value, true); // force compressed point
    this._identifier = hash160(this.publicKey);
    this._fingerprint = this._identifier.slice(0, 4).readUInt32BE(0);
    this._privateKey = null;
  }
});

Object.defineProperty(HDKey.prototype, 'privateExtendedKey', {
  get: function () {
    if (this._privateKey) return bs58check.encode(serialize(this, this.versions.private, Buffer.concat([Buffer.alloc(1, 0), this.privateKey])))
    else return null
  }
});

Object.defineProperty(HDKey.prototype, 'publicExtendedKey', {
  get: function () {
    return bs58check.encode(serialize(this, this.versions.public, this.publicKey))
  }
});

HDKey.prototype.derive = function (path) {
  if (path === 'm' || path === 'M' || path === "m'" || path === "M'") {
    return this
  }

  var entries = path.split('/');
  var hdkey = this;
  entries.forEach(function (c, i) {
    if (i === 0) {
      assert(/^[mM]{1}/.test(c), 'Path must start with "m" or "M"');
      return
    }

    var hardened = (c.length > 1) && (c[c.length - 1] === "'");
    var childIndex = parseInt(c, 10); // & (HARDENED_OFFSET - 1)
    assert(childIndex < HARDENED_OFFSET, 'Invalid index');
    if (hardened) childIndex += HARDENED_OFFSET;

    hdkey = hdkey.deriveChild(childIndex);
  });

  return hdkey
};

HDKey.prototype.deriveChild = function (index) {
  var isHardened = index >= HARDENED_OFFSET;
  var indexBuffer = Buffer.allocUnsafe(4);
  indexBuffer.writeUInt32BE(index, 0);

  var data;

  if (isHardened) { // Hardened child
    assert(this.privateKey, 'Could not derive hardened child key');

    var pk = this.privateKey;
    var zb = Buffer.alloc(1, 0);
    pk = Buffer.concat([zb, pk]);

    // data = 0x00 || ser256(kpar) || ser32(index)
    data = Buffer.concat([pk, indexBuffer]);
  } else { // Normal child
    // data = serP(point(kpar)) || ser32(index)
    //      = serP(Kpar) || ser32(index)
    data = Buffer.concat([this.publicKey, indexBuffer]);
  }

  var I = cryptoShim.createHmac('sha512', this.chainCode).update(data).digest();
  var IL = I.slice(0, 32);
  var IR = I.slice(32);

  var hd = new HDKey(this.versions);

  // Private parent key -> private child key
  if (this.privateKey) {
    // ki = parse256(IL) + kpar (mod n)
    try {
      hd.privateKey = secp256k1Shim.privateKeyTweakAdd(this.privateKey, IL);
      // throw if IL >= n || (privateKey + IL) === 0
    } catch (err) {
      // In case parse256(IL) >= n or ki == 0, one should proceed with the next value for i
      return this.deriveChild(index + 1)
    }
  // Public parent key -> public child key
  } else {
    // Ki = point(parse256(IL)) + Kpar
    //    = G*IL + Kpar
    try {
      hd.publicKey = secp256k1Shim.publicKeyTweakAdd(this.publicKey, IL, true);
      // throw if IL >= n || (g**IL + publicKey) is infinity
    } catch (err) {
      // In case parse256(IL) >= n or Ki is the point at infinity, one should proceed with the next value for i
      return this.deriveChild(index + 1)
    }
  }

  hd.chainCode = IR;
  hd.depth = this.depth + 1;
  hd.parentFingerprint = this.fingerprint;// .readUInt32BE(0)
  hd.index = index;

  return hd
};

HDKey.prototype.sign = function (hash) {
  return secp256k1Shim.sign(hash, this.privateKey).signature
};

HDKey.prototype.verify = function (hash, signature) {
  return secp256k1Shim.verify(hash, signature, this.publicKey)
};

HDKey.prototype.wipePrivateData = function () {
  if (this._privateKey) cryptoShim.randomBytes(this._privateKey.length).copy(this._privateKey);
  this._privateKey = null;
  return this
};

HDKey.prototype.toJSON = function () {
  return {
    xpriv: this.privateExtendedKey,
    xpub: this.publicExtendedKey
  }
};

HDKey.fromMasterSeed = function (seedBuffer, versions) {
  var I = cryptoShim.createHmac('sha512', MASTER_SECRET).update(seedBuffer).digest();
  var IL = I.slice(0, 32);
  var IR = I.slice(32);

  var hdkey = new HDKey(versions);
  hdkey.chainCode = IR;
  hdkey.privateKey = IL;

  return hdkey
};

HDKey.fromExtendedKey = function (base58key, versions) {
  // => version(4) || depth(1) || fingerprint(4) || index(4) || chain(32) || key(33)
  versions = versions || BITCOIN_VERSIONS;
  var hdkey = new HDKey(versions);

  var keyBuffer = bs58check.decode(base58key);

  var version = keyBuffer.readUInt32BE(0);
  assert(version === versions.private || version === versions.public, 'Version mismatch: does not match private or public');

  hdkey.depth = keyBuffer.readUInt8(4);
  hdkey.parentFingerprint = keyBuffer.readUInt32BE(5);
  hdkey.index = keyBuffer.readUInt32BE(9);
  hdkey.chainCode = keyBuffer.slice(13, 45);

  var key = keyBuffer.slice(45);
  if (key.readUInt8(0) === 0) { // private
    assert(version === versions.private, 'Version mismatch: version does not match private');
    hdkey.privateKey = key.slice(1); // cut off first 0x0 byte
  } else {
    assert(version === versions.public, 'Version mismatch: version does not match public');
    hdkey.publicKey = key;
  }

  return hdkey
};

HDKey.fromJSON = function (obj) {
  return HDKey.fromExtendedKey(obj.xpriv)
};

function serialize (hdkey, version, key) {
  // => version(4) || depth(1) || fingerprint(4) || index(4) || chain(32) || key(33)
  var buffer = Buffer.allocUnsafe(LEN);

  buffer.writeUInt32BE(version, 0);
  buffer.writeUInt8(hdkey.depth, 4);

  var fingerprint = hdkey.depth ? hdkey.parentFingerprint : 0x00000000;
  buffer.writeUInt32BE(fingerprint, 5);
  buffer.writeUInt32BE(hdkey.index, 9);

  hdkey.chainCode.copy(buffer, 13);
  key.copy(buffer, 45);

  return buffer
}

function hash160 (buf) {
  var sha = cryptoShim.createHash('sha256').update(buf).digest();
  return cryptoShim.createHash('ripemd160').update(sha).digest()
}

HDKey.HARDENED_OFFSET = HARDENED_OFFSET;
var hdkey = HDKey;

module.exports = hdkey;
