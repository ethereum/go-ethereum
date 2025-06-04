import { type CurveFn, type ExtPointType } from './abstract/edwards.ts';
import { type CurveFn as WCurveFn } from './abstract/weierstrass.ts';
/** Curve over scalar field of bls12-381. jubjub Fp = bls n */
export declare const jubjub: CurveFn;
/** Curve over scalar field of bn254. babyjubjub Fp = bn254 n */
export declare const babyjubjub: CurveFn;
export declare function jubjub_groupHash(tag: Uint8Array, personalization: Uint8Array): ExtPointType;
export declare function jubjub_findGroupHash(m: Uint8Array, personalization: Uint8Array): ExtPointType;
export declare const pasta_p: bigint;
export declare const pasta_q: bigint;
/** https://neuromancer.sk/std/other/Pallas */
export declare const pallas: WCurveFn;
/** https://neuromancer.sk/std/other/Vesta */
export declare const vesta: WCurveFn;
//# sourceMappingURL=misc.d.ts.map