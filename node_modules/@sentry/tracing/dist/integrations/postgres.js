Object.defineProperty(exports, "__esModule", { value: true });
var utils_1 = require("@sentry/utils");
/** Tracing integration for node-postgres package */
var Postgres = /** @class */ (function () {
    function Postgres() {
        /**
         * @inheritDoc
         */
        this.name = Postgres.id;
    }
    /**
     * @inheritDoc
     */
    Postgres.prototype.setupOnce = function (_, getCurrentHub) {
        var client;
        try {
            var pgModule = utils_1.dynamicRequire(module, 'pg');
            client = pgModule.Client;
        }
        catch (e) {
            utils_1.logger.error('Postgres Integration was unable to require `pg` package.');
            return;
        }
        /**
         * function (query, callback) => void
         * function (query, params, callback) => void
         * function (query) => Promise
         * function (query, params) => Promise
         */
        utils_1.fill(client.prototype, 'query', function (orig) {
            return function (config, values, callback) {
                var _a, _b;
                var scope = getCurrentHub().getScope();
                var parentSpan = (_a = scope) === null || _a === void 0 ? void 0 : _a.getSpan();
                var span = (_b = parentSpan) === null || _b === void 0 ? void 0 : _b.startChild({
                    description: typeof config === 'string' ? config : config.text,
                    op: "db",
                });
                if (typeof callback === 'function') {
                    return orig.call(this, config, values, function (err, result) {
                        var _a;
                        (_a = span) === null || _a === void 0 ? void 0 : _a.finish();
                        callback(err, result);
                    });
                }
                if (typeof values === 'function') {
                    return orig.call(this, config, function (err, result) {
                        var _a;
                        (_a = span) === null || _a === void 0 ? void 0 : _a.finish();
                        values(err, result);
                    });
                }
                return orig.call(this, config, values).then(function (res) {
                    var _a;
                    (_a = span) === null || _a === void 0 ? void 0 : _a.finish();
                    return res;
                });
            };
        });
    };
    /**
     * @inheritDoc
     */
    Postgres.id = 'Postgres';
    return Postgres;
}());
exports.Postgres = Postgres;
//# sourceMappingURL=postgres.js.map