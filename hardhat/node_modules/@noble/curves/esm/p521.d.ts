import { type CurveFnWithCreate } from './_shortw_utils.ts';
import { type HTFMethod } from './abstract/hash-to-curve.ts';
/**
 * NIST secp521r1 aka p521 curve, ECDSA and ECDH methods.
 * Field: `2n**521n - 1n`.
 */
export declare const p521: CurveFnWithCreate;
export declare const secp521r1: CurveFnWithCreate;
/** secp521r1 hash-to-curve from RFC 9380. */
export declare const hashToCurve: HTFMethod<bigint>;
/** secp521r1 encode-to-curve from RFC 9380. */
export declare const encodeToCurve: HTFMethod<bigint>;
//# sourceMappingURL=p521.d.ts.map