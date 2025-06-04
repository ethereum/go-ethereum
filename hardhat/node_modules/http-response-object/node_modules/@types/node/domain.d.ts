declare module 'domain' {
    import EventEmitter = require('events');

    class Domain extends EventEmitter implements NodeJS.Domain {
        run(fn: Function): void;
        add(emitter: EventEmitter): void;
        remove(emitter: EventEmitter): void;
        bind(cb: (err: Error, data: any) => any): any;
        intercept(cb: (data: any) => any): any;
        members: any[];
        enter(): void;
        exit(): void;
    }

    function create(): Domain;
}
