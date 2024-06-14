import type { HardhatConfig as HardhatConfigT } from "../../../types";
import type { ValidationError } from "io-ts/lib";
import type { Reporter } from "io-ts/lib/Reporter";
import * as t from "io-ts";
export declare function failure(es: ValidationError[]): string[];
export declare function success(): string[];
export declare const DotPathReporter: Reporter<string[]>;
export declare const hexString: t.Type<string, string, unknown>;
export declare const address: t.Type<string, string, unknown>;
export declare const decimalString: t.Type<string, string, unknown>;
/**
 * Validates the config, throwing a HardhatError if invalid.
 * @param config
 */
export declare function validateConfig(config: any): void;
export declare function getValidationErrors(config: any): string[];
export declare function validateResolvedConfig(resolvedConfig: HardhatConfigT): void;
//# sourceMappingURL=config-validation.d.ts.map