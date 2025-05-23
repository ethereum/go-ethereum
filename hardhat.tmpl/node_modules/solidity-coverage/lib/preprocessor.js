const SolidityParser = require('@solidity-parser/parser');

const crRegex = /[\r\n ]+$/g;
const OPEN = '{';
const CLOSE = '}';

/**
 * Inserts an open or close brace e.g. `{` or `}` at specified position in solidity source
 *
 * @param  {String} contract solidity source
 * @param  {Object} item     AST node to bracket
 * @param  {Number} offset   tracks the number of previously inserted braces
 * @return {String}          contract
 */
function insertBrace(contract, item, offset) {
  return contract.slice(0,item.pos + offset) + item.type + contract.slice(item.pos + offset)
}

/**
 * Locates unbracketed singleton statements attached to if, else, for and while statements
 * and brackets them. Instrumenter needs to inject events at these locations and having
 * them pre-bracketed simplifies the process.
 *
 * @param  {String} contract solidity source code
 * @return {String}          modified solidity source code
 */
function preprocess(contract) {

  try {
    const ast = SolidityParser.parse(contract, { range: true });
    insertions = [];

    SolidityParser.visit(ast, {
      IfStatement: function(node) {
        if (node.trueBody.type !== 'Block') {
          insertions.push({type: OPEN, pos: node.trueBody.range[0]});
          insertions.push({type: CLOSE, pos: node.trueBody.range[1] + 1});
        }
        if ( node.falseBody && node.falseBody.type !== 'Block' ) {
          insertions.push({type: OPEN, pos: node.falseBody.range[0]});
          insertions.push({type: CLOSE, pos: node.falseBody.range[1] + 1});
        }
      },
      ForStatement: function(node){
        if (node.body.type !== 'Block'){
          insertions.push({type: OPEN, pos: node.body.range[0]});
          insertions.push({type: CLOSE, pos: node.body.range[1] + 1});
        }
      },
      WhileStatement: function(node){
        if (node.body.type !== 'Block'){
          insertions.push({type: OPEN, pos: node.body.range[0]});
          insertions.push({type: CLOSE, pos: node.body.range[1] + 1});
        }
      }
    })

    // Sort the insertion points.
    insertions.sort((a,b) => a.pos - b.pos);
    insertions.forEach((item, idx) => contract = insertBrace(contract, item, idx));

  } catch (err) {
    contract = err;
  }
  return contract;
};


module.exports = preprocess;
