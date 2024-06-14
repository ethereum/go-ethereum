"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MemoryJournal = void 0;
const emitExecutionEvent_1 = require("./utils/emitExecutionEvent");
/**
 * An in-memory journal.
 *
 * @beta
 */
class MemoryJournal {
    _executionEventListener;
    messages = [];
    constructor(_executionEventListener) {
        this._executionEventListener = _executionEventListener;
    }
    record(message) {
        this._log(message);
        this.messages.push(message);
    }
    async *read() {
        for (const message of this.messages) {
            yield message;
        }
    }
    _log(message) {
        if (this._executionEventListener !== undefined) {
            (0, emitExecutionEvent_1.emitExecutionEvent)(message, this._executionEventListener);
        }
    }
}
exports.MemoryJournal = MemoryJournal;
//# sourceMappingURL=memory-journal.js.map