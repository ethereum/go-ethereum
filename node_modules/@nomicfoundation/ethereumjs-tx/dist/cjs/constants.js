"use strict";
/** EIP4844 constants */
Object.defineProperty(exports, "__esModule", { value: true });
exports.BYTES_PER_FIELD_ELEMENT = exports.FIELD_ELEMENTS_PER_BLOB = exports.MAX_TX_WRAP_KZG_COMMITMENTS = exports.LIMIT_BLOBS_PER_TX = exports.MAX_VERSIONED_HASHES_LIST_SIZE = exports.MAX_ACCESS_LIST_SIZE = exports.MAX_CALLDATA_SIZE = void 0;
exports.MAX_CALLDATA_SIZE = 16777216; // 2 ** 24
exports.MAX_ACCESS_LIST_SIZE = 16777216; // 2 ** 24
exports.MAX_VERSIONED_HASHES_LIST_SIZE = 16777216; // 2 ** 24
exports.LIMIT_BLOBS_PER_TX = 6; // 786432 / 2^17 (`MAX_BLOB_GAS_PER_BLOCK` / `GAS_PER_BLOB`)
exports.MAX_TX_WRAP_KZG_COMMITMENTS = 16777216; // 2 ** 24
exports.FIELD_ELEMENTS_PER_BLOB = 4096; // This is also in the Common 4844 parameters but needed here since types can't access Common params
exports.BYTES_PER_FIELD_ELEMENT = 32;
//# sourceMappingURL=constants.js.map