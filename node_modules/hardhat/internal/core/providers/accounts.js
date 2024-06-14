"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.FixedSenderProvider = exports.AutomaticSenderProvider = exports.HDWalletProvider = exports.LocalAccountsProvider = void 0;
const t = __importStar(require("io-ts"));
const eth_sig_util_1 = require("@metamask/eth-sig-util");
const ethereumjs_tx_1 = require("@nomicfoundation/ethereumjs-tx");
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
const base_types_1 = require("../jsonrpc/types/base-types");
const transactionRequest_1 = require("../jsonrpc/types/input/transactionRequest");
const validation_1 = require("../jsonrpc/types/input/validation");
const chainId_1 = require("./chainId");
const util_1 = require("./util");
const wrapper_1 = require("./wrapper");
class LocalAccountsProvider extends chainId_1.ProviderWrapperWithChainId {
    constructor(provider, localAccountsHexPrivateKeys) {
        super(provider);
        this._addressToPrivateKey = new Map();
        this._initializePrivateKeys(localAccountsHexPrivateKeys);
    }
    async request(args) {
        const { ecsign, hashPersonalMessage, toRpcSig, toBytes, bytesToHex: bufferToHex, } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ethereumjs-util")));
        if (args.method === "eth_accounts" ||
            args.method === "eth_requestAccounts") {
            return [...this._addressToPrivateKey.keys()];
        }
        const params = this._getParams(args);
        if (args.method === "eth_sign") {
            if (params.length > 0) {
                const [address, data] = (0, validation_1.validateParams)(params, base_types_1.rpcAddress, base_types_1.rpcData);
                if (address !== undefined) {
                    if (data === undefined) {
                        throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.ETHSIGN_MISSING_DATA_PARAM);
                    }
                    const privateKey = this._getPrivateKeyForAddress(address);
                    const messageHash = hashPersonalMessage(toBytes(data));
                    const signature = ecsign(messageHash, privateKey);
                    return toRpcSig(signature.v, signature.r, signature.s);
                }
            }
        }
        if (args.method === "personal_sign") {
            if (params.length > 0) {
                const [data, address] = (0, validation_1.validateParams)(params, base_types_1.rpcData, base_types_1.rpcAddress);
                if (data !== undefined) {
                    if (address === undefined) {
                        throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.PERSONALSIGN_MISSING_ADDRESS_PARAM);
                    }
                    const privateKey = this._getPrivateKeyForAddress(address);
                    const messageHash = hashPersonalMessage(toBytes(data));
                    const signature = ecsign(messageHash, privateKey);
                    return toRpcSig(signature.v, signature.r, signature.s);
                }
            }
        }
        if (args.method === "eth_signTypedData_v4") {
            const [address, data] = (0, validation_1.validateParams)(params, base_types_1.rpcAddress, t.any);
            if (data === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.ETHSIGN_MISSING_DATA_PARAM);
            }
            let typedMessage = data;
            if (typeof data === "string") {
                try {
                    typedMessage = JSON.parse(data);
                }
                catch {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.ETHSIGN_TYPED_DATA_V4_INVALID_DATA_PARAM);
                }
            }
            // if we don't manage the address, the method is forwarded
            const privateKey = this._getPrivateKeyForAddressOrNull(address);
            if (privateKey !== null) {
                return (0, eth_sig_util_1.signTypedData)({
                    privateKey,
                    version: eth_sig_util_1.SignTypedDataVersion.V4,
                    data: typedMessage,
                });
            }
        }
        if (args.method === "eth_sendTransaction" && params.length > 0) {
            const [txRequest] = (0, validation_1.validateParams)(params, transactionRequest_1.rpcTransactionRequest);
            if (txRequest.gas === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY, { param: "gas" });
            }
            if (txRequest.from === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY, { param: "from" });
            }
            const hasGasPrice = txRequest.gasPrice !== undefined;
            const hasEip1559Fields = txRequest.maxFeePerGas !== undefined ||
                txRequest.maxPriorityFeePerGas !== undefined;
            if (!hasGasPrice && !hasEip1559Fields) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.MISSING_FEE_PRICE_FIELDS);
            }
            if (hasGasPrice && hasEip1559Fields) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.INCOMPATIBLE_FEE_PRICE_FIELDS);
            }
            if (hasEip1559Fields && txRequest.maxFeePerGas === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY, { param: "maxFeePerGas" });
            }
            if (hasEip1559Fields && txRequest.maxPriorityFeePerGas === undefined) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.MISSING_TX_PARAM_TO_SIGN_LOCALLY, { param: "maxPriorityFeePerGas" });
            }
            if (txRequest.nonce === undefined) {
                txRequest.nonce = await this._getNonce(txRequest.from);
            }
            const privateKey = this._getPrivateKeyForAddress(txRequest.from);
            const chainId = await this._getChainId();
            const rawTransaction = await this._getSignedTransaction(txRequest, chainId, privateKey);
            return this._wrappedProvider.request({
                method: "eth_sendRawTransaction",
                params: [bufferToHex(rawTransaction)],
            });
        }
        return this._wrappedProvider.request(args);
    }
    _initializePrivateKeys(localAccountsHexPrivateKeys) {
        const { bytesToHex: bufferToHex, toBytes, privateToAddress, } = require("@nomicfoundation/ethereumjs-util");
        const privateKeys = localAccountsHexPrivateKeys.map((h) => toBytes(h));
        for (const pk of privateKeys) {
            const address = bufferToHex(privateToAddress(pk)).toLowerCase();
            this._addressToPrivateKey.set(address, pk);
        }
    }
    _getPrivateKeyForAddress(address) {
        const { bytesToHex: bufferToHex, } = require("@nomicfoundation/ethereumjs-util");
        const pk = this._addressToPrivateKey.get(bufferToHex(address));
        if (pk === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.NOT_LOCAL_ACCOUNT, {
                account: bufferToHex(address),
            });
        }
        return pk;
    }
    _getPrivateKeyForAddressOrNull(address) {
        try {
            return this._getPrivateKeyForAddress(address);
        }
        catch {
            return null;
        }
    }
    async _getNonce(address) {
        const { bytesToHex: bufferToHex } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ethereumjs-util")));
        const response = (await this._wrappedProvider.request({
            method: "eth_getTransactionCount",
            params: [bufferToHex(address), "pending"],
        }));
        return (0, base_types_1.rpcQuantityToBigInt)(response);
    }
    async _getSignedTransaction(transactionRequest, chainId, privateKey) {
        const { AccessListEIP2930Transaction, LegacyTransaction } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ethereumjs-tx")));
        const { Common } = await Promise.resolve().then(() => __importStar(require("@nomicfoundation/ethereumjs-common")));
        const txData = {
            ...transactionRequest,
            gasLimit: transactionRequest.gas,
        };
        // We don't specify a hardfork here because the default hardfork should
        // support all possible types of transactions.
        // If the network doesn't support a given transaction type, then the
        // transaction it will be rejected somewhere else.
        const common = Common.custom({ chainId, networkId: chainId });
        // we convert the access list to the type
        // that AccessListEIP2930Transaction expects
        const accessList = txData.accessList?.map(({ address, storageKeys }) => [address, storageKeys]);
        let transaction;
        if (txData.maxFeePerGas !== undefined) {
            transaction = ethereumjs_tx_1.FeeMarketEIP1559Transaction.fromTxData({
                ...txData,
                accessList,
                gasPrice: undefined,
            }, { common });
        }
        else if (accessList !== undefined) {
            transaction = AccessListEIP2930Transaction.fromTxData({
                ...txData,
                accessList,
            }, { common });
        }
        else {
            transaction = LegacyTransaction.fromTxData(txData, { common });
        }
        const signedTransaction = transaction.sign(privateKey);
        return signedTransaction.serialize();
    }
}
exports.LocalAccountsProvider = LocalAccountsProvider;
class HDWalletProvider extends LocalAccountsProvider {
    constructor(provider, mnemonic, hdpath = "m/44'/60'/0'/0/", initialIndex = 0, count = 10, passphrase = "") {
        // NOTE: If mnemonic has space or newline at the beginning or end, it will be trimmed.
        // This is because mnemonic containing them may generate different private keys.
        const trimmedMnemonic = mnemonic.trim();
        const privateKeys = (0, util_1.derivePrivateKeys)(trimmedMnemonic, hdpath, initialIndex, count, passphrase);
        const { bytesToHex: bufferToHex, } = require("@nomicfoundation/ethereumjs-util");
        const privateKeysAsHex = privateKeys.map((pk) => bufferToHex(pk));
        super(provider, privateKeysAsHex);
    }
}
exports.HDWalletProvider = HDWalletProvider;
class SenderProvider extends wrapper_1.ProviderWrapper {
    async request(args) {
        const method = args.method;
        const params = this._getParams(args);
        if (method === "eth_sendTransaction" ||
            method === "eth_call" ||
            method === "eth_estimateGas") {
            // TODO: Should we validate this type?
            const tx = params[0];
            if (tx !== undefined && tx.from === undefined) {
                const senderAccount = await this._getSender();
                if (senderAccount !== undefined) {
                    tx.from = senderAccount;
                }
                else if (method === "eth_sendTransaction") {
                    throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.NO_REMOTE_ACCOUNT_AVAILABLE);
                }
            }
        }
        return this._wrappedProvider.request(args);
    }
}
class AutomaticSenderProvider extends SenderProvider {
    async _getSender() {
        if (this._firstAccount === undefined) {
            const accounts = (await this._wrappedProvider.request({
                method: "eth_accounts",
            }));
            this._firstAccount = accounts[0];
        }
        return this._firstAccount;
    }
}
exports.AutomaticSenderProvider = AutomaticSenderProvider;
class FixedSenderProvider extends SenderProvider {
    constructor(provider, _sender) {
        super(provider);
        this._sender = _sender;
    }
    async _getSender() {
        return this._sender;
    }
}
exports.FixedSenderProvider = FixedSenderProvider;
//# sourceMappingURL=accounts.js.map