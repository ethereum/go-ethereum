import { BytesLike, Signature, SignatureLike } from "@ethersproject/bytes";
export declare class SigningKey {
    readonly curve: string;
    readonly privateKey: string;
    readonly publicKey: string;
    readonly compressedPublicKey: string;
    readonly _isSigningKey: boolean;
    constructor(privateKey: BytesLike);
    _addPoint(other: BytesLike): string;
    signDigest(digest: BytesLike): Signature;
    computeSharedSecret(otherKey: BytesLike): string;
    static isSigningKey(value: any): value is SigningKey;
}
export declare function recoverPublicKey(digest: BytesLike, signature: SignatureLike): string;
export declare function computePublicKey(key: BytesLike, compressed?: boolean): string;
//# sourceMappingURL=index.d.ts.map