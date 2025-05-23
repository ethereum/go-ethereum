"use strict";
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getEncodedEip712Data = void 0;
__exportStar(require("./api/errors_api.js"), exports);
__exportStar(require("./api/events_api.js"), exports);
__exportStar(require("./api/functions_api.js"), exports);
__exportStar(require("./api/logs_api.js"), exports);
__exportStar(require("./api/parameters_api.js"), exports);
__exportStar(require("./utils.js"), exports);
__exportStar(require("./decode_contract_error_data.js"), exports);
var eip_712_js_1 = require("./eip_712.js");
Object.defineProperty(exports, "getEncodedEip712Data", { enumerable: true, get: function () { return eip_712_js_1.getMessage; } });
//# sourceMappingURL=index.js.map