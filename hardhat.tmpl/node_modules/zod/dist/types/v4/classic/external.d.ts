export * as core from "zod/v4/core";
export * from "./schemas.js";
export * from "./checks.js";
export * from "./errors.js";
export * from "./parse.js";
export * from "./compat.js";
export type { infer, output, input } from "zod/v4/core";
export { globalRegistry, type GlobalMeta, registry, config, function, $output, $input, $brand, clone, regexes, treeifyError, prettifyError, formatError, flattenError, toJSONSchema, locales, } from "zod/v4/core";
