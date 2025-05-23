export * as core from "zod/v4/core";
export * from "./schemas.js";
export * from "./checks.js";
export * from "./errors.js";
export * from "./parse.js";
export * from "./compat.js";
// zod-specified
import { config } from "zod/v4/core";
import en from "zod/v4/locales/en.js";
config(en());
export { globalRegistry, registry, config, function, $output, $input, $brand, clone, regexes, treeifyError, prettifyError, formatError, flattenError, toJSONSchema, locales, } from "zod/v4/core";
