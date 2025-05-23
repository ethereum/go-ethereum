const SolidityParser = require('@solidity-parser/parser');
const path = require('path');

const Injector = require('./injector');
const preprocess = require('./preprocessor');
const parse = require('./parse');

/**
 * Top level controller for the instrumentation sequence. Also hosts the instrumentation data map
 * which the vm step listener writes its output to. This only needs to be instantiated once
 * per coverage run.
 */
class Instrumenter {

  constructor(config={}){
    this.instrumentationData = {};
    this.injector = new Injector(config.viaIR);
    this.modifierWhitelist = config.modifierWhitelist || [];
    this.enabled = {
      statements: (config.measureStatementCoverage === false) ? false : true,
      functions: (config.measureFunctionCoverage === false) ? false: true,
      modifiers: (config.measureModifierCoverage === false) ? false: true,
      branches: (config.measureBranchCoverage === false) ? false: true,
      lines: (config.measureLineCoverage === false) ? false: true
    };
  }

  _isRootNode(node){
    return (node.type === 'ContractDefinition' ||
            node.type === 'LibraryDefinition'  ||
            node.type === 'InterfaceDefinition');
  }

  _initializeCoverageFields(contract){
    contract.runnableLines = [];
    contract.fnMap = {};
    contract.fnId = 0;
    contract.branchMap = {};
    contract.branchId = 0;
    contract.statementMap = {};
    contract.statementId = 0;
    contract.injectionPoints = {};
  }


   /**
    * Per `contractSource`:
    * - wraps any unbracketed singleton consequents of if, for, while stmts (preprocessor.js)
    * - walks the file's AST, creating an instrumentation map (parse.js, registrar.js)
    * - injects `instrumentation` solidity statements into the target solidity source (injector.js)
    *
    * @param  {String} contractSource  solidity source code
    * @param  {String} fileName        absolute path to source file
    * @return {Object}                 instrumented `contract` object
    * {
    *   contract: instrumented solidity source code,
    *   contractName: contract name,
    *   runnableLines: integer
    * }
    *
    */
  instrument(contractSource, fileName) {
    const contract = {};

    this.injector.resetModifierMapping();
    parse.configure(this.enabled, this.modifierWhitelist);

    contract.source = contractSource;
    contract.instrumented = contractSource;

    this._initializeCoverageFields(contract);

    // First, we run over the original contract to get the source mapping.
    let ast = SolidityParser.parse(contract.source, {loc: true, range: true});

    parse[ast.type](contract, ast);
    const retValue = JSON.parse(JSON.stringify(contract)); // Possibly apotropaic.

    // Now, we reset almost everything and use the preprocessor to increase our effectiveness.
    this._initializeCoverageFields(contract);
    contract.instrumented = preprocess(contract.source);

    // Walk the AST, recording injection points
    ast = SolidityParser.parse(contract.instrumented, {loc: true, range: true});

    const root = ast.children.filter(node => this._isRootNode(node));

    // Handle contracts which only contain import statements
    contract.contractName = (root.length) ? root[0].name : null;

    contract.finalParse = true;
    parse[ast.type](contract, ast);

    // We have to iterate through these points in descending order
    const sortedPoints = Object.keys(contract.injectionPoints).sort((a, b) => b - a);

    sortedPoints.forEach(injectionPoint => {

      // Line instrumentation has to happen first
      contract.injectionPoints[injectionPoint].sort((a, b) => {
        const injections = [
          'injectBranch',
          'injectOpenParen',
          'injectOrFalse',
          'injectAndTrue',
          'injectEmptyBranch',
          'injectLine'
        ];
        return injections.indexOf(b.type) - injections.indexOf(a.type);
      });

      contract.injectionPoints[injectionPoint].forEach(injection => {
        this.injector[injection.type](
          contract,
          fileName,
          injectionPoint,
          injection,
          this.instrumentationData
        );
      });
    });

    retValue.runnableLines = contract.runnableLines;
    retValue.contract = contract.instrumented;
    retValue.contractName = contract.contractName;
    return retValue;
  }
}

module.exports = Instrumenter;
