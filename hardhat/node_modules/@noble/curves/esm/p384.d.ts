import { type CurveFnWithCreate } from './_shortw_utils.ts';
import { type HTFMethod } from './abstract/hash-to-curve.ts';
/**
 * secp384r1 curve, ECDSA and ECDH methods.
 * Field: `2n**384n - 2n**128n - 2n**96n + 2n**32n - 1n`.
 * */
export declare const p384: CurveFnWithCreate;
/** Alias to p384. */
export declare const secp384r1: CurveFnWithCreate;
/** secp384r1 hash-to-curve from RFC 9380. */
export declare const hashToCurve: HTFMethod<bigint>;
/** secp384r1 encode-to-curve from RFC 9380. */
export declare const encodeToCurve: HTFMethod<bigint>;
//# sourceMappingURL=p384.d.ts.map