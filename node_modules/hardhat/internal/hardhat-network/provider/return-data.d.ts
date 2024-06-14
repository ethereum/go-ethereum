/**
 * Represents the returnData of a transaction, whose contents are unknown.
 */
export declare class ReturnData {
    value: Uint8Array;
    private _selector;
    constructor(value: Uint8Array);
    isEmpty(): boolean;
    matchesSelector(selector: Uint8Array): boolean;
    isErrorReturnData(): boolean;
    isPanicReturnData(): boolean;
    decodeError(): string;
    decodePanic(): bigint;
    getSelector(): string | undefined;
}
//# sourceMappingURL=return-data.d.ts.map