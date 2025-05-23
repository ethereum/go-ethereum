/*! scure-bip32 - MIT License (c) 2022 Patricio Palladino, Paul Miller (paulmillr.com) */
import { hmac } from '@noble/hashes/hmac';
import { ripemd160 } from '@noble/hashes/ripemd160';
import { sha256 } from '@noble/hashes/sha256';
import { sha512 } from '@noble/hashes/sha512';
import { bytes as assertBytes } from '@noble/hashes/_assert';
import { bytesToHex, concatBytes, createView, hexToBytes, utf8ToBytes } from '@noble/hashes/utils';
import * as secp from '@noble/secp256k1';
import { base58check as base58checker } from '@scure/base';

// Enable sync API for noble-secp256k1
secp.utils.hmacSha256Sync = (key, ...msgs) => hmac(sha256, key, secp.utils.concatBytes(...msgs));
const base58check = base58checker(sha256);

function bytesToNumber(bytes: Uint8Array): bigint {
  return BigInt(`0x${bytesToHex(bytes)}`);
}

function numberToBytes(num: bigint): Uint8Array {
  return hexToBytes(num.toString(16).padStart(64, '0'));
}

const MASTER_SECRET = utf8ToBytes('Bitcoin seed');
// Bitcoin hardcoded by default
const BITCOIN_VERSIONS: Versions = { private: 0x0488ade4, public: 0x0488b21e };
export const HARDENED_OFFSET: number = 0x80000000;

export interface Versions {
  private: number;
  public: number;
}

const hash160 = (data: Uint8Array) => ripemd160(sha256(data));
const fromU32 = (data: Uint8Array) => createView(data).getUint32(0, false);
const toU32 = (n: number) => {
  if (!Number.isSafeInteger(n) || n < 0 || n > 2 ** 32 - 1) {
    throw new Error(`Invalid number=${n}. Should be from 0 to 2 ** 32 - 1`);
  }
  const buf = new Uint8Array(4);
  createView(buf).setUint32(0, n, false);
  return buf;
};

interface HDKeyOpt {
  versions: Versions;
  depth?: number;
  index?: number;
  parentFingerprint?: number;
  chainCode: Uint8Array;
  publicKey?: Uint8Array;
  privateKey?: Uint8Array | bigint;
}

export class HDKey {
  get fingerprint(): number {
    if (!this.pubHash) {
      throw new Error('No publicKey set!');
    }
    return fromU32(this.pubHash);
  }
  get identifier(): Uint8Array | undefined {
    return this.pubHash;
  }
  get pubKeyHash(): Uint8Array | undefined {
    return this.pubHash;
  }
  get privateKey(): Uint8Array | null {
    return this.privKeyBytes || null;
  }
  get publicKey(): Uint8Array | null {
    return this.pubKey || null;
  }
  get privateExtendedKey(): string {
    const priv = this.privateKey;
    if (!priv) {
      throw new Error('No private key');
    }
    return base58check.encode(
      this.serialize(this.versions.private, concatBytes(new Uint8Array([0]), priv))
    );
  }
  get publicExtendedKey(): string {
    if (!this.pubKey) {
      throw new Error('No public key');
    }
    return base58check.encode(this.serialize(this.versions.public, this.pubKey));
  }

  public static fromMasterSeed(seed: Uint8Array, versions: Versions = BITCOIN_VERSIONS): HDKey {
    assertBytes(seed);
    if (8 * seed.length < 128 || 8 * seed.length > 512) {
      throw new Error(
        `HDKey: wrong seed length=${seed.length}. Should be between 128 and 512 bits; 256 bits is advised)`
      );
    }
    const I = hmac(sha512, MASTER_SECRET, seed);
    return new HDKey({
      versions,
      chainCode: I.slice(32),
      privateKey: I.slice(0, 32),
    });
  }

  public static fromExtendedKey(base58key: string, versions: Versions = BITCOIN_VERSIONS): HDKey {
    // => version(4) || depth(1) || fingerprint(4) || index(4) || chain(32) || key(33)
    const keyBuffer: Uint8Array = base58check.decode(base58key);
    const keyView = createView(keyBuffer);
    const version = keyView.getUint32(0, false);
    const opt = {
      versions,
      depth: keyBuffer[4],
      parentFingerprint: keyView.getUint32(5, false),
      index: keyView.getUint32(9, false),
      chainCode: keyBuffer.slice(13, 45),
    };
    const key = keyBuffer.slice(45);
    const isPriv = key[0] === 0;
    if (version !== versions[isPriv ? 'private' : 'public']) {
      throw new Error('Version mismatch');
    }
    if (isPriv) {
      return new HDKey({ ...opt, privateKey: key.slice(1) });
    } else {
      return new HDKey({ ...opt, publicKey: key });
    }
  }

  public static fromJSON(json: { xpriv: string }): HDKey {
    return HDKey.fromExtendedKey(json.xpriv);
  }
  public readonly versions: Versions;
  public readonly depth: number = 0;
  public readonly index: number = 0;
  public readonly chainCode: Uint8Array | null = null;
  public readonly parentFingerprint: number = 0;
  private privKey?: bigint;
  private privKeyBytes?: Uint8Array;
  private pubKey?: Uint8Array;
  private pubHash: Uint8Array | undefined;

