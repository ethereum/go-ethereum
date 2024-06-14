import { ErrorDescriptor } from "./types/errors";
/**
 * Base error class extended by all custom errors.
 * Placeholder to allow us to customize error output formatting in the future.
 *
 * @beta
 */
export declare class CustomError extends Error {
    constructor(message: string, cause?: Error);
}
/**
 * All exceptions intentionally thrown with Ignition-core
 * extend this class.
 *
 * @beta
 */
export declare class IgnitionError extends CustomError {
    #private;
    constructor(errorDescriptor: ErrorDescriptor, messageArguments?: Record<string, string | number>, cause?: Error);
    get errorNumber(): number;
}
/**
 * This class is used to throw errors from Ignition plugins made by third parties.
 *
 * @beta
 */
export declare class IgnitionPluginError extends CustomError {
    static isIgnitionPluginError(error: any): error is IgnitionPluginError;
    private readonly _isIgnitionPluginError;
    readonly pluginName: string;
    constructor(pluginName: string, message: string, cause?: Error);
}
/**
 * This class is used to throw errors from *core* Ignition plugins.
 * If you are developing a third-party plugin, use IgnitionPluginError instead.
 *
 * @beta
 */
export declare class NomicIgnitionPluginError extends IgnitionPluginError {
    static isNomicIgnitionPluginError(error: any): error is NomicIgnitionPluginError;
    private readonly _isNomicIgnitionPluginError;
}
//# sourceMappingURL=errors.d.ts.map