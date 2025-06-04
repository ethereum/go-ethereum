/**
 * @deprecated This initialization method is deprecated since trusted setup loading is done directly in the reference KZG library
 * initialization or should othewise be assured independently before KZG libary usage.
 *
 * @param kzgLib a KZG implementation (defaults to c-kzg)
 * @param a dictionary of trusted setup options
 */
export function initKZG(kzg, _trustedSetupPath) {
    kzg.loadTrustedSetup();
}
//# sourceMappingURL=kzg.js.map