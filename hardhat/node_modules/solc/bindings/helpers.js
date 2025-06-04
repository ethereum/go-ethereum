"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getSupportedMethods = exports.bindSolcMethodWithFallbackFunc = exports.bindSolcMethod = void 0;
const helpers_1 = require("../common/helpers");
function bindSolcMethod(solJson, method, returnType, args, defaultValue) {
    if ((0, helpers_1.isNil)(solJson[`_${method}`]) && defaultValue !== undefined) {
        return defaultValue;
    }
    return solJson.cwrap(method, returnType, args);
}
exports.bindSolcMethod = bindSolcMethod;
function bindSolcMethodWithFallbackFunc(solJson, method, returnType, args, fallbackMethod, finalFallback = undefined) {
    const methodFunc = bindSolcMethod(solJson, method, returnType, args, null);
    if (!(0, helpers_1.isNil)(methodFunc)) {
        return methodFunc;
    }
    return bindSolcMethod(solJson, fallbackMethod, returnType, args, finalFallback);
}
exports.bindSolcMethodWithFallbackFunc = bindSolcMethodWithFallbackFunc;
function getSupportedMethods(solJson) {
    return {
        licenseSupported: anyMethodExists(solJson, 'solidity_license'),
        versionSupported: anyMethodExists(solJson, 'solidity_version'),
        allocSupported: anyMethodExists(solJson, 'solidity_alloc'),
        resetSupported: anyMethodExists(solJson, 'solidity_reset'),
        compileJsonSupported: anyMethodExists(solJson, 'compileJSON'),
        compileJsonMultiSupported: anyMethodExists(solJson, 'compileJSONMulti'),
        compileJsonCallbackSuppported: anyMethodExists(solJson, 'compileJSONCallback'),
        compileJsonStandardSupported: anyMethodExists(solJson, 'compileStandard', 'solidity_compile')
    };
}
exports.getSupportedMethods = getSupportedMethods;
function anyMethodExists(solJson, ...names) {
    return names.some(name => !(0, helpers_1.isNil)(solJson[`_${name}`]));
}
