import { __assign } from "tslib";
export { Severity, Status, } from '@sentry/types';
export { addGlobalEventProcessor, addBreadcrumb, captureException, captureEvent, captureMessage, configureScope, getHubFromCarrier, getCurrentHub, Hub, makeMain, Scope, startTransaction, setContext, setExtra, setExtras, setTag, setTags, setUser, withScope, } from '@sentry/core';
export { NodeBackend } from './backend';
export { NodeClient } from './client';
export { defaultIntegrations, init, lastEventId, flush, close } from './sdk';
export { SDK_NAME, SDK_VERSION } from './version';
import { Integrations as CoreIntegrations } from '@sentry/core';
import { getMainCarrier } from '@sentry/hub';
import * as domain from 'domain';
import * as Handlers from './handlers';
import * as NodeIntegrations from './integrations';
import * as Transports from './transports';
var INTEGRATIONS = __assign(__assign({}, CoreIntegrations), NodeIntegrations);
export { INTEGRATIONS as Integrations, Transports, Handlers };
// We need to patch domain on the global __SENTRY__ object to make it work for node in cross-platform packages like
// @sentry/hub. If we don't do this, browser bundlers will have troubles resolving `require('domain')`.
var carrier = getMainCarrier();
if (carrier.__SENTRY__) {
    carrier.__SENTRY__.extensions = carrier.__SENTRY__.extensions || {};
    carrier.__SENTRY__.extensions.domain = carrier.__SENTRY__.extensions.domain || domain;
}
//# sourceMappingURL=index.js.map