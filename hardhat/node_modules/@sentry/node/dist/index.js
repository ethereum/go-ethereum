Object.defineProperty(exports, "__esModule", { value: true });
var tslib_1 = require("tslib");
var types_1 = require("@sentry/types");
exports.Severity = types_1.Severity;
exports.Status = types_1.Status;
var core_1 = require("@sentry/core");
exports.addGlobalEventProcessor = core_1.addGlobalEventProcessor;
exports.addBreadcrumb = core_1.addBreadcrumb;
exports.captureException = core_1.captureException;
exports.captureEvent = core_1.captureEvent;
exports.captureMessage = core_1.captureMessage;
exports.configureScope = core_1.configureScope;
exports.getHubFromCarrier = core_1.getHubFromCarrier;
exports.getCurrentHub = core_1.getCurrentHub;
exports.Hub = core_1.Hub;
exports.makeMain = core_1.makeMain;
exports.Scope = core_1.Scope;
exports.startTransaction = core_1.startTransaction;
exports.setContext = core_1.setContext;
exports.setExtra = core_1.setExtra;
exports.setExtras = core_1.setExtras;
exports.setTag = core_1.setTag;
exports.setTags = core_1.setTags;
exports.setUser = core_1.setUser;
exports.withScope = core_1.withScope;
var backend_1 = require("./backend");
exports.NodeBackend = backend_1.NodeBackend;
var client_1 = require("./client");
exports.NodeClient = client_1.NodeClient;
var sdk_1 = require("./sdk");
exports.defaultIntegrations = sdk_1.defaultIntegrations;
exports.init = sdk_1.init;
exports.lastEventId = sdk_1.lastEventId;
exports.flush = sdk_1.flush;
exports.close = sdk_1.close;
var version_1 = require("./version");
exports.SDK_NAME = version_1.SDK_NAME;
exports.SDK_VERSION = version_1.SDK_VERSION;
var core_2 = require("@sentry/core");
var hub_1 = require("@sentry/hub");
var domain = require("domain");
var Handlers = require("./handlers");
exports.Handlers = Handlers;
var NodeIntegrations = require("./integrations");
var Transports = require("./transports");
exports.Transports = Transports;
var INTEGRATIONS = tslib_1.__assign(tslib_1.__assign({}, core_2.Integrations), NodeIntegrations);
exports.Integrations = INTEGRATIONS;
// We need to patch domain on the global __SENTRY__ object to make it work for node in cross-platform packages like
// @sentry/hub. If we don't do this, browser bundlers will have troubles resolving `require('domain')`.
var carrier = hub_1.getMainCarrier();
if (carrier.__SENTRY__) {
    carrier.__SENTRY__.extensions = carrier.__SENTRY__.extensions || {};
    carrier.__SENTRY__.extensions.domain = carrier.__SENTRY__.extensions.domain || domain;
}
//# sourceMappingURL=index.js.map