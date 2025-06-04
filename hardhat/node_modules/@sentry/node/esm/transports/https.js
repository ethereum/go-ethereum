import { __extends } from "tslib";
import { SentryError } from '@sentry/utils';
import * as https from 'https';
import { BaseTransport } from './base';
/** Node https module transport */
var HTTPSTransport = /** @class */ (function (_super) {
    __extends(HTTPSTransport, _super);
    /** Create a new instance and set this.agent */
    function HTTPSTransport(options) {
        var _this = _super.call(this, options) || this;
        _this.options = options;
        var proxy = options.httpsProxy || options.httpProxy || process.env.https_proxy || process.env.http_proxy;
        _this.module = https;
        _this.client = proxy
            ? new (require('https-proxy-agent'))(proxy)
            : new https.Agent({ keepAlive: false, maxSockets: 30, timeout: 2000 });
        return _this;
    }
    /**
     * @inheritDoc
     */
    HTTPSTransport.prototype.sendEvent = function (event) {
        if (!this.module) {
            throw new SentryError('No module available in HTTPSTransport');
        }
        return this._sendWithModule(this.module, event);
    };
    return HTTPSTransport;
}(BaseTransport));
export { HTTPSTransport };
//# sourceMappingURL=https.js.map