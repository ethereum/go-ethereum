"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.replaceWithinArg = void 0;
const type_guards_1 = require("../../type-guards");
const module_1 = require("../../types/module");
/**
 * Recursively replace values with an argument based on given replacer
 * functions. This is useful for substituting for futures and runtime
 * arguments within the constructor or call args of a future.
 *
 * @example
 * const args = [1, { nested: { insideArray: [1, new Future(1), 3] }}, "abc"]
 *
 * const updated = replaceWithinArg(args, {
 *   ...,
 *   future: (f) => ({_kind: "Future", id: f.id })
 * })
 *
 * assert.equal(updated, [
 *   1,
 *   { nested: { insideArray: [1, {_kind: "Future", id: 1 }, 3] }},
 *   "abc"]
 * )
 *
 * @param arg - the argument to be replaced
 * @param replacers - substituters for each special value found in the args
 * @returns the args with any special subvalue replaced
 */
function replaceWithinArg(arg, replacers) {
    if (typeof arg === "bigint") {
        return replacers.bigint(arg);
    }
    if ((0, type_guards_1.isFuture)(arg)) {
        return replacers.future(arg);
    }
    if ((0, type_guards_1.isRuntimeValue)(arg)) {
        if (arg.type === module_1.RuntimeValueType.ACCOUNT) {
            return replacers.accountRuntimeValue(arg);
        }
        return replacers.moduleParameterRuntimeValue(arg);
    }
    if (Array.isArray(arg)) {
        return arg.map((a) => replaceWithinArg(a, replacers));
    }
    if (typeof arg === "object" && arg !== null) {
        return Object.fromEntries(Object.entries(arg).map(([k, v]) => [k, replaceWithinArg(v, replacers)]));
    }
    return arg;
}
exports.replaceWithinArg = replaceWithinArg;
//# sourceMappingURL=replace-within-arg.js.map