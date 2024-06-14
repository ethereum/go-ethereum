/**
 * Methods in this file walk the AST and call the instrumenter
 * functions where appropriate, which determine where to inject events.
 * (Listed in alphabetical order)
 */
const semver = require('semver');
const Registrar = require('./registrar');
const register = new Registrar();

const FILE_SCOPED_ID = "fileScopedId";
const parse = {};

// Utilities
parse.configure = function(_enabled, _whitelist){
  register.enabled = Object.assign(register.enabled, _enabled);
  register.modifierWhitelist = _whitelist;
}

// Nodes
parse.AssignmentExpression = function(contract, expression) {
  register.statement(contract, expression);
};

parse.Block = function(contract, expression) {
  for (let x = 0; x < expression.statements.length; x++) {
    register.line(contract, expression.statements[x]);
    parse[expression.statements[x].type] &&
    parse[expression.statements[x].type](contract, expression.statements[x]);
  }
};

parse.BinaryOperation = function(contract, expression, skipStatementRegistry) {
  // Free-floating ternary conditional
  if (expression.left && expression.left.type === 'Conditional'){
    parse[expression.left.type](contract, expression.left, true);

  // Ternary conditional assignment
  } else if (expression.right && expression.right.type === 'Conditional'){
    parse[expression.right.type](contract, expression.right, true);

  // Regular binary operation
  } else if(!skipStatementRegistry){
    // noop

  // LogicalOR condition search...
  } else {
    parse[expression.left.type] &&
    parse[expression.left.type](contract, expression.left, true);

    parse[expression.right.type] &&
    parse[expression.right.type](contract, expression.right, true);

    if (expression.operator === '||'){
      register.logicalOR(contract, expression);
    }
  }
}

parse.TupleExpression = function(contract, expression, skipStatementRegistry) {
  expression.components.forEach(component => {
    parse[component.type] &&
    parse[component.type](contract, component, skipStatementRegistry);
  });
}

parse.FunctionCall = function(contract, expression, skipStatementRegistry) {
  // In any given chain of call expressions, only the last one will fail this check.
  // This makes sure we don't instrument a chain of expressions multiple times.
  if (expression.expression.type !== 'FunctionCall') {

    // Don't register sub-expressions (like intermediate method calls)
    if (!skipStatementRegistry){
      register.statement(contract, expression);
    }

    if (expression.expression.name === 'require') {
      register.requireBranch(contract, expression);
      expression.arguments.forEach(arg => {
        parse[arg.type] && parse[arg.type](contract, arg, true);
      });
    }
    parse[expression.expression.type] &&
    parse[expression.expression.type](contract, expression.expression);
  } else {
    parse[expression.expression.type] &&
    parse[expression.expression.type](contract, expression.expression);
  }
};

parse.Conditional = function(contract, expression, skipStatementRegistry) {
  parse[expression.condition.type] &&
  parse[expression.condition.type](contract, expression.condition, true);

  if (expression.trueExpression && expression.trueExpression.type === 'Conditional'){
    parse[expression.trueExpression.type](contract, expression.trueExpression, true);
  }

  if (expression.falseExpression && expression.falseExpression.type === 'Conditional'){
    parse[expression.falseExpression.type](contract, expression.falseExpression, true);
  }

  register.conditional(contract, expression);
};

parse.ContractDefinition = function(contract, expression) {
  parse.ContractOrLibraryStatement(contract, expression);
};

parse.ContractOrLibraryStatement = function(contract, expression) {

  // We need to define a method to pass coverage hashes into at top of each contract.
  // This lets us get a fresh stack for the hash and avoid stack-too-deep errors.
  if (expression.kind !== 'interface'){
    let start = 0;

    // It's possible a base contract will have constructor string arg
    // which contains an open curly brace. Skip ahead pass the bases...
    if (expression.baseContracts && expression.baseContracts.length){
      for (let base of expression.baseContracts ){
        if (base.range[1] > start){
          start = base.range[1];
        }
      }
    } else {
      start = expression.range[0];
    }

    const end = contract.instrumented.slice(start).indexOf('{') + 1;
    const loc = start + end;;

    contract.contractName = expression.name;

    (contract.injectionPoints[loc])
      ? contract.injectionPoints[loc].push({ type: 'injectHashMethod', contractName: expression.name})
      : contract.injectionPoints[loc] = [{ type: 'injectHashMethod', contractName: expression.name}];
  }

  if (expression.subNodes) {
    // Set flag to allow alternate cov id generation of file-level fn defs
    contract.isContractScoped = true;
    expression.subNodes.forEach(construct => {
      parse[construct.type] &&
      parse[construct.type](contract, construct);
    });
    // Unset flag...
    contract.isContractScoped = false;
  }
};

parse.EmitStatement = function(contract, expression){
  register.statement(contract, expression);
};

parse.ExpressionStatement = function(contract, content) {
  parse[content.expression.type] &&
  parse[content.expression.type](contract, content.expression);
};

