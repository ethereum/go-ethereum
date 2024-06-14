/** EIP4844 constants */
export const MAX_CALLDATA_SIZE = 16777216; // 2 ** 24
export const MAX_ACCESS_LIST_SIZE = 16777216; // 2 ** 24
export const MAX_VERSIONED_HASHES_LIST_SIZE = 16777216; // 2 ** 24
export const LIMIT_BLOBS_PER_TX = 6; // 786432 / 2^17 (`MAX_BLOB_GAS_PER_BLOCK` / `GAS_PER_BLOB`)
export const MAX_TX_WRAP_KZG_COMMITMENTS = 16777216; // 2 ** 24
export const FIELD_ELEMENTS_PER_BLOB = 4096; // This is also in the Common 4844 parameters but needed here since types can't access Common params
export const BYTES_PER_FIELD_ELEMENT = 32;
//# sourceMappingURL=constants.js.map