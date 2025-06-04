"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.CLRequestFactory = exports.ConsolidationRequest = exports.WithdrawalRequest = exports.DepositRequest = exports.CLRequest = exports.CLRequestType = void 0;
const rlp_1 = require("@ethereumjs/rlp");
const utils_1 = require("ethereum-cryptography/utils");
const bytes_js_1 = require("./bytes.js");
const constants_js_1 = require("./constants.js");
var CLRequestType;
(function (CLRequestType) {
    CLRequestType[CLRequestType["Deposit"] = 0] = "Deposit";
    CLRequestType[CLRequestType["Withdrawal"] = 1] = "Withdrawal";
    CLRequestType[CLRequestType["Consolidation"] = 2] = "Consolidation";
})(CLRequestType = exports.CLRequestType || (exports.CLRequestType = {}));
class CLRequest {
    constructor(type) {
        this.type = type;
    }
}
exports.CLRequest = CLRequest;
class DepositRequest extends CLRequest {
    constructor(pubkey, withdrawalCredentials, amount, signature, index) {
        super(CLRequestType.Deposit);
        this.pubkey = pubkey;
        this.withdrawalCredentials = withdrawalCredentials;
        this.amount = amount;
        this.signature = signature;
        this.index = index;
    }
    static fromRequestData(depositData) {
        const { pubkey, withdrawalCredentials, amount, signature, index } = depositData;
        return new DepositRequest(pubkey, withdrawalCredentials, amount, signature, index);
    }
    static fromJSON(jsonData) {
        const { pubkey, withdrawalCredentials, amount, signature, index } = jsonData;
        return this.fromRequestData({
            pubkey: (0, bytes_js_1.hexToBytes)(pubkey),
            withdrawalCredentials: (0, bytes_js_1.hexToBytes)(withdrawalCredentials),
            amount: (0, bytes_js_1.hexToBigInt)(amount),
            signature: (0, bytes_js_1.hexToBytes)(signature),
            index: (0, bytes_js_1.hexToBigInt)(index),
        });
    }
    serialize() {
        const indexBytes = this.index === constants_js_1.BIGINT_0 ? new Uint8Array() : (0, bytes_js_1.bigIntToBytes)(this.index);
        const amountBytes = this.amount === constants_js_1.BIGINT_0 ? new Uint8Array() : (0, bytes_js_1.bigIntToBytes)(this.amount);
        return (0, utils_1.concatBytes)(Uint8Array.from([this.type]), rlp_1.RLP.encode([this.pubkey, this.withdrawalCredentials, amountBytes, this.signature, indexBytes]));
    }
    toJSON() {
        return {
            pubkey: (0, bytes_js_1.bytesToHex)(this.pubkey),
            withdrawalCredentials: (0, bytes_js_1.bytesToHex)(this.withdrawalCredentials),
            amount: (0, bytes_js_1.bigIntToHex)(this.amount),
            signature: (0, bytes_js_1.bytesToHex)(this.signature),
            index: (0, bytes_js_1.bigIntToHex)(this.index),
        };
    }
    static deserialize(bytes) {
        const [pubkey, withdrawalCredentials, amount, signature, index] = rlp_1.RLP.decode(bytes.slice(1));
        return this.fromRequestData({
            pubkey,
            withdrawalCredentials,
            amount: (0, bytes_js_1.bytesToBigInt)(amount),
            signature,
            index: (0, bytes_js_1.bytesToBigInt)(index),
        });
    }
}
exports.DepositRequest = DepositRequest;
class WithdrawalRequest extends CLRequest {
    constructor(sourceAddress, validatorPubkey, amount) {
        super(CLRequestType.Withdrawal);
        this.sourceAddress = sourceAddress;
        this.validatorPubkey = validatorPubkey;
        this.amount = amount;
    }
    static fromRequestData(withdrawalData) {
        const { sourceAddress, validatorPubkey, amount } = withdrawalData;
        return new WithdrawalRequest(sourceAddress, validatorPubkey, amount);
    }
    static fromJSON(jsonData) {
        const { sourceAddress, validatorPubkey, amount } = jsonData;
        return this.fromRequestData({
            sourceAddress: (0, bytes_js_1.hexToBytes)(sourceAddress),
            validatorPubkey: (0, bytes_js_1.hexToBytes)(validatorPubkey),
            amount: (0, bytes_js_1.hexToBigInt)(amount),
        });
    }
    serialize() {
        const amountBytes = this.amount === constants_js_1.BIGINT_0 ? new Uint8Array() : (0, bytes_js_1.bigIntToBytes)(this.amount);
        return (0, utils_1.concatBytes)(Uint8Array.from([this.type]), rlp_1.RLP.encode([this.sourceAddress, this.validatorPubkey, amountBytes]));
    }
    toJSON() {
        return {
            sourceAddress: (0, bytes_js_1.bytesToHex)(this.sourceAddress),
            validatorPubkey: (0, bytes_js_1.bytesToHex)(this.validatorPubkey),
            amount: (0, bytes_js_1.bigIntToHex)(this.amount),
        };
    }
    static deserialize(bytes) {
        const [sourceAddress, validatorPubkey, amount] = rlp_1.RLP.decode(bytes.slice(1));
        return this.fromRequestData({
            sourceAddress,
            validatorPubkey,
            amount: (0, bytes_js_1.bytesToBigInt)(amount),
        });
    }
}
exports.WithdrawalRequest = WithdrawalRequest;
class ConsolidationRequest extends CLRequest {
    constructor(sourceAddress, sourcePubkey, targetPubkey) {
        super(CLRequestType.Consolidation);
        this.sourceAddress = sourceAddress;
        this.sourcePubkey = sourcePubkey;
        this.targetPubkey = targetPubkey;
    }
    static fromRequestData(consolidationData) {
        const { sourceAddress, sourcePubkey, targetPubkey } = consolidationData;
        return new ConsolidationRequest(sourceAddress, sourcePubkey, targetPubkey);
    }
    static fromJSON(jsonData) {
        const { sourceAddress, sourcePubkey, targetPubkey } = jsonData;
        return this.fromRequestData({
            sourceAddress: (0, bytes_js_1.hexToBytes)(sourceAddress),
            sourcePubkey: (0, bytes_js_1.hexToBytes)(sourcePubkey),
            targetPubkey: (0, bytes_js_1.hexToBytes)(targetPubkey),
        });
    }
    serialize() {
        return (0, utils_1.concatBytes)(Uint8Array.from([this.type]), rlp_1.RLP.encode([this.sourceAddress, this.sourcePubkey, this.targetPubkey]));
    }
    toJSON() {
        return {
            sourceAddress: (0, bytes_js_1.bytesToHex)(this.sourceAddress),
            sourcePubkey: (0, bytes_js_1.bytesToHex)(this.sourcePubkey),
            targetPubkey: (0, bytes_js_1.bytesToHex)(this.targetPubkey),
        };
    }
    static deserialize(bytes) {
        const [sourceAddress, sourcePubkey, targetPubkey] = rlp_1.RLP.decode(bytes.slice(1));
        return this.fromRequestData({
            sourceAddress,
            sourcePubkey,
            targetPubkey,
        });
    }
}
exports.ConsolidationRequest = ConsolidationRequest;
class CLRequestFactory {
    static fromSerializedRequest(bytes) {
        switch (bytes[0]) {
            case CLRequestType.Deposit:
                return DepositRequest.deserialize(bytes);
            case CLRequestType.Withdrawal:
                return WithdrawalRequest.deserialize(bytes);
            case CLRequestType.Consolidation:
                return ConsolidationRequest.deserialize(bytes);
            default:
                throw Error(`Invalid request type=${bytes[0]}`);
        }
    }
}
exports.CLRequestFactory = CLRequestFactory;
//# sourceMappingURL=requests.js.map