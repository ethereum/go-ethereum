"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.printStackTrace = exports.printCallTrace = exports.printPrecompileTrace = exports.printCreateTrace = exports.printMessageTrace = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const chalk_1 = __importDefault(require("chalk"));
const message_trace_1 = require("./message-trace");
const model_1 = require("./model");
const opcodes_1 = require("./opcodes");
const solidity_stack_trace_1 = require("./solidity-stack-trace");
const MARGIN_SPACE = 6;
function printMessageTrace(trace, depth = 0) {
    console.log("");
    if ((0, message_trace_1.isCreateTrace)(trace)) {
        printCreateTrace(trace, depth);
    }
    else if ((0, message_trace_1.isPrecompileTrace)(trace)) {
        printPrecompileTrace(trace, depth);
    }
    else {
        printCallTrace(trace, depth);
    }
    console.log("");
}
exports.printMessageTrace = printMessageTrace;
function printCreateTrace(trace, depth) {
    const margin = "".padStart(depth * MARGIN_SPACE);
    console.log(`${margin}Create trace`);
    if (trace.bytecode !== undefined) {
        console.log(`${margin} deploying contract: ${trace.bytecode.contract.location.file.sourceName}:${trace.bytecode.contract.name}`);
        console.log(`${margin} code: ${(0, ethereumjs_util_1.bytesToHex)(trace.code)}`);
    }
    else {
        console.log(`${margin} unrecognized deployment code: ${(0, ethereumjs_util_1.bytesToHex)(trace.code)}`);
    }
    console.log(`${margin} value: ${trace.value.toString(10)}`);
    if (trace.deployedContract !== undefined) {
        console.log(`${margin} contract address: ${(0, ethereumjs_util_1.bytesToHex)(trace.deployedContract)}`);
    }
    if (trace.exit.isError()) {
        console.log(`${margin} error: ${trace.exit.getReason()}`);
        // The return data is the deployed-bytecode if there was no error, so we don't show it
        console.log(`${margin} returnData: ${(0, ethereumjs_util_1.bytesToHex)(trace.returnData)}`);
    }
    traceSteps(trace, depth);
}
exports.printCreateTrace = printCreateTrace;
function printPrecompileTrace(trace, depth) {
    const margin = "".padStart(depth * MARGIN_SPACE);
    console.log(`${margin}Precompile trace`);
    console.log(`${margin} precompile number: ${trace.precompile}`);
    console.log(`${margin} value: ${trace.value.toString(10)}`);
    console.log(`${margin} calldata: ${(0, ethereumjs_util_1.bytesToHex)(trace.calldata)}`);
    if (trace.exit.isError()) {
        console.log(`${margin} error: ${trace.exit.getReason()}`);
    }
    console.log(`${margin} returnData: ${(0, ethereumjs_util_1.bytesToHex)(trace.returnData)}`);
}
exports.printPrecompileTrace = printPrecompileTrace;
function printCallTrace(trace, depth) {
    const margin = "".padStart(depth * MARGIN_SPACE);
    console.log(`${margin}Call trace`);
    if (trace.bytecode !== undefined) {
        console.log(`${margin} calling contract: ${trace.bytecode.contract.location.file.sourceName}:${trace.bytecode.contract.name}`);
    }
    else {
        console.log(`${margin} unrecognized contract code: ${(0, ethereumjs_util_1.bytesToHex)(trace.code)}`);
        console.log(`${margin} contract: ${(0, ethereumjs_util_1.bytesToHex)(trace.address)}`);
    }
    console.log(`${margin} value: ${trace.value.toString(10)}`);
    console.log(`${margin} calldata: ${(0, ethereumjs_util_1.bytesToHex)(trace.calldata)}`);
    if (trace.exit.isError()) {
        console.log(`${margin} error: ${trace.exit.getReason()}`);
    }
    console.log(`${margin} returnData: ${(0, ethereumjs_util_1.bytesToHex)(trace.returnData)}`);
    traceSteps(trace, depth);
}
exports.printCallTrace = printCallTrace;
function traceSteps(trace, depth) {
    const margin = "".padStart(depth * MARGIN_SPACE);
    console.log(`${margin} steps:`);
    console.log("");
    for (const step of trace.steps) {
        if ((0, message_trace_1.isEvmStep)(step)) {
            const pc = step.pc.toString(10).padStart(3, "0").padStart(5);
            if (trace.bytecode !== undefined) {
                const inst = trace.bytecode.getInstruction(step.pc);
                let location = "";
                if (inst.location !== undefined) {
                    location += inst.location.file.sourceName;
                    const func = inst.location.getContainingFunction();
                    if (func !== undefined) {
                        location += `:${func.contract?.name ?? func.location.file.sourceName}:${func.name}`;
                    }
                    location += `   -  ${inst.location.offset}:${inst.location.length}`;
                }
                if ((0, opcodes_1.isJump)(inst.opcode)) {
                    const jump = inst.jumpType !== model_1.JumpType.NOT_JUMP
                        ? chalk_1.default.bold(`(${model_1.JumpType[inst.jumpType]})`)
                        : "";
                    console.log(`${margin}  ${pc}   ${opcodes_1.Opcode[inst.opcode]} ${jump}`.padEnd(50), location);
                }
                else if ((0, opcodes_1.isPush)(inst.opcode)) {
                    console.log(`${margin}  ${pc}   ${opcodes_1.Opcode[inst.opcode]} ${(0, ethereumjs_util_1.bytesToHex)(inst.pushData)}`.padEnd(50), location);
                }
                else {
                    console.log(`${margin}  ${pc}   ${opcodes_1.Opcode[inst.opcode]}`.padEnd(50), location);
                }
            }
            else {
                console.log(`${margin}  ${pc}`);
            }
        }
        else {
            printMessageTrace(step, depth + 1);
        }
    }
}
function flattenSourceReference(sourceReference) {
    if (sourceReference === undefined) {
        return undefined;
    }
    return {
        ...sourceReference,
        file: sourceReference.sourceName,
    };
}
function printStackTrace(trace) {
    const withDecodedMessages = trace.map((entry) => entry.type === solidity_stack_trace_1.StackTraceEntryType.REVERT_ERROR
        ? { ...entry, message: entry.message.decodeError() }
        : entry);
    const withHexAddress = withDecodedMessages.map((entry) => "address" in entry
        ? { ...entry, address: (0, ethereumjs_util_1.bytesToHex)(entry.address) }
        : entry);
    const withTextualType = withHexAddress.map((entry) => ({
        ...entry,
        type: solidity_stack_trace_1.StackTraceEntryType[entry.type],
    }));
    const withFlattenedSourceReferences = withTextualType.map((entry) => ({
        ...entry,
        sourceReference: flattenSourceReference(entry.sourceReference),
    }));
    console.log(JSON.stringify(withFlattenedSourceReferences, (key, value) => (typeof value === "bigint" ? value.toString() : value), 2));
}
exports.printStackTrace = printStackTrace;
//# sourceMappingURL=debug.js.map