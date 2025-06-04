"use strict";
/**
 * Ported to Typescript from original implementation below:
 * https://github.com/ahultgren/async-eventemitter -- MIT licensed
 *
 * Type Definitions based on work by: patarapolw <https://github.com/patarapolw> -- MIT licensed
 * that was contributed to Definitely Typed below:
 * https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/async-eventemitter
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.AsyncEventEmitter = void 0;
const events_1 = require("events");
async function runInSeries(context, tasks, data) {
    let error;
    for await (const task of tasks) {
        try {
            if (task.length < 2) {
                //sync
                task.call(context, data);
            }
            else {
                await new Promise((resolve, reject) => {
                    task.call(context, data, (error) => {
                        if (error) {
                            reject(error);
                        }
                        else {
                            resolve();
                        }
                    });
                });
            }
        }
        catch (e) {
            error = e;
        }
    }
    if (error) {
        throw error;
    }
}
class AsyncEventEmitter extends events_1.EventEmitter {
    emit(event, ...args) {
        let [data, callback] = args;
        const self = this;
        let listeners = self._events[event] ?? [];
        // Optional data argument
        if (callback === undefined && typeof data === 'function') {
            callback = data;
            data = undefined;
        }
        // Special treatment of internal newListener and removeListener events
        if (event === 'newListener' || event === 'removeListener') {
            data = {
                event: data,
                fn: callback,
            };
            callback = undefined;
        }
        // A single listener is just a function not an array...
        listeners = Array.isArray(listeners) ? listeners : [listeners];
        runInSeries(self, listeners.slice(), data).then(callback).catch(callback);
        return self.listenerCount(event) > 0;
    }
    once(event, listener) {
        const self = this;
        let g;
        if (typeof listener !== 'function') {
            throw new TypeError('listener must be a function');
        }
        // Hack to support set arity
        if (listener.length >= 2) {
            g = function (e, next) {
                self.removeListener(event, g);
                void listener(e, next);
            };
        }
        else {
            g = function (e) {
                self.removeListener(event, g);
                void listener(e, g);
            };
        }
        self.on(event, g);
        return self;
    }
    first(event, listener) {
        let listeners = this._events[event] ?? [];
        // Contract
        if (typeof listener !== 'function') {
            throw new TypeError('listener must be a function');
        }
        // Listeners are not always an array
        if (!Array.isArray(listeners)) {
            ;
            this._events[event] = listeners = [listeners];
        }
        listeners.unshift(listener);
        return this;
    }
    before(event, target, listener) {
        return this.beforeOrAfter(event, target, listener);
    }
    after(event, target, listener) {
        return this.beforeOrAfter(event, target, listener, 'after');
    }
    beforeOrAfter(event, target, listener, beforeOrAfter) {
        let listeners = this._events[event] ?? [];
        let i;
        let index;
        const add = beforeOrAfter === 'after' ? 1 : 0;
        // Contract
        if (typeof listener !== 'function') {
            throw new TypeError('listener must be a function');
        }
        if (typeof target !== 'function') {
            throw new TypeError('target must be a function');
        }
        // Listeners are not always an array
        if (!Array.isArray(listeners)) {
            ;
            this._events[event] = listeners = [listeners];
        }
        index = listeners.length;
        for (i = listeners.length; i--;) {
            if (listeners[i] === target) {
                index = i + add;
                break;
            }
        }
        listeners.splice(index, 0, listener);
        return this;
    }
    on(event, listener) {
        return super.on(event, listener);
    }
    addListener(event, listener) {
        return super.addListener(event, listener);
    }
    prependListener(event, listener) {
        return super.prependListener(event, listener);
    }
    prependOnceListener(event, listener) {
        return super.prependOnceListener(event, listener);
    }
    removeAllListeners(event) {
        return super.removeAllListeners(event);
    }
    removeListener(event, listener) {
        return super.removeListener(event, listener);
    }
    eventNames() {
        return super.eventNames();
    }
    listeners(event) {
        return super.listeners(event);
    }
    listenerCount(event) {
        return super.listenerCount(event);
    }
    getMaxListeners() {
        return super.getMaxListeners();
    }
    setMaxListeners(maxListeners) {
        return super.setMaxListeners(maxListeners);
    }
}
exports.AsyncEventEmitter = AsyncEventEmitter;
//# sourceMappingURL=asyncEventEmitter.js.map