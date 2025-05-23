export { Breadcrumb, BreadcrumbHint, Request, SdkInfo, Event, EventHint, Exception, Response, Severity, StackFrame, Stacktrace, Status, Thread, User, } from '@sentry/types';
export { addGlobalEventProcessor, addBreadcrumb, captureException, captureEvent, captureMessage, configureScope, getHubFromCarrier, getCurrentHub, Hub, makeMain, Scope, startTransaction, setContext, setExtra, setExtras, setTag, setTags, setUser, withScope, } from '@sentry/core';
export { NodeBackend, NodeOptions } from './backend';
export { NodeClient } from './client';
export { defaultIntegrations, init, lastEventId, flush, close } from './sdk';
export { SDK_NAME, SDK_VERSION } from './version';
import { Integrations as CoreIntegrations } from '@sentry/core';
import * as Handlers from './handlers';
import * as NodeIntegrations from './integrations';
import * as Transports from './transports';
declare const INTEGRATIONS: {
    Console: typeof NodeIntegrations.Console;
    Http: typeof NodeIntegrations.Http;
    OnUncaughtException: typeof NodeIntegrations.OnUncaughtException;
    OnUnhandledRejection: typeof NodeIntegrations.OnUnhandledRejection;
    LinkedErrors: typeof NodeIntegrations.LinkedErrors;
    Modules: typeof NodeIntegrations.Modules;
    FunctionToString: typeof CoreIntegrations.FunctionToString;
    InboundFilters: typeof CoreIntegrations.InboundFilters;
};
export { INTEGRATIONS as Integrations, Transports, Handlers };
//# sourceMappingURL=index.d.ts.map