"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EventEmitterWrapper = void 0;
// IMPORTANT NOTE: This class is type-checked against the currently installed
// version of @types/node (10.x atm), and manually checked to be compatible with
// Node.js up to 14.3.0 (the latest release atm). There's a test that ensures
// that we are exporting all the EventEmitter's members, but it can't check the
// actual types of those members if they are functions.
//
// If a new version of Node.js adds new members to EventEmitter or overloads
// existing ones this class has to be updated, even if it still type-checks.
// This is a serious limitation ot DefinitelyTyped when the original, un-typed,
// library can change because of the user having a different version.
class EventEmitterWrapper {
    constructor(_wrapped) {
        this._wrapped = _wrapped;
    }
    addListener(event, listener) {
        this._wrapped.addListener(event, listener);
        return this;
    }
    on(event, listener) {
        this._wrapped.on(event, listener);
        return this;
    }
    once(event, listener) {
        this._wrapped.once(event, listener);
        return this;
    }
    prependListener(event, listener) {
        this._wrapped.prependListener(event, listener);
        return this;
    }
    prependOnceListener(event, listener) {
        this._wrapped.prependOnceListener(event, listener);
        return this;
    }
    removeListener(event, listener) {
        this._wrapped.removeListener(event, listener);
        return this;
    }
    off(event, listener) {
        this._wrapped.off(event, listener);
        return this;
    }
    removeAllListeners(event) {
        this._wrapped.removeAllListeners(event);
        return this;
    }
    setMaxListeners(n) {
        this._wrapped.setMaxListeners(n);
        return this;
    }
    getMaxListeners() {
        return this._wrapped.getMaxListeners();
    }
    // eslint-disable-next-line @typescript-eslint/ban-types
    listeners(event) {
        return this._wrapped.listeners(event);
    }
    // eslint-disable-next-line @typescript-eslint/ban-types
    rawListeners(event) {
        return this._wrapped.rawListeners(event);
    }
    emit(event, ...args) {
        return this._wrapped.emit(event, ...args);
    }
    eventNames() {
        return this._wrapped.eventNames();
    }
    listenerCount(type) {
        return this._wrapped.listenerCount(type);
    }
}
exports.EventEmitterWrapper = EventEmitterWrapper;
//# sourceMappingURL=event-emitter.js.map