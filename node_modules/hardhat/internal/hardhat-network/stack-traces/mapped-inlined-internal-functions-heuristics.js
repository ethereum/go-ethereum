"use strict";
/**
 * This file includes Solidity tracing heuristics for solc starting with version
 * 0.6.9.
 *
 * This solc version introduced a significant change to how sourcemaps are
 * handled for inline yul/internal functions. These were mapped to the
 * unmapped/-1 file before, which lead to many unmapped reverts. Now, they are
 * mapped to the part of the Solidity source that lead to their inlining.
 *
 * This change is a very positive change, as errors would point to the correct
 * line by default. The only problem is that we used to rely very heavily on
 * unmapped reverts to decide when our error detection heuristics were to be
 * run. In fact, this heuristics were first introduced because of unmapped
 * reverts.
 *
 * Instead of synthetically completing stack traces when unmapped reverts occur,
 * we now start from complete stack traces and adjust them if we can provide
 * more meaningful errors.
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.adjustStackTrace = exports.stackTraceMayRequireAdjustments = void 0;
const semver_1 = __importDefault(require("semver"));
const message_trace_1 = require("./message-trace");
const opcodes_1 = require("./opcodes");
const solidity_stack_trace_1 = require("./solidity-stack-trace");
const FIRST_SOLC_VERSION_WITH_MAPPED_SMALL_INTERNAL_FUNCTIONS = "0.6.9";
function stackTraceMayRequireAdjustments(stackTrace, decodedTrace) {
    if (stackTrace.length === 0) {
        return false;
    }
    const lastFrame = stackTrace[stackTrace.length - 1];
    return (lastFrame.type === solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR &&
        !lastFrame.isInvalidOpcodeError &&
        lastFrame.message.isEmpty() &&
        semver_1.default.gte(decodedTrace.bytecode.compilerVersion, FIRST_SOLC_VERSION_WITH_MAPPED_SMALL_INTERNAL_FUNCTIONS));
}
exports.stackTraceMayRequireAdjustments = stackTraceMayRequireAdjustments;
function adjustStackTrace(stackTrace, decodedTrace) {
    const start = stackTrace.slice(0, -1);
    const [revert] = stackTrace.slice(-1);
    if (isNonContractAccountCalledError(decodedTrace)) {
        return [
            ...start,
            {
                type: solidity_stack_trace_1.StackTraceEntryType.NONCONTRACT_ACCOUNT_CALLED_ERROR,
                sourceReference: revert.sourceReference,
            },
        ];
    }
    if (isConstructorInvalidParamsError(decodedTrace)) {
        return [
            ...start,
            {
                type: solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR,
                sourceReference: revert.sourceReference,
            },
        ];
    }
    if (isCallInvalidParamsError(decodedTrace)) {
        return [
            ...start,
            {
                type: solidity_stack_trace_1.StackTraceEntryType.INVALID_PARAMS_ERROR,
                sourceReference: revert.sourceReference,
            },
        ];
    }
    return stackTrace;
}
exports.adjustStackTrace = adjustStackTrace;
function isNonContractAccountCalledError(decodedTrace) {
    return matchOpcodes(decodedTrace, -9, [
        opcodes_1.Opcode.EXTCODESIZE,
        opcodes_1.Opcode.ISZERO,
        opcodes_1.Opcode.DUP1,
        opcodes_1.Opcode.ISZERO,
    ]);
}
function isConstructorInvalidParamsError(decodedTrace) {
    if (!(0, message_trace_1.isDecodedCreateTrace)(decodedTrace)) {
        return false;
    }
    return (matchOpcodes(decodedTrace, -20, [opcodes_1.Opcode.CODESIZE]) &&
        matchOpcodes(decodedTrace, -15, [opcodes_1.Opcode.CODECOPY]) &&
        matchOpcodes(decodedTrace, -7, [opcodes_1.Opcode.LT, opcodes_1.Opcode.ISZERO]));
}
function isCallInvalidParamsError(decodedTrace) {
    if (!(0, message_trace_1.isDecodedCallTrace)(decodedTrace)) {
        return false;
    }
    return (matchOpcodes(decodedTrace, -11, [opcodes_1.Opcode.CALLDATASIZE]) &&
        matchOpcodes(decodedTrace, -7, [opcodes_1.Opcode.LT, opcodes_1.Opcode.ISZERO]));
}
function matchOpcode(decodedTrace, stepIndex, opcode) {
    const [step] = decodedTrace.steps.slice(stepIndex, stepIndex + 1);
    if (step === undefined || !(0, message_trace_1.isEvmStep)(step)) {
        return false;
    }
    const instruction = decodedTrace.bytecode.getInstruction(step.pc);
    return instruction.opcode === opcode;
}
function matchOpcodes(decodedTrace, firstStepIndex, opcodes) {
    let index = firstStepIndex;
    for (const opcode of opcodes) {
        if (!matchOpcode(decodedTrace, index, opcode)) {
            return false;
        }
        index += 1;
    }
    return true;
}
//# sourceMappingURL=mapped-inlined-internal-functions-heuristics.js.map