"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.decodeInstructions = void 0;
const model_1 = require("./model");
const opcodes_1 = require("./opcodes");
function jumpLetterToJumpType(letter) {
    if (letter === "i") {
        return model_1.JumpType.INTO_FUNCTION;
    }
    if (letter === "o") {
        return model_1.JumpType.OUTOF_FUNCTION;
    }
    return model_1.JumpType.NOT_JUMP;
}
function uncompressSourcemaps(compressedSourcemap) {
    const mappings = [];
    const compressedMappings = compressedSourcemap.split(";");
    for (let i = 0; i < compressedMappings.length; i++) {
        const parts = compressedMappings[i].split(":");
        const hasParts0 = parts[0] !== undefined && parts[0] !== "";
        const hasParts1 = parts[1] !== undefined && parts[1] !== "";
        const hasParts2 = parts[2] !== undefined && parts[2] !== "";
        const hasParts3 = parts[3] !== undefined && parts[3] !== "";
        const hasEveryPart = hasParts0 && hasParts1 && hasParts2 && hasParts3;
        // See: https://github.com/nomiclabs/hardhat/issues/593
        if (i === 0 && !hasEveryPart) {
            mappings.push({
                jumpType: model_1.JumpType.NOT_JUMP,
                location: {
                    file: -1,
                    offset: 0,
                    length: 0,
                },
            });
            continue;
        }
        mappings.push({
            location: {
                offset: hasParts0 ? +parts[0] : mappings[i - 1].location.offset,
                length: hasParts1 ? +parts[1] : mappings[i - 1].location.length,
                file: hasParts2 ? +parts[2] : mappings[i - 1].location.file,
            },
            jumpType: hasParts3
                ? jumpLetterToJumpType(parts[3])
                : mappings[i - 1].jumpType,
        });
    }
    return mappings;
}
function addUnmappedInstructions(instructions, bytecode) {
    const lastInstrPc = instructions[instructions.length - 1].pc;
    let bytesIndex = lastInstrPc + 1;
    while (bytecode[bytesIndex] !== opcodes_1.Opcode.INVALID) {
        const opcode = bytecode[bytesIndex];
        let pushData;
        let pushDataLenth = 0;
        if ((0, opcodes_1.isPush)(opcode)) {
            pushDataLenth = (0, opcodes_1.getPushLength)(opcode);
            pushData = bytecode.slice(bytesIndex + 1, bytesIndex + 1 + pushDataLenth);
        }
        const jumpType = (0, opcodes_1.isJump)(opcode)
            ? model_1.JumpType.INTERNAL_JUMP
            : model_1.JumpType.NOT_JUMP;
        const instruction = new model_1.Instruction(bytesIndex, opcode, jumpType, pushData);
        instructions.push(instruction);
        bytesIndex += (0, opcodes_1.getOpcodeLength)(opcode);
    }
}
function decodeInstructions(bytecode, compressedSourcemaps, fileIdToSourceFile, isDeployment) {
    const sourceMaps = uncompressSourcemaps(compressedSourcemaps);
    const instructions = [];
    let bytesIndex = 0;
    // Solidity inlines some data after the contract, so we stop decoding
    // as soon as we have enough instructions as uncompressed mappings. This is
    // not very documented, but we manually tested that it works.
    while (instructions.length < sourceMaps.length) {
        const pc = bytesIndex;
        const opcode = bytecode[pc];
        const sourceMap = sourceMaps[instructions.length];
        let pushData;
        let location;
        const jumpType = (0, opcodes_1.isJump)(opcode) && sourceMap.jumpType === model_1.JumpType.NOT_JUMP
            ? model_1.JumpType.INTERNAL_JUMP
            : sourceMap.jumpType;
        if ((0, opcodes_1.isPush)(opcode)) {
            const length = (0, opcodes_1.getPushLength)(opcode);
            pushData = bytecode.slice(bytesIndex + 1, bytesIndex + 1 + length);
        }
        if (sourceMap.location.file !== -1) {
            const file = fileIdToSourceFile.get(sourceMap.location.file);
            if (file !== undefined) {
                location = new model_1.SourceLocation(file, sourceMap.location.offset, sourceMap.location.length);
            }
        }
        const instruction = new model_1.Instruction(pc, opcode, jumpType, pushData, location);
        instructions.push(instruction);
        bytesIndex += (0, opcodes_1.getOpcodeLength)(opcode);
    }
    // See: https://github.com/ethereum/solidity/issues/9133
    if (isDeployment) {
        addUnmappedInstructions(instructions, bytecode);
    }
    return instructions;
}
exports.decodeInstructions = decodeInstructions;
//# sourceMappingURL=source-maps.js.map