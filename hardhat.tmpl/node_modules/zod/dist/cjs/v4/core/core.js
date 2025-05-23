"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.globalConfig = exports.$ZodAsyncError = exports.$brand = void 0;
exports.$constructor = $constructor;
exports.config = config;
function $constructor(name, initializer, params) {
    const Parent = params?.Parent ?? Object;
    class _ extends Parent {
        constructor(def) {
            var _a;
            super();
            const th = this;
            _.init(th, def);
            (_a = th._zod).deferred ?? (_a.deferred = []);
            for (const fn of th._zod.deferred) {
                fn();
            }
        }
        static init(inst, def) {
            var _a;
            Object.defineProperty(inst, "_zod", {
                value: inst._zod ?? {},
                enumerable: false,
            });
            // inst._zod ??= {} as any;
            (_a = inst._zod).traits ?? (_a.traits = new Set());
            // const seen = inst._zod.traits.has(name);
            inst._zod.traits.add(name);
            initializer(inst, def);
            // support prototype modifications
            for (const k in _.prototype) {
                Object.defineProperty(inst, k, { value: _.prototype[k].bind(inst) });
            }
            inst._zod.constr = _;
            inst._zod.def = def;
        }
        static [Symbol.hasInstance](inst) {
            if (params?.Parent && inst instanceof params.Parent)
                return true;
            return inst?._zod?.traits?.has(name);
        }
    }
    Object.defineProperty(_, "name", { value: name });
    return _;
}
//////////////////////////////   UTILITIES   ///////////////////////////////////////
exports.$brand = Symbol("zod_brand");
class $ZodAsyncError extends Error {
    constructor() {
        super(`Encountered Promise during synchronous parse. Use .parseAsync() instead.`);
    }
}
exports.$ZodAsyncError = $ZodAsyncError;
exports.globalConfig = {};
function config(newConfig) {
    if (newConfig)
        Object.assign(exports.globalConfig, newConfig);
    return exports.globalConfig;
}
