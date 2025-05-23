"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getDefaultTypechainConfig = void 0;
function getDefaultTypechainConfig(config) {
    const defaultConfig = {
        outDir: 'typechain-types',
        target: 'ethers-v6',
        alwaysGenerateOverloads: false,
        discriminateTypes: false,
        tsNocheck: false,
        dontOverrideCompile: false,
        node16Modules: false,
    };
    return {
        ...defaultConfig,
        ...config.typechain,
    };
}
exports.getDefaultTypechainConfig = getDefaultTypechainConfig;
//# sourceMappingURL=config.js.map