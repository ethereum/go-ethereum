import type * as errors from "./errors.js";
import type * as schemas from "./schemas.js";
import type { Class } from "./util.js";
type ZodTrait = {
    _zod: {
        def: any;
        [k: string]: any;
    };
};
export interface $constructor<T extends ZodTrait, D = T["_zod"]["def"]> {
    new (def: D): T;
    init(inst: T, def: D): asserts inst is T;
}
export declare function $constructor<T extends ZodTrait, D = T["_zod"]["def"]>(name: string, initializer: (inst: T, def: D) => void, params?: {
    Parent?: typeof Class;
}): $constructor<T, D>;
export declare const $brand: unique symbol;
export type $brand<T extends string | number | symbol = string | number | symbol> = {
    [$brand]: {
        [k in T]: true;
    };
};
export type $ZodBranded<T extends schemas.$ZodType, Brand extends string | number | symbol> = T & Record<"_zod", Record<"~output", output<T> & $brand<Brand>>>;
export declare class $ZodAsyncError extends Error {
    constructor();
}
export type input<T extends schemas.$ZodType> = T["_zod"] extends {
    "~input": any;
} ? T["_zod"]["~input"] : T["_zod"]["input"];
export type output<T extends schemas.$ZodType> = T["_zod"] extends {
    "~output": any;
} ? T["_zod"]["~output"] : T["_zod"]["output"];
export type { output as infer };
export interface $ZodConfig {
    /** Custom error map. Overrides `config().localeError`. */
    customError?: errors.$ZodErrorMap | undefined;
    /** Localized error map. Lowest priority. */
    localeError?: errors.$ZodErrorMap | undefined;
    /** Disable JIT schema compilation. Useful in environments that disallow `eval`. */
    jitless?: boolean | undefined;
}
export declare const globalConfig: $ZodConfig;
export declare function config(newConfig?: Partial<$ZodConfig>): $ZodConfig;
