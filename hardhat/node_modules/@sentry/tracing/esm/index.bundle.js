import { __assign } from "tslib";
export { Severity, Status, } from '@sentry/types';
export { addGlobalEventProcessor, addBreadcrumb, captureException, captureEvent, captureMessage, configureScope, getHubFromCarrier, getCurrentHub, Hub, Scope, setContext, setExtra, setExtras, setTag, setTags, setUser, startTransaction, Transports, withScope, } from '@sentry/browser';
export { BrowserClient } from '@sentry/browser';
export { defaultIntegrations, forceLoad, init, lastEventId, onLoad, showReportDialog, flush, close, wrap, } from '@sentry/browser';
export { SDK_NAME, SDK_VERSION } from '@sentry/browser';
import { Integrations as BrowserIntegrations } from '@sentry/browser';
import { getGlobalObject } from '@sentry/utils';
import { BrowserTracing } from './browser';
import { addExtensionMethods } from './hubextensions';
export { Span } from './span';
var windowIntegrations = {};
// This block is needed to add compatibility with the integrations packages when used with a CDN
var _window = getGlobalObject();
if (_window.Sentry && _window.Sentry.Integrations) {
    windowIntegrations = _window.Sentry.Integrations;
}
var INTEGRATIONS = __assign(__assign(__assign({}, windowIntegrations), BrowserIntegrations), { BrowserTracing: BrowserTracing });
export { INTEGRATIONS as Integrations };
// We are patching the global object with our hub extension methods
addExtensionMethods();
export { addExtensionMethods };
//# sourceMappingURL=index.bundle.js.map