parse.ForStatement = function(contract, expression) {
  register.statement(contract, expression);
  parse[expression.body.type] &&
  parse[expression.body.type](contract, expression.body);
};

parse.FunctionDefinition = function(contract, expression) {
  // Use generic name component to generate the hash for cov fn ids
  // if we're not inside a contract
  let tempContractName = contract.contractName;
  if (!contract.isContractScoped){
    contract.contractName = FILE_SCOPED_ID;
  }

  parse.Modifiers(contract, expression.modifiers);
  if (expression.body) {
    // Skip fn & statement instrumentation for `receive` methods to
    // minimize gas distortion
    (expression.name === null && expression.isReceiveEther)
      ? register.trackStatements = false
      : register.functionDeclaration(contract, expression);

    parse[expression.body.type] &&
    parse[expression.body.type](contract, expression.body);
    register.trackStatements = true;
  }
  // Reset contractName in case it was changed for file scoped methods
  contract.contractName = tempContractName;
};

parse.IfStatement = function(contract, expression) {
  register.statement(contract, expression);
  register.ifStatement(contract, expression);

  parse[expression.condition.type] &&
  parse[expression.condition.type](contract, expression.condition, true);

  parse[expression.trueBody.type] &&
  parse[expression.trueBody.type](contract, expression.trueBody);

  if (expression.falseBody) {
    parse[expression.falseBody.type] &&
    parse[expression.falseBody.type](contract, expression.falseBody);
  }
};

// TODO: Investigate Node structure
/*parse.MemberAccess = function(contract, expression) {
  parse[expression.object.type] &&
  parse[expression.object.type](contract, expression.object);
};*/

parse.Modifiers = function(contract, modifiers) {
  if (modifiers) {
    modifiers.forEach(modifier => {
      parse[modifier.type] && parse[modifier.type](contract, modifier);
    });
  }
};

parse.ModifierDefinition = function(contract, expression) {
  if (expression.body) {
    register.functionDeclaration(contract, expression);
    parse[expression.body.type] &&
    parse[expression.body.type](contract, expression.body);
  }
};

parse.NewExpression = function(contract, expression) {
  parse[expression.typeName.type] &&
  parse[expression.typeName.type](contract, expression.typeName);
};

parse.PragmaDirective = function(contract, expression){
  let minVersion;

  // Some solidity pragmas crash semver (ex: ABIEncoderV2)
  try {
    minVersion = semver.minVersion(expression.value);
  } catch(e){
    return;
  }

  // pragma abicoder v2 passes the semver test above but needs to be ignored
  if (expression.name === 'abicoder'){
    return
  }

  // From solc >=0.7.4, every file should have instrumentation methods
  // defined at the file level which file scoped fns can use. (Make sure we only do this
  // once - flattened contracts have multiple pragma statements)
  if (semver.lt("0.7.3", minVersion) && contract.finalParse && !contract.fileLevelFinished){
    const start = expression.range[0];
    const end = contract.instrumented.slice(start).indexOf(';') + 1;
    const loc = start + end;

    const injectionObject = {
      type: 'injectHashMethod',
      contractName: FILE_SCOPED_ID,
      isFileScoped: true
    };

    contract.injectionPoints[loc] = [injectionObject];
    contract.fileLevelFinished = true;
  }
}

parse.SourceUnit = function(contract, expression) {
  expression.children.forEach(construct => {
    parse[construct.type] &&
    parse[construct.type](contract, construct);
  });
};

parse.ReturnStatement = function(contract, expression) {
  register.statement(contract, expression);

  expression.expression &&
  parse[expression.expression.type] &&
  parse[expression.expression.type](contract, expression.expression, true);
};

// TODO:Investigate node structure
/*parse.UnaryOperation = function(contract, expression) {
  parse[subExpression.argument.type] &&
  parse[subExpression.argument.type](contract, expression.argument);
};*/

parse.TryStatement = function(contract, expression) {
  register.statement(contract, expression);
  parse[expression.body.type] &&
  parse[expression.body.type](contract, expression.body);

  for (let x = 0; x < expression.catchClauses.length; x++) {
    parse[expression.catchClauses[x].body.type] &&
    parse[expression.catchClauses[x].body.type](contract, expression.catchClauses[x].body);
  }
};

parse.UncheckedStatement = function(contract, expression) {
  parse[expression.block.type] &&
  parse[expression.block.type](contract, expression.block);
}

parse.UsingStatement = function (contract, expression) {
  parse[expression.for.type] &&
  parse[expression.for.type](contract, expression.for);
};

parse.VariableDeclarationStatement = function (contract, expression) {
  if (expression.initialValue && expression.initialValue.type === 'Conditional'){
    parse[expression.initialValue.type](contract, expression.initialValue, true)
  }
  register.statement(contract, expression);
};

parse.WhileStatement = function (contract, expression) {
  register.statement(contract, expression);

  parse[expression.condition.type] &&
  parse[expression.condition.type](contract, expression.condition, true);

  parse[expression.body.type] &&
  parse[expression.body.type](contract, expression.body);
};

module.exports = parse;
