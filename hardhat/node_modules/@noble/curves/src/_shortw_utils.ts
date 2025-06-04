/**
 * Utilities for short weierstrass curves, combined with noble-hashes.
 * @module
 */
/*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) */
import { hmac } from '@noble/hashes/hmac';
import { concatBytes, randomBytes } from '@noble/hashes/utils';
import type { CHash } from './abstract/utils.ts';
import { type CurveFn, type CurveType, weierstrass } from './abstract/weierstrass.ts';

/** connects noble-curves to noble-hashes */
export function getHash(hash: CHash): {
  hash: CHash;
  hmac: (key: Uint8Array, ...msgs: Uint8Array[]) => Uint8Array;
  randomBytes: typeof randomBytes;
} {
  return {
    hash,
    hmac: (key: Uint8Array, ...msgs: Uint8Array[]) => hmac(hash, key, concatBytes(...msgs)),
    randomBytes,
  };
}
/** Same API as @noble/hashes, with ability to create curve with custom hash */
export type CurveDef = Readonly<Omit<CurveType, 'hash' | 'hmac' | 'randomBytes'>>;
export type CurveFnWithCreate = CurveFn & { create: (hash: CHash) => CurveFn };

export function createCurve(curveDef: CurveDef, defHash: CHash): CurveFnWithCreate {
  const create = (hash: CHash): CurveFn => weierstrass({ ...curveDef, ...getHash(hash) });
  return { ...create(defHash), create };
}
