import type { PrefixedHexString } from './types.js';
export declare type RequestBytes = Uint8Array;
export declare enum CLRequestType {
    Deposit = 0,
    Withdrawal = 1,
    Consolidation = 2
}
export declare type DepositRequestV1 = {
    pubkey: PrefixedHexString;
    withdrawalCredentials: PrefixedHexString;
    amount: PrefixedHexString;
    signature: PrefixedHexString;
    index: PrefixedHexString;
};
export declare type WithdrawalRequestV1 = {
    sourceAddress: PrefixedHexString;
    validatorPubkey: PrefixedHexString;
    amount: PrefixedHexString;
};
export declare type ConsolidationRequestV1 = {
    sourceAddress: PrefixedHexString;
    sourcePubkey: PrefixedHexString;
    targetPubkey: PrefixedHexString;
};
export interface RequestJSON {
    [CLRequestType.Deposit]: DepositRequestV1;
    [CLRequestType.Withdrawal]: WithdrawalRequestV1;
    [CLRequestType.Consolidation]: ConsolidationRequestV1;
}
export declare type DepositRequestData = {
    pubkey: Uint8Array;
    withdrawalCredentials: Uint8Array;
    amount: bigint;
    signature: Uint8Array;
    index: bigint;
};
export declare type WithdrawalRequestData = {
    sourceAddress: Uint8Array;
    validatorPubkey: Uint8Array;
    amount: bigint;
};
export declare type ConsolidationRequestData = {
    sourceAddress: Uint8Array;
    sourcePubkey: Uint8Array;
    targetPubkey: Uint8Array;
};
export interface RequestData {
    [CLRequestType.Deposit]: DepositRequestData;
    [CLRequestType.Withdrawal]: WithdrawalRequestData;
    [CLRequestType.Consolidation]: ConsolidationRequestData;
}
export declare type TypedRequestData = RequestData[CLRequestType];
export interface CLRequestInterface<T extends CLRequestType = CLRequestType> {
    readonly type: T;
    serialize(): Uint8Array;
    toJSON(): RequestJSON[T];
}
export declare abstract class CLRequest<T extends CLRequestType> implements CLRequestInterface<T> {
    readonly type: T;
    abstract serialize(): Uint8Array;
    abstract toJSON(): RequestJSON[T];
    constructor(type: T);
}
export declare class DepositRequest extends CLRequest<CLRequestType.Deposit> {
    readonly pubkey: Uint8Array;
    readonly withdrawalCredentials: Uint8Array;
    readonly amount: bigint;
    readonly signature: Uint8Array;
    readonly index: bigint;
    constructor(pubkey: Uint8Array, withdrawalCredentials: Uint8Array, amount: bigint, signature: Uint8Array, index: bigint);
    static fromRequestData(depositData: DepositRequestData): DepositRequest;
    static fromJSON(jsonData: DepositRequestV1): DepositRequest;
    serialize(): Uint8Array;
    toJSON(): DepositRequestV1;
    static deserialize(bytes: Uint8Array): DepositRequest;
}
export declare class WithdrawalRequest extends CLRequest<CLRequestType.Withdrawal> {
    readonly sourceAddress: Uint8Array;
    readonly validatorPubkey: Uint8Array;
    readonly amount: bigint;
    constructor(sourceAddress: Uint8Array, validatorPubkey: Uint8Array, amount: bigint);
    static fromRequestData(withdrawalData: WithdrawalRequestData): WithdrawalRequest;
    static fromJSON(jsonData: WithdrawalRequestV1): WithdrawalRequest;
    serialize(): Uint8Array;
    toJSON(): WithdrawalRequestV1;
    static deserialize(bytes: Uint8Array): WithdrawalRequest;
}
export declare class ConsolidationRequest extends CLRequest<CLRequestType.Consolidation> {
    readonly sourceAddress: Uint8Array;
    readonly sourcePubkey: Uint8Array;
    readonly targetPubkey: Uint8Array;
    constructor(sourceAddress: Uint8Array, sourcePubkey: Uint8Array, targetPubkey: Uint8Array);
    static fromRequestData(consolidationData: ConsolidationRequestData): ConsolidationRequest;
    static fromJSON(jsonData: ConsolidationRequestV1): ConsolidationRequest;
    serialize(): Uint8Array;
    toJSON(): ConsolidationRequestV1;
    static deserialize(bytes: Uint8Array): ConsolidationRequest;
}
export declare class CLRequestFactory {
    static fromSerializedRequest(bytes: Uint8Array): CLRequest<CLRequestType>;
}
//# sourceMappingURL=requests.d.ts.map