"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const core_1 = require("./core");
const helpers_1 = require("./helpers");
const compile_1 = require("./compile");
function setupBindings(solJson) {
    const coreBindings = (0, core_1.setupCore)(solJson);
    const compileBindings = (0, compile_1.setupCompile)(solJson, coreBindings);
    const methodFlags = (0, helpers_1.getSupportedMethods)(solJson);
    return {
        methodFlags,
        coreBindings,
        compileBindings
    };
}
exports.default = setupBindings;
