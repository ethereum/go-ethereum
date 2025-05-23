"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.generateStructTypes = void 0;
/* eslint-disable import/no-extraneous-dependencies */
const lodash_1 = require("lodash");
const common_1 = require("../common");
const types_1 = require("./types");
function generateStructTypes(structs) {
    const namedStructs = structs.filter((s) => !!s.structName);
    const namespaces = (0, lodash_1.groupBy)(namedStructs, (s) => s.structName.namespace);
    const exports = [];
    if ('undefined' in namespaces) {
        exports.push(namespaces['undefined'].map((s) => generateExports(s)).join('\n'));
        delete namespaces['undefined'];
    }
    for (const namespace of Object.keys(namespaces)) {
        exports.push(`\nexport declare namespace ${namespace} {
      ${namespaces[namespace].map((s) => generateExports(s)).join('\n')}
    }`);
    }
    return exports.join('\n');
}
exports.generateStructTypes = generateStructTypes;
function generateExports(struct) {
    const { identifier } = struct.structName;
    const inputName = `${identifier}${common_1.STRUCT_INPUT_POSTFIX}`;
    const outputName = `${identifier}${common_1.STRUCT_OUTPUT_POSTFIX}`;
    const inputType = (0, types_1.generateInputType)({ useStructs: false, includeLabelsInTupleTypes: true }, struct);
    const outputType = (0, types_1.generateOutputType)({ useStructs: false, includeLabelsInTupleTypes: true }, struct);
    return `
    export type ${inputName} = ${inputType}

    export type ${outputName} = ${outputType}
  `;
}
//# sourceMappingURL=structs.js.map