"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.prettierOutputTransformer = void 0;
const prettierOutputTransformer = (output, { prettier }, config) => {
    const prettierCfg = { ...(config.prettier || {}), parser: 'typescript' };
    return prettier.format(output, prettierCfg);
};
exports.prettierOutputTransformer = prettierOutputTransformer;
//# sourceMappingURL=prettier.js.map