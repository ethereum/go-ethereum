"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.parseTaskIdentifier = void 0;
function parseTaskIdentifier(taskIdentifier) {
    if (typeof taskIdentifier === "string") {
        return {
            scope: undefined,
            task: taskIdentifier,
        };
    }
    else {
        return {
            scope: taskIdentifier.scope,
            task: taskIdentifier.task,
        };
    }
}
exports.parseTaskIdentifier = parseTaskIdentifier;
//# sourceMappingURL=util.js.map