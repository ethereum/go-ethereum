import { type CurveFnWithCreate } from './_shortw_utils.ts';
import { type HTFMethod } from './abstract/hash-to-curve.ts';
import { mod } from './abstract/modular.ts';
import type { Hex, PrivKey } from './abstract/utils.ts';
import { bytesToNumberBE, numberToBytesBE } from './abstract/utils.ts';
import { type ProjPointType as PointType } from './abstract/weierstrass.ts';
/**
 * secp256k1 curve, ECDSA and ECDH methods.
 *
 * Field: `2n**256n - 2n**32n - 2n**9n - 2n**8n - 2n**7n - 2n**6n - 2n**4n - 1n`
 *
 * @example
 * ```js
 * import { secp256k1 } from '@noble/curves/secp256k1';
 * const priv = secp256k1.utils.randomPrivateKey();
 * const pub = secp256k1.getPublicKey(priv);
 * const msg = new Uint8Array(32).fill(1); // message hash (not message) in ecdsa
 * const sig = secp256k1.sign(msg, priv); // `{prehash: true}` option is available
 * const isValid = secp256k1.verify(sig, msg, pub) === true;
 * ```
 */
export declare const secp256k1: CurveFnWithCreate;
declare function taggedHash(tag: string, ...messages: Uint8Array[]): Uint8Array;
/**
 * lift_x from BIP340. Convert 32-byte x coordinate to elliptic curve point.
 * @returns valid point checked for being on-curve
 */
declare function lift_x(x: bigint): PointType<bigint>;
/**
 * Schnorr public key is just `x` coordinate of Point as per BIP340.
 */
declare function schnorrGetPublicKey(privateKey: Hex): Uint8Array;
/**
 * Creates Schnorr signature as per BIP340. Verifies itself before returning anything.
 * auxRand is optional and is not the sole source of k generation: bad CSPRNG won't be dangerous.
 */
declare function schnorrSign(message: Hex, privateKey: PrivKey, auxRand?: Hex): Uint8Array;
/**
 * Verifies Schnorr signature.
 * Will swallow errors & return false except for initial type validation of arguments.
 */
declare function schnorrVerify(signature: Hex, message: Hex, publicKey: Hex): boolean;
export type SecpSchnorr = {
    getPublicKey: typeof schnorrGetPublicKey;
    sign: typeof schnorrSign;
    verify: typeof schnorrVerify;
    utils: {
        randomPrivateKey: () => Uint8Array;
        lift_x: typeof lift_x;
        pointToBytes: (point: PointType<bigint>) => Uint8Array;
        numberToBytesBE: typeof numberToBytesBE;
        bytesToNumberBE: typeof bytesToNumberBE;
        taggedHash: typeof taggedHash;
        mod: typeof mod;
    };
};
/**
 * Schnorr signatures over secp256k1.
 * https://github.com/bitcoin/bips/blob/master/bip-0340.mediawiki
 * @example
 * ```js
 * import { schnorr } from '@noble/curves/secp256k1';
 * const priv = schnorr.utils.randomPrivateKey();
 * const pub = schnorr.getPublicKey(priv);
 * const msg = new TextEncoder().encode('hello');
 * const sig = schnorr.sign(msg, priv);
 * const isValid = schnorr.verify(sig, msg, pub);
 * ```
 */
export declare const schnorr: SecpSchnorr;
/** secp256k1 hash-to-curve from RFC 9380. */
export declare const hashToCurve: HTFMethod<bigint>;
/** secp256k1 encode-to-curve from RFC 9380. */
export declare const encodeToCurve: HTFMethod<bigint>;
export {};
//# sourceMappingURL=secp256k1.d.ts.map