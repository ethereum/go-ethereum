"use strict";

import { Block, TransactionReceipt, TransactionResponse } from "@ethersproject/abstract-provider";
import { getAddress, getContractAddress } from "@ethersproject/address";
import { BigNumber } from "@ethersproject/bignumber";
import { hexDataLength, hexDataSlice, hexValue, hexZeroPad, isHexString } from "@ethersproject/bytes";
import { AddressZero } from "@ethersproject/constants";
import { shallowCopy } from "@ethersproject/properties";
import { AccessList, accessListify, parse as parseTransaction } from "@ethersproject/transactions";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export type FormatFunc = (value: any) => any;

export type FormatFuncs = { [ key: string ]: FormatFunc };

export type Formats = {
    transaction: FormatFuncs,
    transactionRequest: FormatFuncs,
    receipt: FormatFuncs,
    receiptLog: FormatFuncs,
    block: FormatFuncs,
    blockWithTransactions: FormatFuncs,
    filter: FormatFuncs,
    filterLog: FormatFuncs,
};

export class Formatter {
    readonly formats: Formats;

    constructor() {
        this.formats = this.getDefaultFormats();
    }

    getDefaultFormats(): Formats {
        const formats: Formats = <Formats>({ });

        const address = this.address.bind(this);
        const bigNumber = this.bigNumber.bind(this);
        const blockTag = this.blockTag.bind(this);
        const data = this.data.bind(this);
        const hash = this.hash.bind(this);
        const hex = this.hex.bind(this);
        const number = this.number.bind(this);
        const type = this.type.bind(this);

        const strictData = (v: any) => { return this.data(v, true); };

        formats.transaction = {
            hash: hash,

            type: type,
            accessList: Formatter.allowNull(this.accessList.bind(this), null),

            blockHash: Formatter.allowNull(hash, null),
            blockNumber: Formatter.allowNull(number, null),
            transactionIndex: Formatter.allowNull(number, null),

            confirmations: Formatter.allowNull(number, null),

            from: address,

            // either (gasPrice) or (maxPriorityFeePerGas + maxFeePerGas)
            // must be set
            gasPrice: Formatter.allowNull(bigNumber),
            maxPriorityFeePerGas: Formatter.allowNull(bigNumber),
            maxFeePerGas: Formatter.allowNull(bigNumber),

            gasLimit: bigNumber,
            to: Formatter.allowNull(address, null),
            value: bigNumber,
            nonce: number,
            data: data,

            r: Formatter.allowNull(this.uint256),
            s: Formatter.allowNull(this.uint256),
            v: Formatter.allowNull(number),

            creates: Formatter.allowNull(address, null),

            raw: Formatter.allowNull(data),
        };

        formats.transactionRequest = {
            from: Formatter.allowNull(address),
            nonce: Formatter.allowNull(number),
            gasLimit: Formatter.allowNull(bigNumber),
            gasPrice: Formatter.allowNull(bigNumber),
            maxPriorityFeePerGas: Formatter.allowNull(bigNumber),
            maxFeePerGas: Formatter.allowNull(bigNumber),
            to: Formatter.allowNull(address),
            value: Formatter.allowNull(bigNumber),
            data: Formatter.allowNull(strictData),
            type: Formatter.allowNull(number),
            accessList: Formatter.allowNull(this.accessList.bind(this), null),
        };

        formats.receiptLog = {
            transactionIndex: number,
            blockNumber: number,
            transactionHash: hash,
            address: address,
            topics: Formatter.arrayOf(hash),
            data: data,
            logIndex: number,
            blockHash: hash,
        };

        formats.receipt = {
            to: Formatter.allowNull(this.address, null),
            from: Formatter.allowNull(this.address, null),
            contractAddress: Formatter.allowNull(address, null),
            transactionIndex: number,
            // should be allowNull(hash), but broken-EIP-658 support is handled in receipt
            root: Formatter.allowNull(hex),
            gasUsed: bigNumber,
            logsBloom: Formatter.allowNull(data),// @TODO: should this be data?
            blockHash: hash,
            transactionHash: hash,
            logs: Formatter.arrayOf(this.receiptLog.bind(this)),
            blockNumber: number,
            confirmations: Formatter.allowNull(number, null),
            cumulativeGasUsed: bigNumber,
            effectiveGasPrice: Formatter.allowNull(bigNumber),
            status: Formatter.allowNull(number),
            type: type
        };

        formats.block = {
            hash: Formatter.allowNull(hash),
            parentHash: hash,
            number: number,

            timestamp: number,
            nonce: Formatter.allowNull(hex),
            difficulty: this.difficulty.bind(this),

            gasLimit: bigNumber,
            gasUsed: bigNumber,

            miner: Formatter.allowNull(address),
            extraData: data,

            transactions: Formatter.allowNull(Formatter.arrayOf(hash)),

            baseFeePerGas: Formatter.allowNull(bigNumber)
        };

        formats.blockWithTransactions = shallowCopy(formats.block);
        formats.blockWithTransactions.transactions = Formatter.allowNull(Formatter.arrayOf(this.transactionResponse.bind(this)));

        formats.filter = {
            fromBlock: Formatter.allowNull(blockTag, undefined),
            toBlock: Formatter.allowNull(blockTag, undefined),
            blockHash: Formatter.allowNull(hash, undefined),
            address: Formatter.allowNull(address, undefined),
            topics: Formatter.allowNull(this.topics.bind(this), undefined),
        };

        formats.filterLog = {
            blockNumber: Formatter.allowNull(number),
            blockHash: Formatter.allowNull(hash),
            transactionIndex: number,

            removed: Formatter.allowNull(this.boolean.bind(this)),

            address: address,
            data: Formatter.allowFalsish(data, "0x"),

            topics: Formatter.arrayOf(hash),

            transactionHash: hash,
            logIndex: number,
        };

        return formats;
    }

