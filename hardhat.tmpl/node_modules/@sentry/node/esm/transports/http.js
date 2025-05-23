import { __extends } from "tslib";
import { SentryError } from '@sentry/utils';
import * as http from 'http';
import { BaseTransport } from './base';
/** Node http module transport */
var HTTPTransport = /** @class */ (function (_super) {
    __extends(HTTPTransport, _super);
    /** Create a new instance and set this.agent */
    function HTTPTransport(options) {
        var _this = _super.call(this, options) || this;
        _this.options = options;
        var proxy = options.httpProxy || process.env.http_proxy;
        _this.module = http;
        _this.client = proxy
            ? new (require('https-proxy-agent'))(proxy)
            : new http.Agent({ keepAlive: false, maxSockets: 30, timeout: 2000 });
        return _this;
    }
    /**
     * @inheritDoc
     */
    HTTPTransport.prototype.sendEvent = function (event) {
        if (!this.module) {
            throw new SentryError('No module available in HTTPTransport');
        }
        return this._sendWithModule(this.module, event);
    };
    return HTTPTransport;
}(BaseTransport));
export { HTTPTransport };
//# sourceMappingURL=http.js.map