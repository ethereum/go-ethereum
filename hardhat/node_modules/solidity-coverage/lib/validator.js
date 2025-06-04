const Validator = require('jsonschema').Validator;
const AppUI = require('./ui').AppUI;
const util = require('util')

Validator.prototype.customFormats.isFunction = function(input) {
  return typeof input === "function"
};

const configSchema = {
  id: "/solcoverjs",
  type: "object",
  properties: {

    client: {type: "object"},
    cwd:    {type: "string"},
    host:   {type: "string"},
    abiOutputPath:      {type: "string"},
    matrixOutputPath:   {type: "string"},
    matrixReporterPath: {type: "string"},
    port:                 {type: "number"},
    providerOptions:      {type: "object"},
    silent:               {type: "boolean"},
    autoLaunchServer:     {type: "boolean"},
    istanbulFolder:       {type: "string"},
    measureStatementCoverage: {type: "boolean"},
    measureFunctionCoverage:  {type: "boolean"},
    measureModifierCoverage:  {type: "boolean"},
    measureLineCoverage:      {type: "boolean"},
    measureBranchCoverage:    {type: "boolean"},

    // Hooks:
    onServerReady:        {type: "function", format: "isFunction"},
    onCompileComplete:    {type: "function", format: "isFunction"},
    onTestComplete:       {type: "function", format: "isFunction"},
    onIstanbulComplete:   {type: "function", format: "isFunction"},

    // Arrays
    skipFiles: {
      type: "array",
      items: {type: "string"}
    },

    istanbulReporter: {
      type: "array",
      items: {type: "string"}
    },

    modifierWhitelist: {
      type: "array",
      items: {type: "string"}
    }
  },
};

class ConfigValidator {
  constructor(){
    this.validator = new Validator();
    this.validator.addSchema(configSchema);
    this.ui = new AppUI();
  }

  validate(config){
    let result = this.validator.validate(config, configSchema);

    if (result.errors.length){
      let msg;
      const option = `"${result.errors[0].property.replace('instance.', '')}"`;

      (result.errors[0].argument === 'isFunction')
        ? msg = `${option} is not a function`
        : msg = `${option} ${result.errors[0].message}`;

      throw new Error(this.ui.generate('config-fail', [msg]));
    }

    return true;
  }
}

module.exports = ConfigValidator;