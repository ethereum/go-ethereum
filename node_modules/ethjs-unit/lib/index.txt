const BigNumber = require('bignumber.js');

// complete ethereum unit map
const unitMap = {
  'noether':      '0', // eslint-disable-line
  'wei':          '1', // eslint-disable-line
  'kwei':         '1000', // eslint-disable-line
  'Kwei':         '1000', // eslint-disable-line
  'babbage':      '1000', // eslint-disable-line
  'femtoether':   '1000', // eslint-disable-line
  'mwei':         '1000000', // eslint-disable-line
  'Mwei':         '1000000', // eslint-disable-line
  'lovelace':     '1000000', // eslint-disable-line
  'picoether':    '1000000', // eslint-disable-line
  'gwei':         '1000000000', // eslint-disable-line
  'Gwei':         '1000000000', // eslint-disable-line
  'shannon':      '1000000000', // eslint-disable-line
  'nanoether':    '1000000000', // eslint-disable-line
  'nano':         '1000000000', // eslint-disable-line
  'szabo':        '1000000000000', // eslint-disable-line
  'microether':   '1000000000000', // eslint-disable-line
  'micro':        '1000000000000', // eslint-disable-line
  'finney':       '1000000000000000', // eslint-disable-line
  'milliether':   '1000000000000000', // eslint-disable-line
  'milli':        '1000000000000000', // eslint-disable-line
  'ether':        '1000000000000000000', // eslint-disable-line
  'kether':       '1000000000000000000000', // eslint-disable-line
  'grand':        '1000000000000000000000', // eslint-disable-line
  'mether':       '1000000000000000000000000', // eslint-disable-line
  'gether':       '1000000000000000000000000000', // eslint-disable-line
  'tether':       '1000000000000000000000000000000', // eslint-disable-line
};

/**
 * Returns value of unit in Wei
 *
 * @method getValueOfUnit
 * @param {String} unit the unit to convert to, default ether
 * @returns {BigNumber} value of the unit (in Wei)
 * @throws error if the unit is not correct:w
 */
function getValueOfUnit(unitInput) {
  const unit = unitInput ? unitInput.toLowerCase() : 'ether';
  var unitValue = unitMap[unit]; // eslint-disable-line

  if (typeof unitValue !== 'string') {
    throw new Error(`This unit doesn't exists, please use the one of the following units ${JSON.stringify(unitMap, null, 2)}`);
  }

  return new BigNumber(unitValue, 10);
}

/**
 * Takes a number of wei and converts it to any other ether unit.
 *
 * Possible units are:
 *   SI Short   SI Full        Effigy       Other
 * - kwei       femtoether     babbage
 * - mwei       picoether      lovelace
 * - gwei       nanoether      shannon      nano
 * - --         microether     szabo        micro
 * - --         milliether     finney       milli
 * - ether      --             --
 * - kether                    --           grand
 * - mether
 * - gether
 * - tether
 *
 * @method fromWei
 * @param {Number|String} number can be a number, number string or a HEX of a decimal
 * @param {String} unit the unit to convert to, default ether
 * @return {Object} When given a BigNumber object it returns one as well, otherwise a number
*/
function fromWei(number, unit) {
  const returnValue = toBigNumber(number).dividedBy(getValueOfUnit(unit));

  return returnValue;
}

/**
 * Takes a number of a unit and converts it to wei.
 *
 * Possible units are:
 *   SI Short   SI Full        Effigy       Other
 * - kwei       femtoether     babbage
 * - mwei       picoether      lovelace
 * - gwei       nanoether      shannon      nano
 * - --         microether     szabo        micro
 * - --         microether     szabo        micro
 * - --         milliether     finney       milli
 * - ether      --             --
 * - kether                    --           grand
 * - mether
 * - gether
 * - tether
 *
 * @method toWei
 * @param {Number|String|BigNumber} number can be a number, number string or a HEX of a decimal
 * @param {String} unit the unit to convert from, default ether
 * @return {Object} When given a BigNumber object it returns one as well, otherwise a number
*/
function toWei(number, unit) {
  const returnValue = toBigNumber(number).times(getValueOfUnit(unit));

  return returnValue;
}

/**
 * Returns true if object is BigNumber, otherwise false
 *
 * @method isBigNumber
 * @param {Object}
 * @return {Boolean}
 */
function isBigNumber(object) {
  return object instanceof BigNumber || (object && object.constructor && object.constructor.name === 'BigNumber');
}

/**
 * Takes an input and transforms it into an bignumber
 *
 * @method toBigNumber
 * @param {Number|String|BigNumber} a number, string, HEX string or BigNumber
 * @return {BigNumber} BigNumber
*/
function toBigNumber(numberInput) {
  const number = numberInput || 0;

  if (isBigNumber(number)) {
    return number;
  }

  if (typeof number === 'string' && (number.indexOf('0x') === 0 || number.indexOf('-0x') === 0)) {
    return new BigNumber(number.replace('0x', ''), 16);
  }

  return new BigNumber(number.toString(10), 10);
}

module.exports = {
  unitMap,
  isBigNumber,
  toBigNumber,
  toWei,
  fromWei,
};
