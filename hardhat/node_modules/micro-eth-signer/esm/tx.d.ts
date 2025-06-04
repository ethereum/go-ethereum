import * as P from 'micro-packed';
export type AnyCoder = Record<string, P.Coder<any, any>>;
export type AnyCoderStream = Record<string, P.CoderType<any>>;
type VersionType<V extends AnyCoderStream> = {
    [K in keyof V]: {
        type: K;
        data: P.UnwrapCoder<V[K]>;
    };
}[keyof V];
export type TxCoder<T extends TxType> = P.UnwrapCoder<(typeof TxVersions)[T]>;
type VRS = Partial<{
    v: bigint;
    r: bigint;
    s: bigint;
}>;
type YRS = Partial<{
    chainId: bigint;
    yParity: number;
    r: bigint;
    s: bigint;
}>;
export declare const legacySig: P.Coder<VRS, YRS>;
type CoderOutput<F> = F extends P.Coder<any, infer T> ? T : never;
declare const accessListItem: P.Coder<(Uint8Array | Uint8Array[])[], {
    address: string;
    storageKeys: string[];
}>;
export type AccessList = CoderOutput<typeof accessListItem>[];
export declare const authorizationRequest: P.Coder<Uint8Array[], {
    chainId: bigint;
    address: string;
    nonce: bigint;
}>;
declare const authorizationItem: P.Coder<Uint8Array[], {
    chainId: bigint;
    address: string;
    nonce: bigint;
    yParity: number;
    r: bigint;
    s: bigint;
}>;
export type AuthorizationItem = CoderOutput<typeof authorizationItem>;
export type AuthorizationRequest = CoderOutput<typeof authorizationRequest>;
/**
 * Field types, matching geth. Either u64 or u256.
 */
declare const coders: {
    chainId: P.Coder<P.Bytes, bigint>;
    nonce: P.Coder<P.Bytes, bigint>;
    gasPrice: P.Coder<P.Bytes, bigint>;
    maxPriorityFeePerGas: P.Coder<P.Bytes, bigint>;
    maxFeePerGas: P.Coder<P.Bytes, bigint>;
    gasLimit: P.Coder<P.Bytes, bigint>;
    to: P.Coder<Uint8Array<ArrayBufferLike>, string>;
    value: P.Coder<P.Bytes, bigint>;
    data: P.Coder<Uint8Array<ArrayBufferLike>, string>;
    accessList: P.Coder<(Uint8Array<ArrayBufferLike> | Uint8Array<ArrayBufferLike>[])[][], {
        address: string;
        storageKeys: string[];
    }[]>;
    maxFeePerBlobGas: P.Coder<P.Bytes, bigint>;
    blobVersionedHashes: P.Coder<Uint8Array<ArrayBufferLike>[], string[]>;
    yParity: P.Coder<P.Bytes, number>;
    v: P.Coder<P.Bytes, bigint>;
    r: P.Coder<P.Bytes, bigint>;
    s: P.Coder<P.Bytes, bigint>;
    authorizationList: P.Coder<Uint8Array<ArrayBufferLike>[][], {
        chainId: bigint;
        address: string;
        nonce: bigint;
        yParity: number;
        r: bigint;
        s: bigint;
    }[]>;
};
type Coders = typeof coders;
type CoderName = keyof Coders;
type OptFields<T, O> = T & Partial<O>;
type FieldCoder<C> = P.CoderType<C> & {
    fields: CoderName[];
    optionalFields: CoderName[];
    setOfAllFields: Set<CoderName | 'type'>;
};
export declare function removeSig(raw: TxCoder<any>): TxCoder<any>;
declare const legacyInternal: FieldCoder<OptFields<{
    nonce: bigint;
    gasPrice: bigint;
    gasLimit: bigint;
    to: string;
    value: bigint;
    data: string;
}, {
    r: bigint;
    s: bigint;
    v: bigint;
}>>;
type LegacyInternal = P.UnwrapCoder<typeof legacyInternal>;
type Legacy = Omit<LegacyInternal, 'v'> & {
    chainId?: bigint;
    yParity?: number;
};
export declare const TxVersions: {
    legacy: FieldCoder<Legacy>;
    eip2930: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        gasPrice: bigint;
        chainId: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip1559: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip4844: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
        maxFeePerBlobGas: bigint;
        blobVersionedHashes: string[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip7702: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
        authorizationList: {
            chainId: bigint;
            address: string;
            nonce: bigint;
            yParity: number;
            r: bigint;
            s: bigint;
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
};
export declare const RawTx: P.CoderType<VersionType<{
    legacy: FieldCoder<Legacy>;
    eip2930: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        gasPrice: bigint;
        chainId: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip1559: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip4844: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
        maxFeePerBlobGas: bigint;
        blobVersionedHashes: string[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
    eip7702: FieldCoder<OptFields<{
        value: bigint;
        to: string;
        data: string;
        nonce: bigint;
        chainId: bigint;
        maxPriorityFeePerGas: bigint;
        maxFeePerGas: bigint;
        gasLimit: bigint;
        accessList: {
            address: string;
            storageKeys: string[];
        }[];
        authorizationList: {
            chainId: bigint;
            address: string;
            nonce: bigint;
            yParity: number;
            r: bigint;
            s: bigint;
        }[];
    }, {
        r: bigint;
        yParity: number;
        s: bigint;
    }>>;
}>>;
/**
 * Unchecked TX for debugging. Returns raw Uint8Array-s.
 * Handles versions - plain RLP will crash on it.
 */
export declare const RlpTx: P.CoderType<{
    type: string;
    data: import('./rlp.js').RLPInput;
}>;
export type TxType = keyof typeof TxVersions;
type ErrObj = {
    field: string;
    error: string;
};
export declare class AggregatedError extends Error {
    message: string;
    errors: ErrObj[];
    constructor(message: string, errors: ErrObj[]);
}
export declare function validateFields(type: TxType, data: Record<string, any>, strict?: boolean, allowSignatureFields?: boolean): void;
export declare function sortRawData(raw: TxCoder<any>): any;
export declare function decodeLegacyV(raw: TxCoder<any>): bigint | undefined;
export declare const __tests: any;
export {};
//# sourceMappingURL=tx.d.ts.map