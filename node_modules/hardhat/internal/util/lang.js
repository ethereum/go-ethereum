"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.mapValues = exports.fromEntries = void 0;
function fromEntries(entries) {
    return Object.assign({}, ...entries.map(([name, value]) => ({
        [name]: value,
    })));
}
exports.fromEntries = fromEntries;
function mapValues(o, callback) {
    const result = {};
    for (const [key, value] of Object.entries(o)) {
        result[key] = callback(value);
    }
    return result;
}
exports.mapValues = mapValues;
//# sourceMappingURL=lang.js.map