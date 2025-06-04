"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getBytecodeWithoutMetadata = exports.fail = void 0;
function fail(future, failure) {
    return {
        success: false,
        failure: {
            futureId: future.id,
            failure,
        },
    };
}
exports.fail = fail;
const METADATA_LENGTH = 2;
function getMetadataSectionLength(bytecode) {
    try {
        return bytecode.slice(-METADATA_LENGTH).readUInt16BE(0) + METADATA_LENGTH;
    }
    catch {
        return undefined;
    }
}
function isValidMetadata(data) {
    const { decode } = require("cbor");
    try {
        decode(data);
        return true;
    }
    catch (e) {
        return false;
    }
}
function getBytecodeWithoutMetadata(bytecode) {
    const bytecodeBuffer = Buffer.from(bytecode.slice(2), "hex");
    const metadataSectionLength = getMetadataSectionLength(bytecodeBuffer);
    if (metadataSectionLength === undefined) {
        return bytecode;
    }
    const metadataPayload = bytecodeBuffer.slice(-metadataSectionLength, -METADATA_LENGTH);
    if (isValidMetadata(metadataPayload)) {
        return bytecodeBuffer.slice(0, -metadataSectionLength).toString("hex");
    }
    return bytecode;
}
exports.getBytecodeWithoutMetadata = getBytecodeWithoutMetadata;
//# sourceMappingURL=utils.js.map