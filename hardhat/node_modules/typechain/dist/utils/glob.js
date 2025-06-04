"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.glob = void 0;
const glob_1 = require("glob");
const lodash_1 = require("lodash");
function glob(cwd, patternsOrFiles, ignoreNodeModules = true) {
    const matches = patternsOrFiles.map((p) => (0, glob_1.sync)(p, ignoreNodeModules ? { ignore: 'node_modules/**', absolute: true, cwd } : { absolute: true, cwd }));
    return (0, lodash_1.uniq)((0, lodash_1.flatten)(matches));
}
exports.glob = glob;
//# sourceMappingURL=glob.js.map