"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Etherscan = exports.ETHERSCAN_V2_API_URL = void 0;
const plugins_1 = require("hardhat/plugins");
const picocolors_1 = __importDefault(require("picocolors"));
const errors_1 = require("./errors");
const undici_1 = require("./undici");
const utilities_1 = require("./utilities");
const chain_config_1 = require("./chain-config");
// Used for polling the result of the contract verification.
const VERIFICATION_STATUS_POLLING_TIME = 3000;
exports.ETHERSCAN_V2_API_URL = "https://api.etherscan.io/v2/api";
/**
 * Etherscan verification provider for verifying smart contracts.
 * It should work with other verification providers as long as the interface
 * is compatible with Etherscan's.
 */
class Etherscan {
    /**
     * Create a new instance of the Etherscan verification provider.
     * @param apiKey - The Etherscan API key.
     * @param apiUrl - The Etherscan API URL, e.g. https://api.etherscan.io/api.
     * @param browserUrl - The Etherscan browser URL, e.g. https://etherscan.io.
     * @param chainId - Chain id when willing to use the v2 api, undefined otherwise
     */
    constructor(apiKey, apiUrl, browserUrl, chainId) {
        this.apiKey = apiKey;
        this.apiUrl = apiUrl;
        this.browserUrl = browserUrl;
        this.chainId = chainId;
        this.apiUrl = chainId === undefined ? apiUrl : exports.ETHERSCAN_V2_API_URL;
    }
    static async getCurrentChainConfig(networkName, ethereumProvider, customChains) {
        const currentChainId = parseInt(await ethereumProvider.send("eth_chainId"), 16);
        const currentChainConfig = [
            // custom chains has higher precedence than builtin chains
            ...[...customChains].reverse(),
            ...chain_config_1.builtinChains,
        ].find(({ chainId }) => chainId === currentChainId);
        if (currentChainConfig === undefined) {
            if (networkName === plugins_1.HARDHAT_NETWORK_NAME) {
                throw new errors_1.HardhatNetworkNotSupportedError();
            }
            throw new errors_1.ChainConfigNotFoundError(currentChainId);
        }
        return currentChainConfig;
    }
    static fromChainConfig(apiKey, chainConfig) {
        const resolvedApiKey = resolveApiKey(apiKey, chainConfig.network);
        const apiUrl = chainConfig.urls.apiURL;
        const browserUrl = chainConfig.urls.browserURL.trim().replace(/\/$/, "");
        // If a user sets a single api key, it means it's etherscan.io key, and we can use the api v2 with multiple chain support
        // If multiple keys are set, it means that l2/sidechain explorers are being used, and those keys don't work with etherscan.io api v2.
        // So we keep using the v1 api of their respective explorers
        const isV2 = typeof apiKey === "string";
        if (!isV2) {
            console.warn(picocolors_1.default.yellow("[WARNING] Network and explorer-specific api keys are deprecated in favour of the new Etherscan v2 api. Support for v1 is expected to end by May 31st, 2025. To migrate, please specify a single Etherscan.io api key the apiKey config value."));
        }
        return new Etherscan(resolvedApiKey, apiUrl, browserUrl, isV2 ? chainConfig.chainId : undefined);
    }
    /**
     * Check if a smart contract is verified on Etherscan.
     * @link https://docs.etherscan.io/api-endpoints/contracts#get-contract-source-code-for-verified-contract-source-codes
     * @param address - The address of the smart contract.
     * @returns True if the contract is verified, false otherwise.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     */
    async isVerified(address) {
        const parameters = new URLSearchParams({
            apikey: this.apiKey,
            module: "contract",
            action: "getsourcecode",
            address,
        });
        if (this.chainId !== undefined) {
            parameters.set("chainid", String(this.chainId));
        }
        const url = new URL(this.apiUrl);
        url.search = parameters.toString();
        let response;
        let json;
        try {
            response = await (0, undici_1.sendGetRequest)(url);
            json = (await response.body.json());
        }
        catch (e) {
            throw new errors_1.NetworkRequestError(e);
        }
        if (!(0, undici_1.isSuccessStatusCode)(response.statusCode)) {
            throw new errors_1.ContractVerificationInvalidStatusCodeError(url.toString(), response.statusCode, JSON.stringify(json));
        }
        if (json.message !== "OK") {
            return false;
        }
        const sourceCode = json.result[0]?.SourceCode;
        return sourceCode !== undefined && sourceCode !== null && sourceCode !== "";
    }
    /**
     * Verify a smart contract on Etherscan.
     * @link https://docs.etherscan.io/api-endpoints/contracts#verify-source-code
     * @param contractAddress - The address of the smart contract to verify.
     * @param sourceCode - The source code of the smart contract.
     * @param contractName - The name of the smart contract, e.g. "contracts/Sample.sol:MyContract"
     * @param compilerVersion - The version of the Solidity compiler used, e.g. `v0.8.19+commit.7dd6d404`
     * @param constructorArguments - The encoded constructor arguments of the smart contract.
     * @returns A promise that resolves to an `EtherscanResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractVerificationInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractVerificationMissingBytecodeError} if the bytecode is not found on the block explorer.
     * @throws {ContractAlreadyVerifiedError} if the contract is already verified.
     * @throws {HardhatVerifyError} if the response status is not OK.
     */
    async verify(contractAddress, sourceCode, contractName, compilerVersion, constructorArguments) {
        const parameters = new URLSearchParams({
            apikey: this.apiKey,
            module: "contract",
            action: "verifysourcecode",
            contractaddress: contractAddress,
            sourceCode,
            codeformat: "solidity-standard-json-input",
            contractname: contractName,
            compilerversion: compilerVersion,
            constructorArguements: constructorArguments,
        });
        const url = new URL(this.apiUrl);
        if (this.chainId !== undefined) {
            url.searchParams.append("chainid", String(this.chainId));
        }
        let response;
        let json;
        try {
            response = await (0, undici_1.sendPostRequest)(url, parameters.toString(), {
                "Content-Type": "application/x-www-form-urlencoded",
            });
            json = (await response.body.json());
        }
        catch (e) {
            throw new errors_1.NetworkRequestError(e);
        }
        if (!(0, undici_1.isSuccessStatusCode)(response.statusCode)) {
            throw new errors_1.ContractVerificationInvalidStatusCodeError(url.toString(), response.statusCode, JSON.stringify(json));
        }
        const etherscanResponse = new EtherscanResponse(json);
        if (etherscanResponse.isBytecodeMissingInNetworkError()) {
            throw new errors_1.ContractVerificationMissingBytecodeError(this.apiUrl, contractAddress);
        }
        if (etherscanResponse.isAlreadyVerified()) {
            throw new errors_1.ContractAlreadyVerifiedError(contractName, contractAddress);
        }
        if (!etherscanResponse.isOk()) {
            throw new errors_1.HardhatVerifyError(etherscanResponse.message);
        }
        return etherscanResponse;
    }
    /**
     * Get the verification status of a smart contract from Etherscan.
     * This method performs polling of the verification status if it's pending.
     * @link https://docs.etherscan.io/api-endpoints/contracts#check-source-code-verification-submission-status
     * @param guid - The verification GUID to check.
     * @returns A promise that resolves to an `EtherscanResponse` object.
     * @throws {NetworkRequestError} if there is an error on the request.
     * @throws {ContractStatusPollingInvalidStatusCodeError} if the API returns an invalid status code.
     * @throws {ContractStatusPollingResponseNotOkError} if the response status is not OK.
     */
    async getVerificationStatus(guid) {
        const parameters = new URLSearchParams({
            apikey: this.apiKey,
            module: "contract",
            action: "checkverifystatus",
            guid,
        });
        if (this.chainId !== undefined) {
            parameters.set("chainid", String(this.chainId));
        }
        const url = new URL(this.apiUrl);
        url.search = parameters.toString();
        let response;
        let json;
        try {
            response = await (0, undici_1.sendGetRequest)(url);
            json = (await response.body.json());
        }
        catch (e) {
            throw new errors_1.NetworkRequestError(e);
        }
        if (!(0, undici_1.isSuccessStatusCode)(response.statusCode)) {
            throw new errors_1.ContractStatusPollingInvalidStatusCodeError(response.statusCode, JSON.stringify(json));
        }
        const etherscanResponse = new EtherscanResponse(json);
        if (etherscanResponse.isPending()) {
            await (0, utilities_1.sleep)(VERIFICATION_STATUS_POLLING_TIME);
            return this.getVerificationStatus(guid);
        }
        if (etherscanResponse.isFailure() ||
            etherscanResponse.isAlreadyVerified()) {
            return etherscanResponse;
        }
        if (!etherscanResponse.isOk()) {
            throw new errors_1.ContractStatusPollingResponseNotOkError(etherscanResponse.message);
        }
        return etherscanResponse;
    }
    /**
     * Get the Etherscan URL for viewing a contract's details.
     * @param address - The address of the smart contract.
     * @returns The URL to view the contract on Etherscan's website.
     */
    getContractUrl(address) {
        return `${this.browserUrl}/address/${address}#code`;
    }
}
exports.Etherscan = Etherscan;
class EtherscanResponse {
    constructor(response) {
        this.status = parseInt(response.status, 10);
        this.message = response.result;
    }
    isPending() {
        return this.message === "Pending in queue";
    }
    isFailure() {
        return this.message === "Fail - Unable to verify";
    }
    isSuccess() {
        return this.message === "Pass - Verified";
    }
    isBytecodeMissingInNetworkError() {
        return this.message.startsWith("Unable to locate ContractCode at");
    }
    isAlreadyVerified() {
        return (
        // returned by blockscout
        this.message.startsWith("Smart-contract already verified") ||
            // returned by etherscan
            this.message.startsWith("Contract source code already verified") ||
            this.message.startsWith("Already Verified"));
    }
    isOk() {
        return this.status === 1;
    }
}
function resolveApiKey(apiKey, network) {
    if (apiKey === undefined || apiKey === "") {
        throw new errors_1.MissingApiKeyError(network);
    }
    if (typeof apiKey === "string") {
        return apiKey;
    }
    const key = apiKey[network];
    if (key === undefined || key === "") {
        throw new errors_1.MissingApiKeyError(network);
    }
    return key;
}
//# sourceMappingURL=etherscan.js.map