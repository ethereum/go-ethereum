import { isBytes as _isBytes } from '@noble/hashes/utils';
import { type Coder } from 'micro-packed';
export declare const isBytes: typeof _isBytes;
export type Web3CallArgs = Partial<{
    to: string;
    from: string;
    data: string;
    nonce: string;
    value: string;
    gas: string;
    gasPrice: string;
    tag: number | 'latest' | 'earliest' | 'pending';
}>;
export type IWeb3Provider = {
    ethCall: (args: Web3CallArgs) => Promise<string>;
    estimateGas: (args: Web3CallArgs) => Promise<bigint>;
    call: (method: string, ...args: any[]) => Promise<any>;
};
export declare const amounts: {
    GWEI_PRECISION: number;
    ETH_PRECISION: number;
    GWEI: bigint;
    ETHER: bigint;
    maxAmount: bigint;
    minGasLimit: bigint;
    maxGasLimit: bigint;
    maxGasPrice: bigint;
    maxNonce: bigint;
    maxDataSize: number;
    maxInitDataSize: number;
    maxChainId: bigint;
    maxUint64: bigint;
    maxUint256: bigint;
};
export declare const ethHex: Coder<Uint8Array, string>;
export declare const ethHexNoLeadingZero: Coder<Uint8Array, string>;
export declare function add0x(hex: string): string;
export declare function strip0x(hex: string): string;
export declare function numberTo0xHex(num: number | bigint): string;
export declare function hexToNumber(hex: string): bigint;
export declare function isObject(item: unknown): item is Record<string, any>;
export declare function astr(str: unknown): void;
export declare function sign(hash: Uint8Array, privKey: Uint8Array, extraEntropy?: boolean | Uint8Array): import("@noble/curves/abstract/weierstrass").RecoveredSignatureType;
export type RawSig = {
    r: bigint;
    s: bigint;
};
export type Sig = RawSig | Uint8Array;
export declare function verify(sig: Sig, hash: Uint8Array, publicKey: Uint8Array): boolean;
export declare function initSig(sig: Sig, bit: number): import("@noble/curves/abstract/weierstrass").RecoveredSignatureType;
export declare function cloneDeep<T>(obj: T): T;
export declare function omit<T extends object, K extends Extract<keyof T, string>>(obj: T, ...keys: K[]): Omit<T, K>;
export declare function zip<A, B>(a: A[], b: B[]): [A, B][];
export declare const createDecimal: (precision: number, round?: boolean) => Coder<bigint, string>;
export declare const weieth: Coder<bigint, string>;
export declare const weigwei: Coder<bigint, string>;
export declare const ethDecimal: typeof weieth;
export declare const gweiDecimal: typeof weigwei;
export declare const formatters: {
    perCentDecimal(precision: number, price: number): bigint;
    formatBigint(amount: bigint, base: bigint, precision: number, fixed?: boolean): string;
    fromWei(wei: string | number | bigint): string;
};
//# sourceMappingURL=utils.d.ts.map