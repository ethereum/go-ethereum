"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Exit = exports.ExitCode = void 0;
const napi_rs_1 = require("../../../../common/napi-rs");
var ExitCode;
(function (ExitCode) {
    ExitCode[ExitCode["SUCCESS"] = 0] = "SUCCESS";
    ExitCode[ExitCode["REVERT"] = 1] = "REVERT";
    ExitCode[ExitCode["OUT_OF_GAS"] = 2] = "OUT_OF_GAS";
    ExitCode[ExitCode["INTERNAL_ERROR"] = 3] = "INTERNAL_ERROR";
    ExitCode[ExitCode["INVALID_OPCODE"] = 4] = "INVALID_OPCODE";
    ExitCode[ExitCode["STACK_UNDERFLOW"] = 5] = "STACK_UNDERFLOW";
    ExitCode[ExitCode["CODESIZE_EXCEEDS_MAXIMUM"] = 6] = "CODESIZE_EXCEEDS_MAXIMUM";
    ExitCode[ExitCode["CREATE_COLLISION"] = 7] = "CREATE_COLLISION";
    ExitCode[ExitCode["STATIC_STATE_CHANGE"] = 8] = "STATIC_STATE_CHANGE";
})(ExitCode = exports.ExitCode || (exports.ExitCode = {}));
class Exit {
    static fromEdrSuccessReason(reason) {
        const { SuccessReason } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
        switch (reason) {
            case 0 /* SuccessReason.Stop */:
            case 1 /* SuccessReason.Return */:
            case 2 /* SuccessReason.SelfDestruct */:
                return new Exit(ExitCode.SUCCESS);
        }
        const _exhaustiveCheck = reason;
    }
    static fromEdrExceptionalHalt(halt) {
        const { ExceptionalHalt } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
        switch (halt) {
            case 0 /* ExceptionalHalt.OutOfGas */:
                return new Exit(ExitCode.OUT_OF_GAS);
            case 1 /* ExceptionalHalt.OpcodeNotFound */:
            case 2 /* ExceptionalHalt.InvalidFEOpcode */:
            // Returned when an opcode is not implemented for the hardfork
            case 4 /* ExceptionalHalt.NotActivated */:
                return new Exit(ExitCode.INVALID_OPCODE);
            case 5 /* ExceptionalHalt.StackUnderflow */:
                return new Exit(ExitCode.STACK_UNDERFLOW);
            case 8 /* ExceptionalHalt.CreateCollision */:
                return new Exit(ExitCode.CREATE_COLLISION);
            case 11 /* ExceptionalHalt.CreateContractSizeLimit */:
                return new Exit(ExitCode.CODESIZE_EXCEEDS_MAXIMUM);
            default: {
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw new Error(`Unmatched EDR exceptional halt: ${halt}`);
            }
        }
    }
    constructor(kind) {
        this.kind = kind;
    }
    isError() {
        return this.kind !== ExitCode.SUCCESS;
    }
    getReason() {
        switch (this.kind) {
            case ExitCode.SUCCESS:
                return "Success";
            case ExitCode.REVERT:
                return "Reverted";
            case ExitCode.OUT_OF_GAS:
                return "Out of gas";
            case ExitCode.INTERNAL_ERROR:
                return "Internal error";
            case ExitCode.INVALID_OPCODE:
                return "Invalid opcode";
            case ExitCode.STACK_UNDERFLOW:
                return "Stack underflow";
            case ExitCode.CODESIZE_EXCEEDS_MAXIMUM:
                return "Codesize exceeds maximum";
            case ExitCode.CREATE_COLLISION:
                return "Create collision";
            case ExitCode.STATIC_STATE_CHANGE:
                return "Static state change";
        }
        const _exhaustiveCheck = this.kind;
    }
    getEdrExceptionalHalt() {
        const { ExceptionalHalt } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
        switch (this.kind) {
            case ExitCode.OUT_OF_GAS:
                return 0 /* ExceptionalHalt.OutOfGas */;
            case ExitCode.INVALID_OPCODE:
                return 1 /* ExceptionalHalt.OpcodeNotFound */;
            case ExitCode.CODESIZE_EXCEEDS_MAXIMUM:
                return 11 /* ExceptionalHalt.CreateContractSizeLimit */;
            case ExitCode.CREATE_COLLISION:
                return 8 /* ExceptionalHalt.CreateCollision */;
            default:
                // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
                throw new Error(`Unmatched exit code: ${this.kind}`);
        }
    }
}
exports.Exit = Exit;
//# sourceMappingURL=exit.js.map