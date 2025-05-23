export declare class AbiHelpers {
    /**
     * Try to compute the selector for the function/event/error
     * with the given name and param types. Return undefined
     * if it cannot do it. This can happen if some ParamType is
     * not understood by @ethersproject/abi
     */
    static computeSelector(name: string, inputs: any[]): Uint8Array | undefined;
    static isValidCalldata(inputs: any[], calldata: Uint8Array): boolean;
    static formatValues(values: any[]): string;
    private static _formatValue;
}
//# sourceMappingURL=abi-helpers.d.ts.map