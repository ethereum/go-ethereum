Object.defineProperty(exports, "__esModule", { value: true });
var hub_1 = require("@sentry/hub");
var utils_1 = require("@sentry/utils");
/**
 * Internal function to create a new SDK client instance. The client is
 * installed and then bound to the current scope.
 *
 * @param clientClass The client class to instantiate.
 * @param options Options to pass to the client.
 */
function initAndBind(clientClass, options) {
    if (options.debug === true) {
        utils_1.logger.enable();
    }
    var hub = hub_1.getCurrentHub();
    var client = new clientClass(options);
    hub.bindClient(client);
}
exports.initAndBind = initAndBind;
//# sourceMappingURL=sdk.js.map