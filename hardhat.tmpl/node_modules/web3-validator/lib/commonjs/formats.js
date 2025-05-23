"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const address_js_1 = require("./validation/address.js");
const block_js_1 = require("./validation/block.js");
const bloom_js_1 = require("./validation/bloom.js");
const boolean_js_1 = require("./validation/boolean.js");
const bytes_js_1 = require("./validation/bytes.js");
const filter_js_1 = require("./validation/filter.js");
const string_js_1 = require("./validation/string.js");
const numbers_js_1 = require("./validation/numbers.js");
const formats = {
    address: (data) => (0, address_js_1.isAddress)(data),
    bloom: (data) => (0, bloom_js_1.isBloom)(data),
    blockNumber: (data) => (0, block_js_1.isBlockNumber)(data),
    blockTag: (data) => (0, block_js_1.isBlockTag)(data),
    blockNumberOrTag: (data) => (0, block_js_1.isBlockNumberOrTag)(data),
    bool: (data) => (0, boolean_js_1.isBoolean)(data),
    bytes: (data) => (0, bytes_js_1.isBytes)(data),
    filter: (data) => (0, filter_js_1.isFilterObject)(data),
    hex: (data) => (0, string_js_1.isHexStrict)(data),
    uint: (data) => (0, numbers_js_1.isUInt)(data),
    int: (data) => (0, numbers_js_1.isInt)(data),
    number: (data) => (0, numbers_js_1.isNumber)(data),
    string: (data) => (0, string_js_1.isString)(data),
};
// generate formats for all numbers types
for (let bitSize = 8; bitSize <= 256; bitSize += 8) {
    formats[`int${bitSize}`] = data => (0, numbers_js_1.isInt)(data, { bitSize });
    formats[`uint${bitSize}`] = data => (0, numbers_js_1.isUInt)(data, { bitSize });
}
// generate bytes
for (let size = 1; size <= 32; size += 1) {
    formats[`bytes${size}`] = data => (0, bytes_js_1.isBytes)(data, { size });
}
formats.bytes256 = formats.bytes;
exports.default = formats;
//# sourceMappingURL=formats.js.map