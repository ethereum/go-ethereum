import { _tuple } from "./api.js";
import { parse, parseAsync } from "./parse.js";
import * as schemas from "./schemas.js";
import { $ZodTuple } from "./schemas.js";
export class $ZodFunction {
    constructor(def) {
        this._def = def;
    }
    implement(func) {
        if (typeof func !== "function") {
            throw new Error("implement() must be called with a function");
        }
        const impl = ((...args) => {
            const parsedArgs = this._def.input ? parse(this._def.input, args, undefined, { callee: impl }) : args;
            if (!Array.isArray(parsedArgs)) {
                throw new Error("Invalid arguments schema: not an array or tuple schema.");
            }
            const output = func(...parsedArgs);
            return this._def.output ? parse(this._def.output, output, undefined, { callee: impl }) : output;
        });
        return impl;
    }
    implementAsync(func) {
        if (typeof func !== "function") {
            throw new Error("implement() must be called with a function");
        }
        const impl = (async (...args) => {
            const parsedArgs = this._def.input ? await parseAsync(this._def.input, args, undefined, { callee: impl }) : args;
            if (!Array.isArray(parsedArgs)) {
                throw new Error("Invalid arguments schema: not an array or tuple schema.");
            }
            const output = await func(...parsedArgs);
            return this._def.output ? parseAsync(this._def.output, output, undefined, { callee: impl }) : output;
        });
        return impl;
    }
    input(...args) {
        if (Array.isArray(args[0])) {
            return new $ZodFunction({
                type: "function",
                input: new $ZodTuple({
                    type: "tuple",
                    items: args[0],
                    rest: args[1],
                }),
                output: this._def.output,
            });
        }
        return new $ZodFunction({
            type: "function",
            input: args[0],
            output: this._def.output,
        });
    }
    output(output) {
        return new $ZodFunction({
            type: "function",
            input: this._def.input,
            output,
        });
    }
}
function _function(params) {
    return new $ZodFunction({
        type: "function",
        input: Array.isArray(params?.input) ? _tuple(schemas.$ZodTuple, params?.input) : (params?.input ?? null),
        output: params?.output ?? null,
    });
}
export { _function as function };
