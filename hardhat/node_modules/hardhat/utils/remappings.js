"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.applyRemappings = void 0;
function applyRemappings(remappings, sourceName) {
    const selectedRemapping = { from: "", to: "" };
    for (const [from, to] of Object.entries(remappings)) {
        if (sourceName.startsWith(from) &&
            from.length >= selectedRemapping.from.length) {
            [selectedRemapping.from, selectedRemapping.to] = [from, to];
        }
    }
    return sourceName.replace(selectedRemapping.from, selectedRemapping.to);
}
exports.applyRemappings = applyRemappings;
//# sourceMappingURL=remappings.js.map