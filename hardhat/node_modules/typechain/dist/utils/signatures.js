"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getSignatureForFn = exports.getArgumentForSignature = exports.getIndexedSignatureForEvent = exports.getFullSignatureForEvent = exports.getFullSignatureAsSymbolForEvent = void 0;
function getFullSignatureAsSymbolForEvent(event) {
    return `${event.name}_${event.inputs
        .map((e) => {
        if (e.type.type === 'array') {
            return e.type.itemType.originalType + '_array';
        }
        else {
            return e.type.originalType;
        }
    })
        .join('_')}`;
}
exports.getFullSignatureAsSymbolForEvent = getFullSignatureAsSymbolForEvent;
function getFullSignatureForEvent(event) {
    return `${event.name}(${event.inputs.map((e) => getArgumentForSignature(e)).join(',')})`;
}
exports.getFullSignatureForEvent = getFullSignatureForEvent;
function getIndexedSignatureForEvent(event) {
    const indexedType = event.inputs.filter((e) => e.isIndexed);
    return `${event.name}(${indexedType.map((e) => getArgumentForSignature(e)).join(',')})`;
}
exports.getIndexedSignatureForEvent = getIndexedSignatureForEvent;
function getArgumentForSignature(argument) {
    var _a;
    if (argument.type.originalType === 'tuple') {
        return `(${argument.type.components.map((i) => getArgumentForSignature(i)).join(',')})`;
    }
    else if (argument.type.originalType.startsWith('tuple')) {
        const arr = argument.type;
        return `${getArgumentForSignature({ name: '', type: arr.itemType })}[${((_a = arr.size) === null || _a === void 0 ? void 0 : _a.toString()) || ''}]`;
    }
    else {
        return argument.type.originalType;
    }
}
exports.getArgumentForSignature = getArgumentForSignature;
function getSignatureForFn(fn) {
    return `${fn.name}(${fn.inputs.map((i) => getArgumentForSignature(i)).join(',')})`;
}
exports.getSignatureForFn = getSignatureForFn;
//# sourceMappingURL=signatures.js.map