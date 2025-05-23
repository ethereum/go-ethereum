/*! noble-secp256k1 - MIT License (c) 2019 Paul Miller (paulmillr.com) */
declare const CURVE: Readonly<{
    a: bigint;
    b: bigint;
    P: bigint;
    n: bigint;
    h: bigint;
    Gx: bigint;
    Gy: bigint;
    beta: bigint;
}>;
export { CURVE };
declare type Hex = Uint8Array | string;
declare type PrivKey = Hex | bigint | number;
declare type PubKey = Hex | Point;
declare type Sig = Hex | Signature;
declare class JacobianPoint {
    readonly x: bigint;
    readonly y: bigint;
    readonly z: bigint;
    constructor(x: bigint, y: bigint, z: bigint);
    static readonly BASE: JacobianPoint;
    static readonly ZERO: JacobianPoint;
    static fromAffine(p: Point): JacobianPoint;
    static toAffineBatch(points: JacobianPoint[]): Point[];
    static normalizeZ(points: JacobianPoint[]): JacobianPoint[];
    equals(other: JacobianPoint): boolean;
    negate(): JacobianPoint;
    double(): JacobianPoint;
    add(other: JacobianPoint): JacobianPoint;
    subtract(other: JacobianPoint): JacobianPoint;
    multiplyUnsafe(scalar: bigint): JacobianPoint;
    private precomputeWindow;
    private wNAF;
    multiply(scalar: number | bigint, affinePoint?: Point): JacobianPoint;
    toAffine(invZ?: bigint): Point;
}
export declare class Point {
    readonly x: bigint;
    readonly y: bigint;
    static BASE: Point;
    static ZERO: Point;
    _WINDOW_SIZE?: number;
    constructor(x: bigint, y: bigint);
    _setWindowSize(windowSize: number): void;
    hasEvenY(): boolean;
    private static fromCompressedHex;
    private static fromUncompressedHex;
    static fromHex(hex: Hex): Point;
    static fromPrivateKey(privateKey: PrivKey): Point;
    static fromSignature(msgHash: Hex, signature: Sig, recovery: number): Point;
    toRawBytes(isCompressed?: boolean): Uint8Array;
    toHex(isCompressed?: boolean): string;
    toHexX(): string;
    toRawX(): Uint8Array;
    assertValidity(): void;
    equals(other: Point): boolean;
    negate(): Point;
    double(): Point;
    add(other: Point): Point;
    subtract(other: Point): Point;
    multiply(scalar: number | bigint): Point;
    multiplyAndAddUnsafe(Q: Point, a: bigint, b: bigint): Point | undefined;
}
export declare class Signature {
    readonly r: bigint;
    readonly s: bigint;
    constructor(r: bigint, s: bigint);
    static fromCompact(hex: Hex): Signature;
    static fromDER(hex: Hex): Signature;
    static fromHex(hex: Hex): Signature;
    assertValidity(): void;
    hasHighS(): boolean;
    normalizeS(): Signature;
    toDERRawBytes(): Uint8Array;
    toDERHex(): string;
    toRawBytes(): Uint8Array;
    toHex(): string;
    toCompactRawBytes(): Uint8Array;
    toCompactHex(): string;
}
declare function concatBytes(...arrays: Uint8Array[]): Uint8Array;
declare function bytesToHex(uint8a: Uint8Array): string;
declare function numTo32b(num: bigint): Uint8Array;
declare function hexToBytes(hex: string): Uint8Array;
declare function mod(a: bigint, b?: bigint): bigint;
declare function invert(number: bigint, modulo?: bigint): bigint;
declare type U8A = Uint8Array;
declare type Sha256FnSync = undefined | ((...messages: Uint8Array[]) => Uint8Array);
declare type HmacFnSync = undefined | ((key: Uint8Array, ...messages: Uint8Array[]) => Uint8Array);
declare function normalizePrivateKey(key: PrivKey): bigint;
export declare function getPublicKey(privateKey: PrivKey, isCompressed?: boolean): Uint8Array;
export declare function recoverPublicKey(msgHash: Hex, signature: Sig, recovery: number, isCompressed?: boolean): Uint8Array;
export declare function getSharedSecret(privateA: PrivKey, publicB: PubKey, isCompressed?: boolean): Uint8Array;
declare type Entropy = Hex | true;
declare type OptsOther = {
    canonical?: boolean;
    der?: boolean;
    extraEntropy?: Entropy;
};
declare type OptsRecov = {
    recovered: true;
} & OptsOther;
declare type OptsNoRecov = {
    recovered?: false;
} & OptsOther;
declare function sign(msgHash: Hex, privKey: PrivKey, opts: OptsRecov): Promise<[U8A, number]>;
declare function sign(msgHash: Hex, privKey: PrivKey, opts?: OptsNoRecov): Promise<U8A>;
declare function signSync(msgHash: Hex, privKey: PrivKey, opts: OptsRecov): [U8A, number];
declare function signSync(msgHash: Hex, privKey: PrivKey, opts?: OptsNoRecov): U8A;
export { sign, signSync };
declare type VOpts = {
    strict?: boolean;
};
export declare function verify(signature: Sig, msgHash: Hex, publicKey: PubKey, opts?: VOpts): boolean;
declare class SchnorrSignature {
    readonly r: bigint;
    readonly s: bigint;
    constructor(r: bigint, s: bigint);
    static fromHex(hex: Hex): SchnorrSignature;
    assertValidity(): void;
    toHex(): string;
    toRawBytes(): Uint8Array;
}
declare function schnorrGetPublicKey(privateKey: PrivKey): Uint8Array;
declare function schnorrSign(msg: Hex, privKey: PrivKey, auxRand?: Hex): Promise<Uint8Array>;
declare function schnorrSignSync(msg: Hex, privKey: PrivKey, auxRand?: Hex): Uint8Array;
declare function schnorrVerify(signature: Hex, message: Hex, publicKey: Hex): Promise<boolean>;
declare function schnorrVerifySync(signature: Hex, message: Hex, publicKey: Hex): boolean;
export declare const schnorr: {
    Signature: typeof SchnorrSignature;
    getPublicKey: typeof schnorrGetPublicKey;
    sign: typeof schnorrSign;
    verify: typeof schnorrVerify;
    signSync: typeof schnorrSignSync;
    verifySync: typeof schnorrVerifySync;
};
export declare const utils: {
    bytesToHex: typeof bytesToHex;
    hexToBytes: typeof hexToBytes;
    concatBytes: typeof concatBytes;
    mod: typeof mod;
    invert: typeof invert;
    isValidPrivateKey(privateKey: PrivKey): boolean;
    _bigintTo32Bytes: typeof numTo32b;
    _normalizePrivateKey: typeof normalizePrivateKey;
    hashToPrivateKey: (hash: Hex) => Uint8Array;
    randomBytes: (bytesLength?: number) => Uint8Array;
    randomPrivateKey: () => Uint8Array;
    precompute(windowSize?: number, point?: Point): Point;
    sha256: (...messages: Uint8Array[]) => Promise<Uint8Array>;
    hmacSha256: (key: Uint8Array, ...messages: Uint8Array[]) => Promise<Uint8Array>;
    sha256Sync: Sha256FnSync;
    hmacSha256Sync: HmacFnSync;
    taggedHash: (tag: string, ...messages: Uint8Array[]) => Promise<Uint8Array>;
    taggedHashSync: (tag: string, ...messages: Uint8Array[]) => Uint8Array;
    _JacobianPoint: typeof JacobianPoint;
};
