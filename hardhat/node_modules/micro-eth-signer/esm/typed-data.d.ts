import type { GetType as AbiGetType } from './abi/decoder.ts';
export type Hex = string | Uint8Array;
export interface TypedSigner<T> {
    _getHash: (message: T) => string;
    sign(message: T, privateKey: Hex, extraEntropy?: boolean | Uint8Array): string;
    recoverPublicKey(signature: string, message: T): string;
    verify(signature: string, message: T, address: string): boolean;
}
export declare const personal: TypedSigner<string | Uint8Array>;
export type EIP712Component = {
    name: string;
    type: string;
};
export type EIP712Types = Record<string, readonly EIP712Component[]>;
type ProcessType<T extends string, Types extends EIP712Types> = T extends `${infer Base}[]${infer Rest}` ? ProcessType<`${Base}${Rest}`, Types>[] : T extends `${infer Base}[${number}]${infer Rest}` ? ProcessType<`${Base}${Rest}`, Types>[] : T extends keyof Types ? GetType<Types, T> | undefined : AbiGetType<T>;
export type GetType<Types extends EIP712Types, K extends keyof Types & string> = {
    [C in Types[K][number] as C['name']]: ProcessType<C['type'], Types>;
};
type Key<T extends EIP712Types> = keyof T & string;
export declare function encoder<T extends EIP712Types>(types: T, domain: GetType<T, 'EIP712Domain'>): {
    encodeData: <K extends Key<T>>(type: K, message: GetType<T, K>) => string;
    structHash: <K extends Key<T>>(type: K, message: GetType<T, K>) => string;
    _getHash: <K extends Key<T>>(primaryType: K, message: GetType<T, K>) => string;
    sign: <K extends Key<T>>(primaryType: K, message: GetType<T, K>, privateKey: Hex, extraEntropy?: boolean | Uint8Array) => string;
    verify: <K extends Key<T>>(primaryType: K, signature: string, message: GetType<T, K>, address: string) => boolean;
    recoverPublicKey: <K extends Key<T>>(primaryType: K, signature: string, message: GetType<T, K>) => string;
};
export declare const EIP712Domain: readonly [{
    readonly name: "name";
    readonly type: "string";
}, {
    readonly name: "version";
    readonly type: "string";
}, {
    readonly name: "chainId";
    readonly type: "uint256";
}, {
    readonly name: "verifyingContract";
    readonly type: "address";
}, {
    readonly name: "salt";
    readonly type: "bytes32";
}];
export type DomainParams = typeof EIP712Domain;
declare const domainTypes: {
    EIP712Domain: DomainParams;
};
export type EIP712Domain = GetType<typeof domainTypes, 'EIP712Domain'>;
export declare function getDomainType(domain: EIP712Domain): ({
    readonly name: "name";
    readonly type: "string";
} | {
    readonly name: "version";
    readonly type: "string";
} | {
    readonly name: "chainId";
    readonly type: "uint256";
} | {
    readonly name: "verifyingContract";
    readonly type: "address";
} | {
    readonly name: "salt";
    readonly type: "bytes32";
})[];
export type TypedData<T extends EIP712Types, K extends Key<T>> = {
    types: T;
    primaryType: K;
    domain: GetType<T, 'EIP712Domain'>;
    message: GetType<T, K>;
};
export declare function encodeData<T extends EIP712Types, K extends Key<T>>(typed: TypedData<T, K>): string;
export declare function sigHash<T extends EIP712Types, K extends Key<T>>(typed: TypedData<T, K>): string;
export declare function signTyped<T extends EIP712Types, K extends Key<T>>(typed: TypedData<T, K>, privateKey: Hex, extraEntropy?: boolean | Uint8Array): string;
export declare function verifyTyped<T extends EIP712Types, K extends Key<T>>(signature: string, typed: TypedData<T, K>, address: string): boolean;
export declare function recoverPublicKeyTyped<T extends EIP712Types, K extends Key<T>>(signature: string, typed: TypedData<T, K>): string;
export declare const _TEST: any;
export {};
//# sourceMappingURL=typed-data.d.ts.map