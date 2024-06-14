"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeLibraryRuntimeBytecodeIfNecessary = exports.zeroOutSlices = exports.zeroOutAddresses = exports.linkHexStringBytecode = exports.normalizeCompilerOutputBytecode = exports.getLibraryAddressPositions = void 0;
const opcodes_1 = require("./opcodes");
function getLibraryAddressPositions(bytecodeOutput) {
    const positions = [];
    for (const libs of Object.values(bytecodeOutput.linkReferences)) {
        for (const references of Object.values(libs)) {
            for (const ref of references) {
                positions.push(ref.start);
            }
        }
    }
    return positions;
}
exports.getLibraryAddressPositions = getLibraryAddressPositions;
function normalizeCompilerOutputBytecode(compilerOutputBytecodeObject, addressesPositions) {
    const ZERO_ADDRESS = "0000000000000000000000000000000000000000";
    for (const position of addressesPositions) {
        compilerOutputBytecodeObject = linkHexStringBytecode(compilerOutputBytecodeObject, ZERO_ADDRESS, position);
    }
    return Buffer.from(compilerOutputBytecodeObject, "hex");
}
exports.normalizeCompilerOutputBytecode = normalizeCompilerOutputBytecode;
function linkHexStringBytecode(code, address, position) {
    if (address.startsWith("0x")) {
        address = address.substring(2);
    }
    return (code.substring(0, position * 2) +
        address +
        code.slice(position * 2 + address.length));
}
exports.linkHexStringBytecode = linkHexStringBytecode;
function zeroOutAddresses(code, addressesPositions) {
    const addressesSlices = addressesPositions.map((start) => ({
        start,
        length: 20,
    }));
    return zeroOutSlices(code, addressesSlices);
}
exports.zeroOutAddresses = zeroOutAddresses;
function zeroOutSlices(code, slices) {
    for (const { start, length } of slices) {
        code = Buffer.concat([
            code.slice(0, start),
            Buffer.alloc(length, 0),
            code.slice(start + length),
        ]);
    }
    return code;
}
exports.zeroOutSlices = zeroOutSlices;
function normalizeLibraryRuntimeBytecodeIfNecessary(code) {
    // Libraries' protection normalization:
    // Solidity 0.4.20 introduced a protection to prevent libraries from being called directly.
    // This is done by modifying the code on deployment, and hard-coding the contract address.
    // The first instruction is a PUSH20 of the address, which we zero-out as a way of normalizing
    // it. Note that it's also zeroed-out in the compiler output.
    if (code[0] === opcodes_1.Opcode.PUSH20) {
        return zeroOutAddresses(code, [1]);
    }
    return code;
}
exports.normalizeLibraryRuntimeBytecodeIfNecessary = normalizeLibraryRuntimeBytecodeIfNecessary;
//# sourceMappingURL=library-utils.js.map