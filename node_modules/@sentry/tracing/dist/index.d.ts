import { BrowserTracing } from './browser';
import { addExtensionMethods } from './hubextensions';
import * as TracingIntegrations from './integrations';
declare const Integrations: {
    BrowserTracing: typeof BrowserTracing;
    Express: typeof TracingIntegrations.Express;
    Postgres: typeof TracingIntegrations.Postgres;
    Mysql: typeof TracingIntegrations.Mysql;
    Mongo: typeof TracingIntegrations.Mongo;
};
export { Integrations };
export { Span } from './span';
export { Transaction } from './transaction';
export { registerRequestInstrumentation, RequestInstrumentationOptions, defaultRequestInstrumentationOptions, } from './browser';
export { SpanStatus } from './spanstatus';
export { IdleTransaction } from './idletransaction';
export { startIdleTransaction } from './hubextensions';
export { addExtensionMethods };
export { extractTraceparentData, getActiveTransaction, hasTracingEnabled, stripUrlQueryAndFragment, TRACEPARENT_REGEXP, } from './utils';
//# sourceMappingURL=index.d.ts.map