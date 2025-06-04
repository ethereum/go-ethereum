"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.addHint = addHint;
exports.addHints = addHints;
exports.addHook = addHook;
function addHint(abi, name, fn) {
    const res = [];
    for (const elm of abi) {
        if (elm.name === name)
            res.push({ ...elm, hint: fn });
        else
            res.push(elm);
    }
    return res;
}
function addHints(abi, map) {
    const res = [];
    for (const elm of abi) {
        if (['event', 'function'].includes(elm.type) && elm.name && map[elm.name]) {
            res.push({ ...elm, hint: map[elm.name] });
        }
        else
            res.push(elm);
    }
    return res;
}
function addHook(abi, name, fn) {
    const res = [];
    for (const elm of abi) {
        if (elm.type === 'function' && elm.name === name)
            res.push({ ...elm, hook: fn });
        else
            res.push(elm);
    }
    return res;
}
//# sourceMappingURL=common.js.map