    accessList(accessList: Array<any>): AccessList {
        return accessListify(accessList || []);
    }

    // Requires a BigNumberish that is within the IEEE754 safe integer range; returns a number
    // Strict! Used on input.
    number(number: any): number {
        if (number === "0x") { return 0; }
        return BigNumber.from(number).toNumber();
    }

    type(number: any): number {
        if (number === "0x" || number == null) { return 0; }
        return BigNumber.from(number).toNumber();
    }

    // Strict! Used on input.
    bigNumber(value: any): BigNumber {
        return BigNumber.from(value);
    }

    // Requires a boolean, "true" or  "false"; returns a boolean
    boolean(value: any): boolean {
        if (typeof(value) === "boolean") { return value; }
        if (typeof(value) === "string") {
            value = value.toLowerCase();
            if (value === "true") { return true; }
            if (value === "false") { return false; }
        }
        throw new Error("invalid boolean - " + value);
    }

    hex(value: any, strict?: boolean): string {
        if (typeof(value) === "string") {
            if (!strict && value.substring(0, 2) !== "0x") { value = "0x" + value; }
            if (isHexString(value)) {
               return value.toLowerCase();
            }
        }
        return logger.throwArgumentError("invalid hash", "value", value);
    }

    data(value: any, strict?: boolean): string {
        const result = this.hex(value, strict);
        if ((result.length % 2) !== 0) {
            throw new Error("invalid data; odd-length - " + value);
        }
        return result;
    }

    // Requires an address
    // Strict! Used on input.
    address(value: any): string {
        return getAddress(value);
    }

    callAddress(value: any): string {
        if (!isHexString(value, 32)) { return null; }
        const address = getAddress(hexDataSlice(value, 12));
        return (address === AddressZero) ? null: address;
    }

    contractAddress(value: any): string {
        return getContractAddress(value);
    }

    // Strict! Used on input.
    blockTag(blockTag: any): string {
        if (blockTag == null) { return "latest"; }

        if (blockTag === "earliest") { return "0x0"; }

        switch (blockTag) {
            case "earliest": return "0x0";
            case "latest": case "pending": case "safe": case "finalized":
                return blockTag;
        }

        if (typeof(blockTag) === "number" || isHexString(blockTag)) {
            return hexValue(<number | string>blockTag);
        }

        throw new Error("invalid blockTag");
    }

    // Requires a hash, optionally requires 0x prefix; returns prefixed lowercase hash.
    hash(value: any, strict?: boolean): string {
        const result = this.hex(value, strict);
        if (hexDataLength(result) !== 32) {
            return logger.throwArgumentError("invalid hash", "value", value);
        }
        return result;
    }

    // Returns the difficulty as a number, or if too large (i.e. PoA network) null
    difficulty(value: any): number {
        if (value == null) { return null; }

        const v = BigNumber.from(value);

        try {
            return v.toNumber();
        } catch (error) { }

       return null;
    }

    uint256(value: any): string {
        if (!isHexString(value)) {
            throw new Error("invalid uint256");
        }
        return hexZeroPad(value, 32);
    }

    _block(value: any, format: any): Block {
        if (value.author != null && value.miner == null) {
            value.miner = value.author;
        }
        // The difficulty may need to come from _difficulty in recursed blocks
        const difficulty = (value._difficulty != null) ? value._difficulty: value.difficulty;
        const result = Formatter.check(format, value);
        result._difficulty = ((difficulty == null) ? null: BigNumber.from(difficulty));
        return result;
    }

    block(value: any): Block {
        return this._block(value, this.formats.block);
    }

    blockWithTransactions(value: any): Block {
        return this._block(value, this.formats.blockWithTransactions);
    }

    // Strict! Used on input.
    transactionRequest(value: any): any {
        return Formatter.check(this.formats.transactionRequest, value);
    }

