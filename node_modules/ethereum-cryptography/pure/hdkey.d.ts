/// <reference types="node" />
export interface Versions {
    private: number;
    public: number;
}
export declare class HDKeyT {
    static HARDENED_OFFSET: number;
    static fromMasterSeed(seed: Buffer, versions?: Versions): HDKeyT;
    static fromExtendedKey(base58key: string, versions?: Versions): HDKeyT;
    static fromJSON(json: {
        xpriv: string;
    }): HDKeyT;
    versions: Versions;
    depth: number;
    index: number;
    chainCode: Buffer | null;
    privateKey: Buffer | null;
    publicKey: Buffer | null;
    fingerprint: number;
    parentFingerprint: number;
    pubKeyHash: Buffer | undefined;
    identifier: Buffer | undefined;
    privateExtendedKey: string;
    publicExtendedKey: string;
    private constructor();
    derive(path: string): HDKeyT;
    deriveChild(index: number): HDKeyT;
    sign(hash: Buffer): Buffer;
    verify(hash: Buffer, signature: Buffer): boolean;
    wipePrivateData(): this;
    toJSON(): {
        xpriv: string;
        xpub: string;
    };
}
export declare const HDKey: typeof HDKeyT;
//# sourceMappingURL=hdkey.d.ts.map