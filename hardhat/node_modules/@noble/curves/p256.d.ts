import { type CurveFnWithCreate } from './_shortw_utils.ts';
import { type HTFMethod } from './abstract/hash-to-curve.ts';
/**
 * secp256r1 curve, ECDSA and ECDH methods.
 * Field: `2n**224n * (2n**32n-1n) + 2n**192n + 2n**96n-1n`
 */
export declare const p256: CurveFnWithCreate;
/** Alias to p256. */
export declare const secp256r1: CurveFnWithCreate;
/** secp256r1 hash-to-curve from RFC 9380. */
export declare const hashToCurve: HTFMethod<bigint>;
/** secp256r1 encode-to-curve from RFC 9380. */
export declare const encodeToCurve: HTFMethod<bigint>;
//# sourceMappingURL=p256.d.ts.map