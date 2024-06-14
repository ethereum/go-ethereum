"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.serializeReplacer = void 0;
/**
 * When stringifying messages to the journal, this defines the replacer.
 */
function serializeReplacer(_key, value) {
    if (value instanceof Set) {
        return Array.from(value).sort();
    }
    if (value instanceof Map) {
        return Object.fromEntries(value);
    }
    if (typeof value === "bigint") {
        return { _kind: "bigint", value: value.toString(10) };
    }
    if (value instanceof Object && !(value instanceof Array)) {
        const obj = value;
        return Object.keys(obj)
            .sort()
            .reduce((sorted, key) => {
            sorted[key] = obj[key];
            return sorted;
        }, {});
    }
    return value;
}
exports.serializeReplacer = serializeReplacer;
//# sourceMappingURL=serialize-replacer.js.map