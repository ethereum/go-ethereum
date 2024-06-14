declare module 'events' {
    interface NodeEventTarget {
        once(event: string | symbol, listener: (...args: any[]) => void): this;
    }

    interface DOMEventTarget {
        addEventListener(event: string, listener: (...args: any[]) => void, opts?: { once: boolean }): any;
    }

    class EventEmitter extends NodeJS.EventEmitter {
        constructor();

        static once(emitter: NodeEventTarget, event: string | symbol): Promise<any[]>;
        static once(emitter: DOMEventTarget, event: string): Promise<any[]>;

        /** @deprecated since v4.0.0 */
        static listenerCount(emitter: NodeJS.EventEmitter, event: string | symbol): number;

        // TODO: This should be described using a static getter/setter pair:
        static defaultMaxListeners: number;
    }

    import internal = require('events');
    namespace EventEmitter {
        // Should just be `export { EventEmitter }`, but that doesn't work in TypeScript 3.4
        export { internal as EventEmitter };
    }

    export = EventEmitter;
}
