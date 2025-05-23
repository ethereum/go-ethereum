"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getApiKeyAndUrls = void 0;
const plugins_1 = require("hardhat/plugins");
function getApiKeyAndUrls(etherscanApiKey, chainConfig) {
    const apiKey = typeof etherscanApiKey === "string"
        ? etherscanApiKey
        : etherscanApiKey[chainConfig.network];
    if (apiKey === undefined) {
        throw new plugins_1.NomicLabsHardhatPluginError("@nomicfoundation/hardhat-ignition", `No etherscan API key configured for network ${chainConfig.network}`);
    }
    return [apiKey, chainConfig.urls.apiURL, chainConfig.urls.browserURL];
}
exports.getApiKeyAndUrls = getApiKeyAndUrls;
//# sourceMappingURL=getApiKeyAndUrls.js.map