export interface ABIArgumentLengthErrorType extends Error {
    code: "INVALID_ARGUMENT";
    count: {
        types: number;
        values: number;
    };
    value: {
        types: Array<{
            name: string;
            type: string;
        }>;
        values: any[];
    };
    reason: string;
}
export interface ABIArgumentTypeErrorType extends Error {
    code: "INVALID_ARGUMENT";
    argument: string;
    value: any;
    reason: string;
}
export interface ABIArgumentOverflowErrorType extends Error {
    code: "NUMERIC_FAULT";
    fault: "overflow";
    value: any;
    reason: string;
    operation: string;
}
export declare function isABIArgumentLengthError(error: any): error is ABIArgumentLengthErrorType;
export declare function isABIArgumentTypeError(error: any): error is ABIArgumentTypeErrorType;
export declare function isABIArgumentOverflowError(error: any): error is ABIArgumentOverflowErrorType;
//# sourceMappingURL=abi-validation-extras.d.ts.map