/// <reference types="node" />
export declare function privateKeyVerify(privateKey: Buffer): boolean;
export declare function publicKeyCreate(privateKey: Buffer, compressed?: boolean): Buffer;
export declare function publicKeyVerify(publicKey: Buffer): boolean;
export declare function publicKeyConvert(publicKey: Buffer, compressed?: boolean): Buffer;
export declare function privateKeyTweakAdd(publicKey: Buffer, tweak: Buffer): Buffer;
export declare function publicKeyTweakAdd(publicKey: Buffer, tweak: Buffer, compressed?: boolean): Buffer;
export declare function sign(message: Buffer, privateKey: Buffer): {
    signature: Buffer;
    recovery: number;
};
export declare function verify(message: Buffer, signature: Buffer, publicKey: Buffer): boolean;
//# sourceMappingURL=hdkey-secp256k1v3.d.ts.map