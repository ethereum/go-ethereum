declare type Output = Uint8Array | ((len: number) => Uint8Array);
interface Signature {
    signature: Uint8Array;
    recid: number;
}
export declare function createPrivateKeySync(): Uint8Array;
export declare function createPrivateKey(): Promise<Uint8Array>;
export declare function privateKeyVerify(privateKey: Uint8Array): boolean;
export declare function publicKeyCreate(privateKey: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function publicKeyVerify(publicKey: Uint8Array): boolean;
export declare function publicKeyConvert(publicKey: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function ecdsaSign(msgHash: Uint8Array, privateKey: Uint8Array, options?: {
    noncefn: undefined;
    data: undefined;
}, out?: Output): Signature;
export declare function ecdsaRecover(signature: Uint8Array, recid: number, msgHash: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function ecdsaVerify(signature: Uint8Array, msgHash: Uint8Array, publicKey: Uint8Array): boolean;
export declare function privateKeyTweakAdd(privateKey: Uint8Array, tweak: Uint8Array): Uint8Array;
export declare function privateKeyNegate(privateKey: Uint8Array): Uint8Array;
export declare function publicKeyNegate(publicKey: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function publicKeyCombine(publicKeys: Uint8Array[], compressed?: boolean, out?: Output): Uint8Array;
export declare function publicKeyTweakAdd(publicKey: Uint8Array, tweak: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function publicKeyTweakMul(publicKey: Uint8Array, tweak: Uint8Array, compressed?: boolean, out?: Output): Uint8Array;
export declare function privateKeyTweakMul(privateKey: Uint8Array, tweak: Uint8Array): Uint8Array;
export declare function signatureExport(signature: Uint8Array, out?: Output): Uint8Array;
export declare function signatureImport(signature: Uint8Array, out?: Output): Uint8Array;
export declare function signatureNormalize(signature: Uint8Array): Uint8Array;
export declare function ecdh(publicKey: Uint8Array, privateKey: Uint8Array, options?: {
    xbuf?: Uint8Array;
    ybuf?: Uint8Array;
    data?: Uint8Array;
    hashfn?: (x: Uint8Array, y: Uint8Array, data: Uint8Array) => Uint8Array;
}, out?: Output): Uint8Array;
export declare function contextRandomize(seed: Uint8Array): void;
export {};
