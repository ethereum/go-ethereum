// This is an indirect file to import the tokens string
// It needs to be a js file so that tsc doesn't complain

if (typeof BROWSER !== "undefined") {
  module.exports = require('./antlr/Solidity.tokens')
} else {
  module.exports = require('fs')
    .readFileSync(require('path').join(__dirname, './antlr/Solidity.tokens'))
    .toString()
}
