const UI = require('./../../lib/ui').UI;

/**
 * Nomiclabs Plugin logging
 */
class PluginUI extends UI {
  constructor(log){
    super(log);

    this.flags = {
      testfiles:  `Path (or glob) defining a subset of tests to run`,

      testMatrix: `Generate a json object which maps which unit tests hit which lines of code.`,

      abi:        `Generate a json object which can be used to produce a unified diff of your ` +
                  `contracts public interface between two commits.`,

      solcoverjs: `Relative path from working directory to config. ` +
                  `Useful for monorepo packages that share settings.`,

      temp:       `Path to a disposable folder to store compilation artifacts in. ` +
                  `Useful when your test setup scripts include hard-coded paths to ` +
                  `a build directory.`,
    }
  }

  /**
   * Writes a formatted message via log
   * @param  {String}   kind  message selector
   * @param  {String[]} args  info to inject into template
   */
  report(kind, args=[]){
    const c = this.chalk;
    const ct = c.bold.green('>');
    const ds = c.bold.yellow('>');
    const w = ":warning:";

    const kinds = {

      'instr-skip':  `\n${c.bold('Coverage skipped for:')}` +
                     `\n${c.bold('=====================')}\n`,

      'compilation':  `\n${c.bold('Compilation:')}` +
                      `\n${c.bold('============')}\n`,

      'instr-skipped': `${ds} ${c.grey(args[0])}`,

      'hardhat-versions': `\n${c.bold('Version')}` +
                          `\n${c.bold('=======')}\n` +
                          `${ct} ${c.bold('solidity-coverage')}: v${args[0]}`,

      'hardhat-network': `\n${c.bold('Network Info')}` +
                         `\n${c.bold('============')}\n` +
                         `${ct} ${c.bold('HardhatEVM')}: v${args[0]}\n` +
                         `${ct} ${c.bold('network')}:    ${args[1]}\n`,

      'hardhat-viem': `\n${w}${c.red("  Coverage requires a special environment variable when used with 'hardhat-viem'  ")}${w}` +
                      `\n${c.red(    "====================================================================================")}`   +
                      `\n${c.bold(   "Please run the coverage command as:" )}` +
                      `\n${c(        "SOLIDITY_COVERAGE=true npx hardhat coverage")}` +
                      `\n${c.red(    "====================================================================================")}`
                      ,
    }

    this._write(kinds[kind]);
  }

  /**
   * Returns a formatted message. Useful for error message.
   * @param  {String}   kind  message selector
   * @param  {String[]} args  info to inject into template
   * @return {String}         message
   */
  generate(kind, args=[]){
    const c = this.chalk;
    const x = ":x:";

    const kinds = {
      'network-fail': `${c.red('--network cli flag is not supported for the coverage task. ')}` +
                      `${c.red('Beginning with v0.8.7, coverage must use the default "hardhat" network.')}`,

      'sources-fail': `${c.red('Cannot locate expected contract sources folder: ')} ${args[0]}`,

      'solcoverjs-fail': `${c.red('Could not load .solcover.js config file. ')}` +
                         `${c.red('This can happen if it has a syntax error or ')}` +
                         `${c.red('the path you specified for it is wrong.')}`,

      'tests-fail': `${x} ${c.bold(args[0])} ${c.red('test(s) failed under coverage.')}`,

      'hardhat-viem': "'hardhat-viem' requires an environment variable to be set when used with the solidity-coverage plugin"
    }


    return this._format(kinds[kind])
  }


}

module.exports = PluginUI;