    transactionResponse(transaction: any): TransactionResponse {

        // Rename gas to gasLimit
        if (transaction.gas != null && transaction.gasLimit == null) {
            transaction.gasLimit = transaction.gas;
        }

        // Some clients (TestRPC) do strange things like return 0x0 for the
        // 0 address; correct this to be a real address
        if (transaction.to && BigNumber.from(transaction.to).isZero()) {
            transaction.to = "0x0000000000000000000000000000000000000000";
        }

        // Rename input to data
        if (transaction.input != null && transaction.data == null) {
            transaction.data = transaction.input;
        }

        // If to and creates are empty, populate the creates from the transaction
        if (transaction.to == null && transaction.creates == null) {
            transaction.creates = this.contractAddress(transaction);
        }

        if ((transaction.type === 1 || transaction.type === 2)&& transaction.accessList == null) {
            transaction.accessList = [ ];
        }

        const result: TransactionResponse = Formatter.check(this.formats.transaction, transaction);

        if (transaction.chainId != null) {
            let chainId = transaction.chainId;

            if (isHexString(chainId)) {
                chainId = BigNumber.from(chainId).toNumber();
            }

            result.chainId = chainId;

        } else {
            let chainId = transaction.networkId;

            // geth-etc returns chainId
            if (chainId == null && result.v == null) {
                chainId = transaction.chainId;
            }

            if (isHexString(chainId)) {
                chainId = BigNumber.from(chainId).toNumber();
            }

            if (typeof(chainId) !== "number" && result.v != null) {
                chainId = (result.v - 35) / 2;
                if (chainId < 0) { chainId = 0; }
                chainId = parseInt(chainId);
            }

            if (typeof(chainId) !== "number") { chainId = 0; }

            result.chainId = chainId;
        }

        // 0x0000... should actually be null
        if (result.blockHash && result.blockHash.replace(/0/g, "") === "x") {
            result.blockHash = null;
        }

        return result;
    }

    transaction(value: any): any {
        return parseTransaction(value);
    }

    receiptLog(value: any): any {
        return Formatter.check(this.formats.receiptLog, value);
    }

    receipt(value: any): TransactionReceipt {
        const result: TransactionReceipt = Formatter.check(this.formats.receipt, value);

        // RSK incorrectly implemented EIP-658, so we munge things a bit here for it
        if (result.root != null) {
            if (result.root.length <= 4) {
                // Could be 0x00, 0x0, 0x01 or 0x1
                const value = BigNumber.from(result.root).toNumber();
                if (value === 0 || value === 1) {
                    // Make sure if both are specified, they match
                    if (result.status != null && (result.status !== value)) {
                        logger.throwArgumentError("alt-root-status/status mismatch", "value", { root: result.root, status: result.status });
                    }
                    result.status = value;
                    delete result.root;
                } else {
                    logger.throwArgumentError("invalid alt-root-status", "value.root", result.root);
                }
            } else if (result.root.length !== 66) {
                // Must be a valid bytes32
                logger.throwArgumentError("invalid root hash", "value.root", result.root);
            }
        }

        if (result.status != null) {
            result.byzantium = true;
        }

        return result;
    }

    topics(value: any): any {
        if (Array.isArray(value)) {
            return value.map((v) => this.topics(v));

        } else if (value != null) {
            return this.hash(value, true);
        }

        return null;
    }

    filter(value: any): any {
        return Formatter.check(this.formats.filter, value);
    }

    filterLog(value: any): any {
        return Formatter.check(this.formats.filterLog, value);
    }

    static check(format: { [ name: string ]: FormatFunc }, object: any): any {
        const result: any = {};
        for (const key in format) {
            try {
                const value = format[key](object[key]);
                if (value !== undefined) { result[key] = value; }
            } catch (error) {
                error.checkKey = key;
                error.checkValue = object[key];
                throw error;
            }
        }
        return result;
    }

    // if value is null-ish, nullValue is returned
    static allowNull(format: FormatFunc, nullValue?: any): FormatFunc {
        return (function(value: any) {
            if (value == null) { return nullValue; }
            return format(value);
        });
    }

    // If value is false-ish, replaceValue is returned
    static allowFalsish(format: FormatFunc, replaceValue: any): FormatFunc {
        return (function(value: any) {
            if (!value) { return replaceValue; }
            return format(value);
        });
    }

    // Requires an Array satisfying check
    static arrayOf(format: FormatFunc): FormatFunc {
        return (function(array: any): Array<any> {
            if (!Array.isArray(array)) { throw new Error("not an array"); }

            const result: any = [];

            array.forEach(function(value) {
                result.push(format(value));
            });

            return result;
        });
    }
}

export interface CommunityResourcable {
    isCommunityResource(): boolean;
}

export function isCommunityResourcable(value: any): value is CommunityResourcable {
    return (value && typeof(value.isCommunityResource) === "function");
}

export function isCommunityResource(value: any): boolean {
    return (isCommunityResourcable(value) && value.isCommunityResource());
}

// Show the throttle message only once
let throttleMessage = false;
export function showThrottleMessage() {
    if (throttleMessage) { return; }
    throttleMessage = true;

    console.log("========= NOTICE =========")
    console.log("Request-Rate Exceeded  (this message will not be repeated)");
    console.log("");
    console.log("The default API keys for each service are provided as a highly-throttled,");
    console.log("community resource for low-traffic projects and early prototyping.");
    console.log("");
    console.log("While your application will continue to function, we highly recommended");
    console.log("signing up for your own API keys to improve performance, increase your");
    console.log("request rate/limit and enable other perks, such as metrics and advanced APIs.");
    console.log("");
    console.log("For more details: https:/\/docs.ethers.io/api-keys/");
    console.log("==========================");
}

