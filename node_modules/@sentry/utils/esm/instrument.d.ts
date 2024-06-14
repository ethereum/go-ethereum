/** Object describing handler that will be triggered for a given `type` of instrumentation */
interface InstrumentHandler {
    type: InstrumentHandlerType;
    callback: InstrumentHandlerCallback;
}
declare type InstrumentHandlerType = 'console' | 'dom' | 'fetch' | 'history' | 'sentry' | 'xhr' | 'error' | 'unhandledrejection';
declare type InstrumentHandlerCallback = (data: any) => void;
/**
 * Add handler that will be called when given type of instrumentation triggers.
 * Use at your own risk, this might break without changelog notice, only used internally.
 * @hidden
 */
export declare function addInstrumentationHandler(handler: InstrumentHandler): void;
export {};
//# sourceMappingURL=instrument.d.ts.map