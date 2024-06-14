"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.LazyInitializationProviderAdapter = void 0;
const events_1 = require("events");
const errors_1 = require("../errors");
const errors_list_1 = require("../errors-list");
/**
 * A class that delays the (async) creation of its internal provider until the first call
 * to a JSON RPC method via request/send/sendAsync or the init method is called.
 */
class LazyInitializationProviderAdapter {
    constructor(_providerFactory) {
        this._providerFactory = _providerFactory;
        this._emitter = new events_1.EventEmitter();
    }
    /**
     * Gets the internal wrapped provider.
     * Using it directly is discouraged and should be done with care,
     * use the public methods from the class like `request` and all event emitter methods instead
     */
    get _wrapped() {
        if (this.provider === undefined) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.UNINITIALIZED_PROVIDER);
        }
        return this.provider;
    }
    async init() {
        if (this.provider === undefined) {
            if (this._initializingPromise === undefined) {
                this._initializingPromise = this._providerFactory();
            }
            this.provider = await this._initializingPromise;
        }
        return this.provider;
    }
    // Provider methods
    async request(args) {
        const provider = await this._getOrInitProvider();
        return provider.request(args);
    }
    async send(method, params) {
        const provider = await this._getOrInitProvider();
        return provider.send(method, params);
    }
    sendAsync(payload, callback) {
        this._getOrInitProvider().then((provider) => {
            provider.sendAsync(payload, callback);
        }, (e) => {
            callback(e, null);
        });
    }
    // EventEmitter methods
    addListener(event, listener) {
        this._getEmitter().addListener(event, listener);
        return this;
    }
    on(event, listener) {
        this._getEmitter().on(event, listener);
        return this;
    }
    once(event, listener) {
        this._getEmitter().once(event, listener);
        return this;
    }
    prependListener(event, listener) {
        this._getEmitter().prependListener(event, listener);
        return this;
    }
    prependOnceListener(event, listener) {
        this._getEmitter().prependOnceListener(event, listener);
        return this;
    }
    removeListener(event, listener) {
        this._getEmitter().removeListener(event, listener);
        return this;
    }
    off(event, listener) {
        this._getEmitter().off(event, listener);
        return this;
    }
    removeAllListeners(event) {
        this._getEmitter().removeAllListeners(event);
        return this;
    }
    setMaxListeners(n) {
        this._getEmitter().setMaxListeners(n);
        return this;
    }
    getMaxListeners() {
        return this._getEmitter().getMaxListeners();
    }
    // disable ban-types to satisfy the EventEmitter interface
    // eslint-disable-next-line @typescript-eslint/ban-types
    listeners(event) {
        return this._getEmitter().listeners(event);
    }
    // disable ban-types to satisfy the EventEmitter interface
    // eslint-disable-next-line @typescript-eslint/ban-types
    rawListeners(event) {
        return this._getEmitter().rawListeners(event);
    }
    emit(event, ...args) {
        return this._getEmitter().emit(event, ...args);
    }
    eventNames() {
        return this._getEmitter().eventNames();
    }
    listenerCount(type) {
        return this._getEmitter().listenerCount(type);
    }
    _getEmitter() {
        return this.provider === undefined ? this._emitter : this.provider;
    }
    async _getOrInitProvider() {
        // This is here to avoid multiple calls to send async stacking and re-creating the provider
        // over and over again. It shouldn't run for request or send
        if (this._initializingPromise !== undefined) {
            await this._initializingPromise;
        }
        if (this.provider === undefined) {
            this.provider = await this.init();
            // Copy any event emitter events before initialization over to the provider
            const recordedEvents = this._emitter.eventNames();
            for (const event of recordedEvents) {
                const listeners = this._emitter.rawListeners(event);
                for (const listener of listeners) {
                    this.provider.on(event, listener);
                    this._emitter.removeListener(event, listener);
                }
            }
            this.provider.setMaxListeners(this._emitter.getMaxListeners());
        }
        return this.provider;
    }
}
exports.LazyInitializationProviderAdapter = LazyInitializationProviderAdapter;
//# sourceMappingURL=lazy-initialization.js.map