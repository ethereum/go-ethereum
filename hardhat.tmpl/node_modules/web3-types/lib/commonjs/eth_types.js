"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.HardforksOrdered = exports.BlockTags = void 0;
var BlockTags;
(function (BlockTags) {
    BlockTags["EARLIEST"] = "earliest";
    BlockTags["LATEST"] = "latest";
    BlockTags["PENDING"] = "pending";
    BlockTags["SAFE"] = "safe";
    BlockTags["FINALIZED"] = "finalized";
    BlockTags["COMMITTED"] = "committed";
})(BlockTags || (exports.BlockTags = BlockTags = {}));
// This list of hardforks is expected to be in order
// keep this in mind when making changes to it
var HardforksOrdered;
(function (HardforksOrdered) {
    HardforksOrdered["chainstart"] = "chainstart";
    HardforksOrdered["frontier"] = "frontier";
    HardforksOrdered["homestead"] = "homestead";
    HardforksOrdered["dao"] = "dao";
    HardforksOrdered["tangerineWhistle"] = "tangerineWhistle";
    HardforksOrdered["spuriousDragon"] = "spuriousDragon";
    HardforksOrdered["byzantium"] = "byzantium";
    HardforksOrdered["constantinople"] = "constantinople";
    HardforksOrdered["petersburg"] = "petersburg";
    HardforksOrdered["istanbul"] = "istanbul";
    HardforksOrdered["muirGlacier"] = "muirGlacier";
    HardforksOrdered["berlin"] = "berlin";
    HardforksOrdered["london"] = "london";
    HardforksOrdered["altair"] = "altair";
    HardforksOrdered["arrowGlacier"] = "arrowGlacier";
    HardforksOrdered["grayGlacier"] = "grayGlacier";
    HardforksOrdered["bellatrix"] = "bellatrix";
    HardforksOrdered["merge"] = "merge";
    HardforksOrdered["capella"] = "capella";
    HardforksOrdered["shanghai"] = "shanghai";
})(HardforksOrdered || (exports.HardforksOrdered = HardforksOrdered = {}));
//# sourceMappingURL=eth_types.js.map