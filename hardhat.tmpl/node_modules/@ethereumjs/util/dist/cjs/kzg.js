"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.initKZG = void 0;
/**
 * @deprecated This initialization method is deprecated since trusted setup loading is done directly in the reference KZG library
 * initialization or should othewise be assured independently before KZG libary usage.
 *
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param a dictionary of trusted setup options
 */
function initKZG(kzg, _trustedSetupPath) {
    kzg.loadTrustedSetup();
}
exports.initKZG = initKZG;
//# sourceMappingURL=kzg.js.map