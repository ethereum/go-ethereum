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
exports.SignerWithAddress = exports.HardhatEthersSigner = void 0;
const ethers_1 = require("ethers");
const ethers_utils_1 = require("./internal/ethers-utils");
const errors_1 = require("./internal/errors");
class HardhatEthersSigner {
    static async create(provider, address) {
        const hre = await Promise.resolve().then(() => __importStar(require("hardhat")));
        // depending on the config, we set a fixed gas limit for all transactions
        let gasLimit;
        if (hre.network.name === "hardhat") {
            // If we are connected to the in-process hardhat network and the config
            // has a fixed number as the gas config, we use that.
            // Hardhat core already sets this value to the block gas limit when the
            // user doesn't specify a number.
            if (hre.network.config.gas !== "auto") {
                gasLimit = hre.network.config.gas;
            }
        }
        else if (hre.network.name === "localhost") {
            const configuredGasLimit = hre.config.networks.localhost.gas;
            if (configuredGasLimit !== "auto") {
                // if the resolved gas config is a number, we use that
                gasLimit = configuredGasLimit;
            }
            else {
                // if the resolved gas config is "auto", we need to check that
                // the user config is undefined, because that's the default value;
                // otherwise explicitly setting the gas to "auto" would have no effect
                if (hre.userConfig.networks?.localhost?.gas === undefined) {
                    // finally, we check if we are connected to a hardhat network
                    let isHardhatNetwork = false;
                    try {
                        await hre.network.provider.send("hardhat_metadata");
                        isHardhatNetwork = true;
                    }
                    catch { }
                    if (isHardhatNetwork) {
                        // WARNING: this assumes that the hardhat node is being run in the
                        // same project which might be wrong
                        gasLimit = hre.config.networks.hardhat.blockGasLimit;
                    }
                }
            }
        }
        return new HardhatEthersSigner(address, provider, gasLimit);
    }
    constructor(address, _provider, _gasLimit) {
        this._gasLimit = _gasLimit;
        this.address = (0, ethers_1.getAddress)(address);
        this.provider = _provider;
    }
    connect(provider) {
        return new HardhatEthersSigner(this.address, provider);
    }
    getNonce(blockTag) {
        return this.provider.getTransactionCount(this.address, blockTag);
    }
    populateCall(tx) {
        return populate(this, tx);
    }
    populateTransaction(tx) {
        return this.populateCall(tx);
    }
    async estimateGas(tx) {
        return this.provider.estimateGas(await this.populateCall(tx));
    }
    async call(tx) {
        return this.provider.call(await this.populateCall(tx));
    }
    resolveName(name) {
        return this.provider.resolveName(name);
    }
    async signTransaction(_tx) {
        // TODO if we split the signer for the in-process and json-rpc networks,
        // we can enable this method when using the in-process network or when the
        // json-rpc network has a private key
        throw new errors_1.NotImplementedError("HardhatEthersSigner.signTransaction");
    }
    async sendTransaction(tx) {
        // This cannot be mined any earlier than any recent block
        const blockNumber = await this.provider.getBlockNumber();
        // Send the transaction
        const hash = await this._sendUncheckedTransaction(tx);
        // Unfortunately, JSON-RPC only provides and opaque transaction hash
        // for a response, and we need the actual transaction, so we poll
        // for it; it should show up very quickly
        return new Promise((resolve) => {
            const timeouts = [1000, 100];
            const checkTx = async () => {
                // Try getting the transaction
                const txPolled = await this.provider.getTransaction(hash);
                if (txPolled !== null) {
                    resolve(txPolled.replaceableTransaction(blockNumber));
                    return;
                }
                // Wait another 4 seconds
                setTimeout(() => {
                    // eslint-disable-next-line @typescript-eslint/no-floating-promises
                    checkTx();
                }, timeouts.pop() ?? 4000);
            };
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            checkTx();
        });
    }
    signMessage(message) {
        const resolvedMessage = typeof message === "string" ? (0, ethers_1.toUtf8Bytes)(message) : message;
        return this.provider.send("personal_sign", [
            (0, ethers_1.hexlify)(resolvedMessage),
            this.address.toLowerCase(),
        ]);
    }
    async signTypedData(domain, types, value) {
        const copiedValue = deepCopy(value);
        // Populate any ENS names (in-place)
        const populated = await ethers_1.TypedDataEncoder.resolveNames(domain, types, copiedValue, async (v) => {
            return v;
        });
        return this.provider.send("eth_signTypedData_v4", [
            this.address.toLowerCase(),
            JSON.stringify(ethers_1.TypedDataEncoder.getPayload(populated.domain, types, populated.value), (_k, v) => {
                if (typeof v === "bigint") {
                    return v.toString();
                }
                return v;
            }),
        ]);
    }
    async getAddress() {
        return this.address;
    }
    toJSON() {
        return `<SignerWithAddress ${this.address}>`;
    }
    async _sendUncheckedTransaction(tx) {
        const resolvedTx = deepCopy(tx);
        const promises = [];
        // Make sure the from matches the sender
        if (resolvedTx.from !== null && resolvedTx.from !== undefined) {
            const _from = resolvedTx.from;
            promises.push((async () => {
                const from = await (0, ethers_1.resolveAddress)(_from, this.provider);
                (0, ethers_1.assertArgument)(from !== null &&
                    from !== undefined &&
                    from.toLowerCase() === this.address.toLowerCase(), "from address mismatch", "transaction", tx);
                resolvedTx.from = from;
            })());
        }
        else {
            resolvedTx.from = this.address;
        }
        if (resolvedTx.gasLimit === null || resolvedTx.gasLimit === undefined) {
            if (this._gasLimit !== undefined) {
                resolvedTx.gasLimit = this._gasLimit;
            }
            else {
                promises.push((async () => {
                    resolvedTx.gasLimit = await this.provider.estimateGas({
                        ...resolvedTx,
                        from: this.address,
                    });
                })());
            }
        }
        // The address may be an ENS name or Addressable
        if (resolvedTx.to !== null && resolvedTx.to !== undefined) {
            const _to = resolvedTx.to;
            promises.push((async () => {
                resolvedTx.to = await (0, ethers_1.resolveAddress)(_to, this.provider);
            })());
        }
        // Wait until all of our properties are filled in
        if (promises.length > 0) {
            await Promise.all(promises);
        }
        const hexTx = (0, ethers_utils_1.getRpcTransaction)(resolvedTx);
        return this.provider.send("eth_sendTransaction", [hexTx]);
    }
}
exports.HardhatEthersSigner = HardhatEthersSigner;
exports.SignerWithAddress = HardhatEthersSigner;
async function populate(signer, tx) {
    const pop = (0, ethers_utils_1.copyRequest)(tx);
    if (pop.to !== null && pop.to !== undefined) {
        pop.to = (0, ethers_1.resolveAddress)(pop.to, signer);
    }
    if (pop.from !== null && pop.from !== undefined) {
        const from = pop.from;
        pop.from = Promise.all([
            signer.getAddress(),
            (0, ethers_1.resolveAddress)(from, signer),
        ]).then(([address, resolvedFrom]) => {
            (0, ethers_1.assertArgument)(address.toLowerCase() === resolvedFrom.toLowerCase(), "transaction from mismatch", "tx.from", resolvedFrom);
            return address;
        });
    }
    else {
        pop.from = signer.getAddress();
    }
    return (0, ethers_utils_1.resolveProperties)(pop);
}
const Primitive = "bigint,boolean,function,number,string,symbol".split(/,/g);
function deepCopy(value) {
    if (value === null ||
        value === undefined ||
        Primitive.indexOf(typeof value) >= 0) {
        return value;
    }
    // Keep any Addressable
    if (typeof value.getAddress === "function") {
        return value;
    }
    if (Array.isArray(value)) {
        return value.map(deepCopy);
    }
    if (typeof value === "object") {
        return Object.keys(value).reduce((accum, key) => {
            accum[key] = value[key];
            return accum;
        }, {});
    }
    throw new errors_1.HardhatEthersError(`Assertion error: ${value} (${typeof value})`);
}
//# sourceMappingURL=signers.js.map