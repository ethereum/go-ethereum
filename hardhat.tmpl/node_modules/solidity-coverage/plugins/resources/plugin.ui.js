const UI = require('./../../lib/ui').UI;

/**
 * Plugin logging
 */
class PluginUI extends UI {
  constructor(log){
    super(log);
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

      'instr-skipped': `${ds} ${c.grey(args[0])}`,

      'network': `\n${c.bold('Network Info')}` +
                 `\n${c.bold('============')}\n` +
                 `${ct} ${c.bold('id')}:      ${args[1]}\n` +
                 `${ct} ${c.bold('port')}:    ${args[2]}\n` +
                 `${ct} ${c.bold('network')}: ${args[0]}\n`,

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

      'sources-fail': `${c.red('Cannot locate expected contract sources folder: ')} ${args[0]}`,

      'solcoverjs-fail': `${c.red('Could not load .solcover.js config file. ')}` +
                         `${c.red('This can happen if it has a syntax error or ')}` +
                         `${c.red('the path you specified for it is wrong.')}`,

      'tests-fail': `${x} ${c.bold(args[0])} ${c.red('test(s) failed under coverage.')}`,

      'mocha-parallel-fail': `${c.red('Coverage cannot be run in mocha parallel mode. ')}` +
                             `${c.red('Set \`mocha: { parallel: false }\` in .solcover.js ')}` +
                             `${c.red('to disable the option for the coverage task. ')}` +
                             `${c.red('See the solidity-coverage README FAQ for info on parallelizing coverage in CI.')}`,
    }


    return this._format(kinds[kind])
  }
}

module.exports = PluginUI;