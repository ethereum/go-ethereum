import { __assign } from "tslib";
import { BrowserTracing } from './browser';
import { addExtensionMethods } from './hubextensions';
import * as TracingIntegrations from './integrations';
var Integrations = __assign(__assign({}, TracingIntegrations), { BrowserTracing: BrowserTracing });
export { Integrations };
export { Span } from './span';
export { Transaction } from './transaction';
export { registerRequestInstrumentation, defaultRequestInstrumentationOptions, } from './browser';
export { SpanStatus } from './spanstatus';
export { IdleTransaction } from './idletransaction';
export { startIdleTransaction } from './hubextensions';
// We are patching the global object with our hub extension methods
addExtensionMethods();
export { addExtensionMethods };
export { extractTraceparentData, getActiveTransaction, hasTracingEnabled, stripUrlQueryAndFragment, TRACEPARENT_REGEXP, } from './utils';
//# sourceMappingURL=index.js.map