Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var types_1 = require("@sentry/types");
exports.Severity = types_1.Severity;
exports.Status = types_1.Status;
var browser_1 = require("@sentry/browser");
exports.addGlobalEventProcessor = browser_1.addGlobalEventProcessor;
exports.addBreadcrumb = browser_1.addBreadcrumb;
exports.captureException = browser_1.captureException;
exports.captureEvent = browser_1.captureEvent;
exports.captureMessage = browser_1.captureMessage;
exports.configureScope = browser_1.configureScope;
exports.getHubFromCarrier = browser_1.getHubFromCarrier;
exports.getCurrentHub = browser_1.getCurrentHub;
exports.Hub = browser_1.Hub;
exports.Scope = browser_1.Scope;
exports.setContext = browser_1.setContext;
exports.setExtra = browser_1.setExtra;
exports.setExtras = browser_1.setExtras;
exports.setTag = browser_1.setTag;
exports.setTags = browser_1.setTags;
exports.setUser = browser_1.setUser;
exports.startTransaction = browser_1.startTransaction;
exports.Transports = browser_1.Transports;
exports.withScope = browser_1.withScope;
var browser_2 = require("@sentry/browser");
exports.BrowserClient = browser_2.BrowserClient;
var browser_3 = require("@sentry/browser");
exports.defaultIntegrations = browser_3.defaultIntegrations;
exports.forceLoad = browser_3.forceLoad;
exports.init = browser_3.init;
exports.lastEventId = browser_3.lastEventId;
exports.onLoad = browser_3.onLoad;
exports.showReportDialog = browser_3.showReportDialog;
exports.flush = browser_3.flush;
exports.close = browser_3.close;
exports.wrap = browser_3.wrap;
var browser_4 = require("@sentry/browser");
exports.SDK_NAME = browser_4.SDK_NAME;
exports.SDK_VERSION = browser_4.SDK_VERSION;
var browser_5 = require("@sentry/browser");
var utils_1 = require("@sentry/utils");
var browser_6 = require("./browser");
var hubextensions_1 = require("./hubextensions");
exports.addExtensionMethods = hubextensions_1.addExtensionMethods;
var span_1 = require("./span");
exports.Span = span_1.Span;
var windowIntegrations = {};
// This block is needed to add compatibility with the integrations packages when used with a CDN
var _window = utils_1.getGlobalObject();
if (_window.Sentry && _window.Sentry.Integrations) {
    windowIntegrations = _window.Sentry.Integrations;
}
var INTEGRATIONS = tslib_1.__assign(tslib_1.__assign(tslib_1.__assign({}, windowIntegrations), browser_5.Integrations), { BrowserTracing: browser_6.BrowserTracing });
exports.Integrations = INTEGRATIONS;
// We are patching the global object with our hub extension methods
hubextensions_1.addExtensionMethods();
//# sourceMappingURL=index.bundle.js.map