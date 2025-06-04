"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Withdrawal = void 0;
const address_1 = require("./address");
const bytes_1 = require("./bytes");
const types_1 = require("./types");
/**
 * Representation of EIP-4895 withdrawal data
 */
class Withdrawal {
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating a Withdrawal object from varying data types.
     * Its amount is in Gwei to match CL representation and for eventual ssz withdrawalsRoot
     */
    constructor(index, validatorIndex, address, 
    /**
     * withdrawal amount in Gwei to match the CL repesentation and eventually ssz withdrawalsRoot
     */
    amount) {
        this.index = index;
        this.validatorIndex = validatorIndex;
        this.address = address;
        this.amount = amount;
    }
    static fromWithdrawalData(withdrawalData) {
        const { index: indexData, validatorIndex: validatorIndexData, address: addressData, amount: amountData, } = withdrawalData;
        const index = (0, types_1.toType)(indexData, types_1.TypeOutput.BigInt);
        const validatorIndex = (0, types_1.toType)(validatorIndexData, types_1.TypeOutput.BigInt);
        const address = new address_1.Address((0, types_1.toType)(addressData, types_1.TypeOutput.Buffer));
        const amount = (0, types_1.toType)(amountData, types_1.TypeOutput.BigInt);
        return new Withdrawal(index, validatorIndex, address, amount);
    }
    static fromValuesArray(withdrawalArray) {
        if (withdrawalArray.length !== 4) {
            throw Error(`Invalid withdrawalArray length expected=4 actual=${withdrawalArray.length}`);
        }
        const [index, validatorIndex, address, amount] = withdrawalArray;
        return Withdrawal.fromWithdrawalData({ index, validatorIndex, address, amount });
    }
    /**
     * Convert a withdrawal to a buffer array
     * @param withdrawal the withdrawal to convert
     * @returns buffer array of the withdrawal
     */
    static toBufferArray(withdrawal) {
        const { index, validatorIndex, address, amount } = withdrawal;
        const indexBuffer = (0, types_1.toType)(index, types_1.TypeOutput.BigInt) === BigInt(0)
            ? Buffer.alloc(0)
            : (0, types_1.toType)(index, types_1.TypeOutput.Buffer);
        const validatorIndexBuffer = (0, types_1.toType)(validatorIndex, types_1.TypeOutput.BigInt) === BigInt(0)
            ? Buffer.alloc(0)
            : (0, types_1.toType)(validatorIndex, types_1.TypeOutput.Buffer);
        let addressBuffer;
        if (address instanceof address_1.Address) {
            addressBuffer = address.buf;
        }
        else {
            addressBuffer = (0, types_1.toType)(address, types_1.TypeOutput.Buffer);
        }
        const amountBuffer = (0, types_1.toType)(amount, types_1.TypeOutput.BigInt) === BigInt(0)
            ? Buffer.alloc(0)
            : (0, types_1.toType)(amount, types_1.TypeOutput.Buffer);
        return [indexBuffer, validatorIndexBuffer, addressBuffer, amountBuffer];
    }
    raw() {
        return Withdrawal.toBufferArray(this);
    }
    toValue() {
        return {
            index: this.index,
            validatorIndex: this.validatorIndex,
            address: this.address.buf,
            amount: this.amount,
        };
    }
    toJSON() {
        return {
            index: (0, bytes_1.bigIntToHex)(this.index),
            validatorIndex: (0, bytes_1.bigIntToHex)(this.validatorIndex),
            address: '0x' + this.address.buf.toString('hex'),
            amount: (0, bytes_1.bigIntToHex)(this.amount),
        };
    }
}
exports.Withdrawal = Withdrawal;
//# sourceMappingURL=withdrawal.js.map