  constructor(opt: HDKeyOpt) {
    if (!opt || typeof opt !== 'object') {
      throw new Error('HDKey.constructor must not be called directly');
    }
    this.versions = opt.versions || BITCOIN_VERSIONS;
    this.depth = opt.depth || 0;
    this.chainCode = opt.chainCode;
    this.index = opt.index || 0;
    this.parentFingerprint = opt.parentFingerprint || 0;
    if (!this.depth) {
      if (this.parentFingerprint || this.index) {
        throw new Error('HDKey: zero depth with non-zero index/parent fingerprint');
      }
    }
    if (opt.publicKey && opt.privateKey) {
      throw new Error('HDKey: publicKey and privateKey at same time.');
    }
    if (opt.privateKey) {
      if (!secp.utils.isValidPrivateKey(opt.privateKey)) {
        throw new Error('Invalid private key');
      }
      this.privKey =
        typeof opt.privateKey === 'bigint' ? opt.privateKey : bytesToNumber(opt.privateKey);
      this.privKeyBytes = numberToBytes(this.privKey);
      this.pubKey = secp.getPublicKey(opt.privateKey, true);
    } else if (opt.publicKey) {
      this.pubKey = secp.Point.fromHex(opt.publicKey).toRawBytes(true); // force compressed point
    } else {
      throw new Error('HDKey: no public or private key provided');
    }
    this.pubHash = hash160(this.pubKey);
  }

  public derive(path: string): HDKey {
    if (!/^[mM]'?/.test(path)) {
      throw new Error('Path must start with "m" or "M"');
    }
    if (/^[mM]'?$/.test(path)) {
      return this;
    }
    const parts = path.replace(/^[mM]'?\//, '').split('/');
    // tslint:disable-next-line
    let child: HDKey = this;
    for (const c of parts) {
      const m = /^(\d+)('?)$/.exec(c);
      if (!m || m.length !== 3) {
        throw new Error(`Invalid child index: ${c}`);
      }
      let idx = +m[1];
      if (!Number.isSafeInteger(idx) || idx >= HARDENED_OFFSET) {
        throw new Error('Invalid index');
      }
      // hardened key
      if (m[2] === "'") {
        idx += HARDENED_OFFSET;
      }
      child = child.deriveChild(idx);
    }
    return child;
  }

  public deriveChild(index: number): HDKey {
    if (!this.pubKey || !this.chainCode) {
      throw new Error('No publicKey or chainCode set');
    }
    let data = toU32(index);
    if (index >= HARDENED_OFFSET) {
      // Hardened
      const priv = this.privateKey;
      if (!priv) {
        throw new Error('Could not derive hardened child key');
      }
      // Hardened child: 0x00 || ser256(kpar) || ser32(index)
      data = concatBytes(new Uint8Array([0]), priv, data);
    } else {
      // Normal child: serP(point(kpar)) || ser32(index)
      data = concatBytes(this.pubKey, data);
    }
    const I = hmac(sha512, this.chainCode, data);
    const childTweak = bytesToNumber(I.slice(0, 32));
    const chainCode = I.slice(32);
    if (!secp.utils.isValidPrivateKey(childTweak)) {
      throw new Error('Tweak bigger than curve order');
    }
    const opt: HDKeyOpt = {
      versions: this.versions,
      chainCode,
      depth: this.depth + 1,
      parentFingerprint: this.fingerprint,
      index,
    };
    try {
      // Private parent key -> private child key
      if (this.privateKey) {
        const added = secp.utils.mod(this.privKey! + childTweak, secp.CURVE.n);
        if (!secp.utils.isValidPrivateKey(added)) {
          throw new Error('The tweak was out of range or the resulted private key is invalid');
        }
        opt.privateKey = added;
      } else {
        const added = secp.Point.fromHex(this.pubKey).add(secp.Point.fromPrivateKey(childTweak));
        // Cryptographically impossible: hmac-sha512 preimage would need to be found
        if (added.equals(secp.Point.ZERO)) {
          throw new Error('The tweak was equal to negative P, which made the result key invalid');
        }
        opt.publicKey = added.toRawBytes(true);
      }
      return new HDKey(opt);
    } catch (err) {
      return this.deriveChild(index + 1);
    }
  }

  public sign(hash: Uint8Array): Uint8Array {
    if (!this.privateKey) {
      throw new Error('No privateKey set!');
    }
    assertBytes(hash, 32);
    return secp.signSync(hash, this.privKey!, {
      canonical: true,
      der: false,
    });
  }

  public verify(hash: Uint8Array, signature: Uint8Array): boolean {
    assertBytes(hash, 32);
    assertBytes(signature, 64);
    if (!this.publicKey) {
      throw new Error('No publicKey set!');
    }
    let sig;
    try {
      sig = secp.Signature.fromCompact(signature);
    } catch (error) {
      return false;
    }
    return secp.verify(sig, hash, this.publicKey);
  }

  public wipePrivateData(): this {
    this.privKey = undefined;
    if (this.privKeyBytes) {
      this.privKeyBytes.fill(0);
      this.privKeyBytes = undefined;
    }
    return this;
  }
  public toJSON(): { xpriv: string; xpub: string } {
    return {
      xpriv: this.privateExtendedKey,
      xpub: this.publicExtendedKey,
    };
  }

  private serialize(version: number, key: Uint8Array) {
    if (!this.chainCode) {
      throw new Error('No chainCode set');
    }
    assertBytes(key, 33);
    // version(4) || depth(1) || fingerprint(4) || index(4) || chain(32) || key(33)
    return concatBytes(
      toU32(version),
      new Uint8Array([this.depth]),
      toU32(this.parentFingerprint),
      toU32(this.index),
      this.chainCode,
      key
    );
  }
}
