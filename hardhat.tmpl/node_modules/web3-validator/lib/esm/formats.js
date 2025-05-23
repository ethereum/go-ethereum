import { isAddress } from './validation/address.js';
import { isBlockNumber, isBlockNumberOrTag, isBlockTag } from './validation/block.js';
import { isBloom } from './validation/bloom.js';
import { isBoolean } from './validation/boolean.js';
import { isBytes } from './validation/bytes.js';
import { isFilterObject } from './validation/filter.js';
import { isHexStrict, isString } from './validation/string.js';
import { isNumber, isInt, isUInt } from './validation/numbers.js';
const formats = {
    address: (data) => isAddress(data),
    bloom: (data) => isBloom(data),
    blockNumber: (data) => isBlockNumber(data),
    blockTag: (data) => isBlockTag(data),
    blockNumberOrTag: (data) => isBlockNumberOrTag(data),
    bool: (data) => isBoolean(data),
    bytes: (data) => isBytes(data),
    filter: (data) => isFilterObject(data),
    hex: (data) => isHexStrict(data),
    uint: (data) => isUInt(data),
    int: (data) => isInt(data),
    number: (data) => isNumber(data),
    string: (data) => isString(data),
};
// generate formats for all numbers types
for (let bitSize = 8; bitSize <= 256; bitSize += 8) {
    formats[`int${bitSize}`] = data => isInt(data, { bitSize });
    formats[`uint${bitSize}`] = data => isUInt(data, { bitSize });
}
// generate bytes
for (let size = 1; size <= 32; size += 1) {
    formats[`bytes${size}`] = data => isBytes(data, { size });
}
formats.bytes256 = formats.bytes;
export default formats;
//# sourceMappingURL=formats.js.map