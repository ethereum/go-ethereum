export declare const HARDENED_OFFSET: number;
export interface Versions {
    private: number;
    public: number;
}
interface HDKeyOpt {
    versions?: Versions;
    depth?: number;
    index?: number;
    parentFingerprint?: number;
    chainCode?: Uint8Array;
    publicKey?: Uint8Array;
    privateKey?: Uint8Array | bigint;
}
export declare class HDKey {
    get fingerprint(): number;
    get identifier(): Uint8Array | undefined;
    get pubKeyHash(): Uint8Array | undefined;
    get privateKey(): Uint8Array | null;
    get publicKey(): Uint8Array | null;
    get privateExtendedKey(): string;
    get publicExtendedKey(): string;
    static fromMasterSeed(seed: Uint8Array, versions?: Versions): HDKey;
    static fromExtendedKey(base58key: string, versions?: Versions): HDKey;
    static fromJSON(json: {
        xpriv: string;
    }): HDKey;
    readonly versions: Versions;
    readonly depth: number;
    readonly index: number;
    readonly chainCode: Uint8Array | null;
    readonly parentFingerprint: number;
    private privKey?;
    private privKeyBytes?;
    private pubKey?;
    private pubHash;
    constructor(opt: HDKeyOpt);
    derive(path: string): HDKey;
    deriveChild(index: number): HDKey;
    sign(hash: Uint8Array): Uint8Array;
    verify(hash: Uint8Array, signature: Uint8Array): boolean;
    wipePrivateData(): this;
    toJSON(): {
        xpriv: string;
        xpub: string;
    };
    private serialize;
}
export {};
//# sourceMappingURL=index.d.ts.map