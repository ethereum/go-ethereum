export declare const PANIC_CODES: {
    ASSERTION_ERROR: number;
    ARITHMETIC_OVERFLOW: number;
    DIVISION_BY_ZERO: number;
    ENUM_CONVERSION_OUT_OF_BOUNDS: number;
    INCORRECTLY_ENCODED_STORAGE_BYTE_ARRAY: number;
    POP_ON_EMPTY_ARRAY: number;
    ARRAY_ACCESS_OUT_OF_BOUNDS: number;
    TOO_MUCH_MEMORY_ALLOCATED: number;
    ZERO_INITIALIZED_VARIABLE: number;
};
export declare function panicErrorCodeToReason(errorCode: bigint): string | undefined;
//# sourceMappingURL=panic.d.ts.map