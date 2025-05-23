"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.hardforks = void 0;
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
const chainstart_js_1 = __importDefault(require("./chainstart.js"));
const dao_js_1 = __importDefault(require("./dao.js"));
const homestead_js_1 = __importDefault(require("./homestead.js"));
const tangerineWhistle_js_1 = __importDefault(require("./tangerineWhistle.js"));
const spuriousDragon_js_1 = __importDefault(require("./spuriousDragon.js"));
const byzantium_js_1 = __importDefault(require("./byzantium.js"));
const constantinople_js_1 = __importDefault(require("./constantinople.js"));
const petersburg_js_1 = __importDefault(require("./petersburg.js"));
const istanbul_js_1 = __importDefault(require("./istanbul.js"));
const muirGlacier_js_1 = __importDefault(require("./muirGlacier.js"));
const berlin_js_1 = __importDefault(require("./berlin.js"));
const london_js_1 = __importDefault(require("./london.js"));
const shanghai_js_1 = __importDefault(require("./shanghai.js"));
const arrowGlacier_js_1 = __importDefault(require("./arrowGlacier.js"));
const grayGlacier_js_1 = __importDefault(require("./grayGlacier.js"));
const mergeForkIdTransition_js_1 = __importDefault(require("./mergeForkIdTransition.js"));
const merge_js_1 = __importDefault(require("./merge.js"));
exports.hardforks = {
    chainstart: chainstart_js_1.default,
    homestead: homestead_js_1.default,
    dao: dao_js_1.default,
    tangerineWhistle: tangerineWhistle_js_1.default,
    spuriousDragon: spuriousDragon_js_1.default,
    byzantium: byzantium_js_1.default,
    constantinople: constantinople_js_1.default,
    petersburg: petersburg_js_1.default,
    istanbul: istanbul_js_1.default,
    muirGlacier: muirGlacier_js_1.default,
    berlin: berlin_js_1.default,
    london: london_js_1.default,
    shanghai: shanghai_js_1.default,
    arrowGlacier: arrowGlacier_js_1.default,
    grayGlacier: grayGlacier_js_1.default,
    mergeForkIdTransition: mergeForkIdTransition_js_1.default,
    merge: merge_js_1.default,
};
//# sourceMappingURL=index.js.map