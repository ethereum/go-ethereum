"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.blockscoutConfigExtender = exports.sourcifyConfigExtender = exports.etherscanConfigExtender = void 0;
const picocolors_1 = __importDefault(require("picocolors"));
function etherscanConfigExtender(config, userConfig) {
    const defaultEtherscanConfig = {
        apiKey: "",
        customChains: [],
        enabled: true,
    };
    const cloneDeep = require("lodash.clonedeep");
    const userEtherscanConfig = cloneDeep(userConfig.etherscan);
    config.etherscan = { ...defaultEtherscanConfig, ...userEtherscanConfig };
    // check that there is no etherscan entry in the networks object, since
    // this is a common mistake made by users
    if (userConfig.etherscan === undefined &&
        config.networks?.etherscan !== undefined) {
        console.warn(picocolors_1.default.yellow("WARNING: you have an 'etherscan' entry in your networks configuration. This is likely a mistake. The etherscan configuration should be at the root of the configuration, not within the networks object."));
    }
}
exports.etherscanConfigExtender = etherscanConfigExtender;
function sourcifyConfigExtender(config, userConfig) {
    const defaultSourcifyConfig = {
        enabled: false,
        apiUrl: "https://sourcify.dev/server",
        browserUrl: "https://repo.sourcify.dev",
    };
    const cloneDeep = require("lodash.clonedeep");
    const userSourcifyConfig = cloneDeep(userConfig.sourcify);
    config.sourcify = { ...defaultSourcifyConfig, ...userSourcifyConfig };
}
exports.sourcifyConfigExtender = sourcifyConfigExtender;
function blockscoutConfigExtender(config, userConfig) {
    const defaultBlockscoutConfig = {
        enabled: false,
        customChains: [],
    };
    const cloneDeep = require("lodash.clonedeep");
    const userBlockscoutConfig = cloneDeep(userConfig.blockscout);
    config.blockscout = { ...defaultBlockscoutConfig, ...userBlockscoutConfig };
}
exports.blockscoutConfigExtender = blockscoutConfigExtender;
//# sourceMappingURL=config.js.map