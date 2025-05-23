export declare const secp256k1: Readonly<{
    create: (hash: import("@noble/curves/abstract/utils").CHash) => import("@noble/curves/abstract/weierstrass").CurveFn;
    CURVE: Readonly<{
        readonly nBitLength: number;
        readonly nByteLength: number;
        readonly Fp: import("@noble/curves/abstract/modular").IField<bigint>;
        readonly n: bigint;
        readonly h: bigint;
        readonly hEff?: bigint | undefined;
        readonly Gx: bigint;
        readonly Gy: bigint;
        readonly allowInfinityPoint?: boolean | undefined;
        readonly a: bigint;
        readonly b: bigint;
        readonly allowedPrivateKeyLengths?: readonly number[] | undefined;
        readonly wrapPrivateKey?: boolean | undefined;
        readonly endo?: {
            beta: bigint;
            splitScalar: (k: bigint) => {
                k1neg: boolean;
                k1: bigint;
                k2neg: boolean;
                k2: bigint;
            };
        } | undefined;
        readonly isTorsionFree?: ((c: import("@noble/curves/abstract/weierstrass").ProjConstructor<bigint>, point: import("@noble/curves/abstract/weierstrass").ProjPointType<bigint>) => boolean) | undefined;
        readonly clearCofactor?: ((c: import("@noble/curves/abstract/weierstrass").ProjConstructor<bigint>, point: import("@noble/curves/abstract/weierstrass").ProjPointType<bigint>) => import("@noble/curves/abstract/weierstrass").ProjPointType<bigint>) | undefined;
        readonly hash: import("@noble/curves/abstract/utils").CHash;
        readonly hmac: (key: Uint8Array, ...messages: Uint8Array[]) => Uint8Array;
        readonly randomBytes: (bytesLength?: number | undefined) => Uint8Array;
        lowS: boolean;
        readonly bits2int?: ((bytes: Uint8Array) => bigint) | undefined;
        readonly bits2int_modN?: ((bytes: Uint8Array) => bigint) | undefined;
        readonly p: bigint;
    }>;
    getPublicKey: (privateKey: import("@noble/curves/abstract/utils").PrivKey, isCompressed?: boolean | undefined) => Uint8Array;
    getSharedSecret: (privateA: import("@noble/curves/abstract/utils").PrivKey, publicB: import("@noble/curves/abstract/utils").Hex, isCompressed?: boolean | undefined) => Uint8Array;
    sign: (msgHash: import("@noble/curves/abstract/utils").Hex, privKey: import("@noble/curves/abstract/utils").PrivKey, opts?: import("@noble/curves/abstract/weierstrass").SignOpts | undefined) => import("@noble/curves/abstract/weierstrass").SignatureType;
    verify: (signature: import("@noble/curves/abstract/utils").Hex | {
        r: bigint;
        s: bigint;
    }, msgHash: import("@noble/curves/abstract/utils").Hex, publicKey: import("@noble/curves/abstract/utils").Hex, opts?: import("@noble/curves/abstract/weierstrass").VerOpts | undefined) => boolean;
    ProjectivePoint: import("@noble/curves/abstract/weierstrass").ProjConstructor<bigint>;
    Signature: import("@noble/curves/abstract/weierstrass").SignatureConstructor;
    utils: {
        normPrivateKeyToScalar: (key: import("@noble/curves/abstract/utils").PrivKey) => bigint;
        isValidPrivateKey(privateKey: import("@noble/curves/abstract/utils").PrivKey): boolean;
        randomPrivateKey: () => Uint8Array;
        precompute: (windowSize?: number | undefined, point?: import("@noble/curves/abstract/weierstrass").ProjPointType<bigint> | undefined) => import("@noble/curves/abstract/weierstrass").ProjPointType<bigint>;
    };
}>;
/**
 * 2^64-1
 */
export declare const MAX_UINT64: bigint;
/**
 * The max integer that the evm can handle (2^256-1)
 */
export declare const MAX_INTEGER: bigint;
export declare const SECP256K1_ORDER: bigint;
export declare const SECP256K1_ORDER_DIV_2: bigint;
//# sourceMappingURL=constants.d.ts.map