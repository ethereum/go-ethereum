'use strict';
const reqFrom = require('req-from');

module.exports = moduleId => reqFrom(process.cwd(), moduleId);
module.exports.silent = moduleId => reqFrom.silent(process.cwd(), moduleId);
