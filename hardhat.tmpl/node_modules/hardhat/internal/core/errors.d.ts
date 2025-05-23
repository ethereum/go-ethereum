import { ErrorDescriptor } from "./errors-list";
declare const inspect: unique symbol;
export declare class CustomError extends Error {
    readonly parent?: Error | undefined;
    private _stack;
    constructor(message: string, parent?: Error | undefined);
    [inspect](): string;
}
export declare class HardhatError extends CustomError {
    static isHardhatError(other: any): other is HardhatError;
    static isHardhatErrorType(other: any, descriptor: ErrorDescriptor): other is HardhatError;
    readonly errorDescriptor: ErrorDescriptor;
    readonly number: number;
    readonly messageArguments: Record<string, any>;
    private readonly _isHardhatError;
    constructor(errorDescriptor: ErrorDescriptor, messageArguments?: Record<string, string | number>, parentError?: Error);
}
/**
 * This class is used to throw errors from hardhat plugins made by third parties.
 */
export declare class HardhatPluginError extends CustomError {
    static isHardhatPluginError(other: any): other is HardhatPluginError;
    readonly pluginName: string;
    private readonly _isHardhatPluginError;
    /**
     * Creates a HardhatPluginError.
     *
     * @param pluginName The name of the plugin.
     * @param message An error message that will be shown to the user.
     * @param parent The error that causes this error to be thrown.
     */
    constructor(pluginName: string, message: string, parent?: Error);
    /**
     * A DEPRECATED constructor that automatically obtains the caller package and
     * use it as plugin name.
     *
     * @deprecated Use the above constructor.
     *
     * @param message An error message that will be shown to the user.
     * @param parent The error that causes this error to be thrown.
     */
    constructor(message: string, parent?: Error);
}
export declare class NomicLabsHardhatPluginError extends HardhatPluginError {
    shouldBeReported: boolean;
    static isNomicLabsHardhatPluginError(other: any): other is NomicLabsHardhatPluginError;
    private readonly _isNomicLabsHardhatPluginError;
    /**
     * This class is used to throw errors from *core* hardhat plugins. If you are
     * developing a third-party plugin, use HardhatPluginError instead.
     */
    constructor(pluginName: string, message: string, parent?: Error, shouldBeReported?: boolean);
}
/**
 * This function applies error messages templates like this:
 *
 *  - Template is a string which contains a variable tags. A variable tag is a
 *    a variable name surrounded by %. Eg: %plugin1%
 *  - A variable name is a string of alphanumeric ascii characters.
 *  - Every variable tag is replaced by its value.
 *  - %% is replaced by %.
 *  - Values can't contain variable tags.
 *  - If a variable is not present in the template, but present in the values
 *    object, an error is thrown.
 *
 * @param template The template string.
 * @param values A map of variable names to their values.
 */
export declare function applyErrorMessageTemplate(template: string, values: {
    [templateVar: string]: any;
}): string;
export declare function assertHardhatInvariant(invariant: boolean, message: string): asserts invariant;
export {};
//# sourceMappingURL=errors.d.ts.map