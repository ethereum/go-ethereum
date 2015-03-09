require=(function e(t,n,r){function s(o,u){if(!n[o]){if(!t[o]){var a=typeof require=="function"&&require;if(!u&&a)return a(o,!0);if(i)return i(o,!0);var f=new Error("Cannot find module '"+o+"'");throw f.code="MODULE_NOT_FOUND",f}var l=n[o]={exports:{}};t[o][0].call(l.exports,function(e){var n=t[o][1][e];return s(n?n:e)},l,l.exports,e,t,n,r)}return n[o].exports}var i=typeof require=="function"&&require;for(var o=0;o<r.length;o++)s(r[o]);return s})({1:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file abi.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var utils = require('./utils');
var types = require('./types');
var c = require('./const');
var f = require('./formatters');

var displayTypeError = function (type) {
    console.error('parser does not support type: ' + type);
};

/// This method should be called if we want to check if givent type is an array type
/// @returns true if it is, otherwise false
var arrayType = function (type) {
    return type.slice(-2) === '[]';
};

var dynamicTypeBytes = function (type, value) {
    // TODO: decide what to do with array of strings
    if (arrayType(type) || type === 'string')    // only string itself that is dynamic; stringX is static length.
        return f.formatInputInt(value.length);
    return "";
};

var inputTypes = types.inputTypes();

/// Formats input params to bytes
/// @param abi contract method inputs
/// @param array of params that will be formatted to bytes
/// @returns bytes representation of input params
var formatInput = function (inputs, params) {
    var bytes = "";
    var toAppendConstant = "";
    var toAppendArrayContent = "";

    /// first we iterate in search for dynamic
    inputs.forEach(function (input, index) {
        bytes += dynamicTypeBytes(input.type, params[index]);
    });

    inputs.forEach(function (input, i) {
        /*jshint maxcomplexity:5 */
        var typeMatch = false;
        for (var j = 0; j < inputTypes.length && !typeMatch; j++) {
            typeMatch = inputTypes[j].type(inputs[i].type, params[i]);
        }
        if (!typeMatch) {
            displayTypeError(inputs[i].type);
        }

        var formatter = inputTypes[j - 1].format;

        if (arrayType(inputs[i].type))
            toAppendArrayContent += params[i].reduce(function (acc, curr) {
                return acc + formatter(curr);
            }, "");
        else if (inputs[i].type === 'string')
            toAppendArrayContent += formatter(params[i]);
        else
            toAppendConstant += formatter(params[i]);
    });

    bytes += toAppendConstant + toAppendArrayContent;

    return bytes;
};

var dynamicBytesLength = function (type) {
    if (arrayType(type) || type === 'string')   // only string itself that is dynamic; stringX is static length.
        return c.ETH_PADDING * 2;
    return 0;
};

var outputTypes = types.outputTypes();

/// Formats output bytes back to param list
/// @param contract abi method outputs
/// @param bytes representtion of output
/// @returns array of output params
var formatOutput = function (outs, output) {

    output = output.slice(2);
    var result = [];
    var padding = c.ETH_PADDING * 2;

    var dynamicPartLength = outs.reduce(function (acc, curr) {
        return acc + dynamicBytesLength(curr.type);
    }, 0);

    var dynamicPart = output.slice(0, dynamicPartLength);
    output = output.slice(dynamicPartLength);

    outs.forEach(function (out, i) {
        /*jshint maxcomplexity:6 */
        var typeMatch = false;
        for (var j = 0; j < outputTypes.length && !typeMatch; j++) {
            typeMatch = outputTypes[j].type(outs[i].type);
        }

        if (!typeMatch) {
            displayTypeError(outs[i].type);
        }

        var formatter = outputTypes[j - 1].format;
        if (arrayType(outs[i].type)) {
            var size = f.formatOutputUInt(dynamicPart.slice(0, padding));
            dynamicPart = dynamicPart.slice(padding);
            var array = [];
            for (var k = 0; k < size; k++) {
                array.push(formatter(output.slice(0, padding)));
                output = output.slice(padding);
            }
            result.push(array);
        }
        else if (types.prefixedType('string')(outs[i].type)) {
            dynamicPart = dynamicPart.slice(padding);
            result.push(formatter(output.slice(0, padding)));
            output = output.slice(padding);
        } else {
            result.push(formatter(output.slice(0, padding)));
            output = output.slice(padding);
        }
    });

    return result;
};

/// @param json abi for contract
/// @returns input parser object for given json abi
/// TODO: refactor creating the parser, do not double logic from contract
var inputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {
        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            return formatInput(method.inputs, params);
        };

        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

/// @param json abi for contract
/// @returns output parser for given json abi
var outputParser = function (json) {
    var parser = {};
    json.forEach(function (method) {

        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function (output) {
            return formatOutput(method.outputs, output);
        };

        if (parser[displayName] === undefined) {
            parser[displayName] = impl;
        }

        parser[displayName][typeName] = impl;
    });

    return parser;
};

module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    formatInput: formatInput,
    formatOutput: formatOutput
};

},{"./const":2,"./formatters":8,"./types":15,"./utils":16}],2:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file const.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// required to define ETH_BIGNUMBER_ROUNDING_MODE
if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var ETH_UNITS = [ 
    'wei', 
    'Kwei', 
    'Mwei', 
    'Gwei', 
    'szabo', 
    'finney', 
    'ether', 
    'grand', 
    'Mether', 
    'Gether', 
    'Tether', 
    'Pether', 
    'Eether', 
    'Zether', 
    'Yether', 
    'Nether', 
    'Dether', 
    'Vether', 
    'Uether' 
];

module.exports = {
    ETH_PADDING: 32,
    ETH_SIGNATURE_LENGTH: 4,
    ETH_UNITS: ETH_UNITS,
    ETH_BIGNUMBER_ROUNDING_MODE: { ROUNDING_MODE: BigNumber.ROUND_DOWN },
    ETH_POLLING_TIMEOUT: 1000
};


},{}],3:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file contract.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

var web3 = require('./web3'); 
var abi = require('./abi');
var utils = require('./utils');
var eventImpl = require('./event');
var signature = require('./signature');

var exportNatspecGlobals = function (vars) {
    // it's used byt natspec.js
    // TODO: figure out better way to solve this
    web3._currentContractAbi = vars.abi;
    web3._currentContractAddress = vars.address;
    web3._currentContractMethodName = vars.method;
    web3._currentContractMethodParams = vars.params;
};

var addFunctionRelatedPropertiesToContract = function (contract) {
    
    contract.call = function (options) {
        contract._isTransaction = false;
        contract._options = options;
        return contract;
    };


    contract.sendTransaction = function (options) {
        contract._isTransaction = true;
        contract._options = options;
        return contract;
    };
    // DEPRECATED
    contract.transact = function (options) {

        console.warn('myContract.transact() is deprecated please use myContract.sendTransaction() instead.');

        return contract.sendTransaction(options);
    };

    contract._options = {};
    ['gas', 'gasPrice', 'value', 'from'].forEach(function(p) {
        contract[p] = function (v) {
            contract._options[p] = v;
            return contract;
        };
    });

};

var addFunctionsToContract = function (contract, desc, address) {
    var inputParser = abi.inputParser(desc);
    var outputParser = abi.outputParser(desc);

    // create contract functions
    utils.filterFunctions(desc).forEach(function (method) {

        var displayName = utils.extractDisplayName(method.name);
        var typeName = utils.extractTypeName(method.name);

        var impl = function () {
            /*jshint maxcomplexity:7 */
            var params = Array.prototype.slice.call(arguments);
            var sign = signature.functionSignatureFromAscii(method.name);
            var parsed = inputParser[displayName][typeName].apply(null, params);

            var options = contract._options || {};
            options.to = address;
            options.data = sign + parsed;
            
            var isTransaction = contract._isTransaction === true || (contract._isTransaction !== false && !method.constant);
            var collapse = options.collapse !== false;
            
            // reset
            contract._options = {};
            contract._isTransaction = null;

            if (isTransaction) {
                
                exportNatspecGlobals({
                    abi: desc,
                    address: address,
                    method: method.name,
                    params: params
                });

                // transactions do not have any output, cause we do not know, when they will be processed
                web3.eth.sendTransaction(options);
                return;
            }
            
            var output = web3.eth.call(options);
            var ret = outputParser[displayName][typeName](output);
            if (collapse)
            {
                if (ret.length === 1)
                    ret = ret[0];
                else if (ret.length === 0)
                    ret = null;
            }
            return ret;
        };

        if (contract[displayName] === undefined) {
            contract[displayName] = impl;
        }

        contract[displayName][typeName] = impl;
    });
};

var addEventRelatedPropertiesToContract = function (contract, desc, address) {
    contract.address = address;
    contract._onWatchEventResult = function (data) {
        var matchingEvent = event.getMatchingEvent(utils.filterEvents(desc));
        var parser = eventImpl.outputParser(matchingEvent);
        return parser(data);
    };
    
    Object.defineProperty(contract, 'topic', {
        get: function() {
            return utils.filterEvents(desc).map(function (e) {
                return signature.eventSignatureFromAscii(e.name);
            });
        }
    });

};

var addEventsToContract = function (contract, desc, address) {
    // create contract events
    utils.filterEvents(desc).forEach(function (e) {

        var impl = function () {
            var params = Array.prototype.slice.call(arguments);
            var sign = signature.eventSignatureFromAscii(e.name);
            var event = eventImpl.inputParser(address, sign, e);
            var o = event.apply(null, params);
            var outputFormatter = function (data) {
                var parser = eventImpl.outputParser(e);
                return parser(data);
            };
            return web3.eth.filter(o, undefined, undefined, outputFormatter);
        };
        
        // this property should be used by eth.filter to check if object is an event
        impl._isEvent = true;

        var displayName = utils.extractDisplayName(e.name);
        var typeName = utils.extractTypeName(e.name);

        if (contract[displayName] === undefined) {
            contract[displayName] = impl;
        }

        contract[displayName][typeName] = impl;

    });
};


/**
 * This method should be called when we want to call / transact some solidity method from javascript
 * it returns an object which has same methods available as solidity contract description
 * usage example: 
 *
 * var abi = [{
 *      name: 'myMethod',
 *      inputs: [{ name: 'a', type: 'string' }],
 *      outputs: [{name: 'd', type: 'string' }]
 * }];  // contract abi
 *
 * var MyContract = web3.eth.contract(abi); // creation of contract prototype
 *
 * var contractInstance = new MyContract('0x0123123121');
 *
 * contractInstance.myMethod('this is test string param for call'); // myMethod call (implicit, default)
 * contractInstance.call().myMethod('this is test string param for call'); // myMethod call (explicit)
 * contractInstance.sendTransaction().myMethod('this is test string param for transact'); // myMethod sendTransaction
 *
 * @param abi - abi json description of the contract, which is being created
 * @returns contract object
 */
var contract = function (abi) {

    // return prototype
    if(abi instanceof Array && arguments.length === 1) {
        return Contract.bind(null, abi);

    // deprecated: auto initiate contract
    } else {

        console.warn('Initiating a contract like this is deprecated please use var MyContract = eth.contract(abi); new MyContract(address); instead.');

        return new Contract(arguments[1], arguments[0]);
    }

};

function Contract(abi, address) {

    // workaround for invalid assumption that method.name is the full anonymous prototype of the method.
    // it's not. it's just the name. the rest of the code assumes it's actually the anonymous
    // prototype, so we make it so as a workaround.
    // TODO: we may not want to modify input params, maybe use copy instead?
    abi.forEach(function (method) {
        if (method.name.indexOf('(') === -1) {
            var displayName = method.name;
            var typeName = method.inputs.map(function(i){return i.type; }).join();
            method.name = displayName + '(' + typeName + ')';
        }
    });

    var result = {};
    addFunctionRelatedPropertiesToContract(result);
    addFunctionsToContract(result, abi, address);
    addEventRelatedPropertiesToContract(result, abi, address);
    addEventsToContract(result, abi, address);

    return result;
}

module.exports = contract;


},{"./abi":1,"./event":6,"./signature":14,"./utils":16,"./web3":18}],4:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file db.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// @returns an array of objects describing web3.db api methods
var methods = function () {
    return [
    { name: 'put', call: 'db_put' },
    { name: 'get', call: 'db_get' },
    { name: 'putString', call: 'db_putString' },
    { name: 'getString', call: 'db_getString' }
    ];
};

module.exports = {
    methods: methods
};

},{}],5:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file eth.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var formatters = require('./formatters');


var blockCall = function (args) {
    return typeof args[0] === "string" ? "eth_blockByHash" : "eth_blockByNumber";
};

var transactionCall = function (args) {
    return typeof args[0] === "string" ? 'eth_transactionByHash' : 'eth_transactionByNumber';
};

var uncleCall = function (args) {
    return typeof args[0] === "string" ? 'eth_uncleByHash' : 'eth_uncleByNumber';
};

var transactionCountCall = function (args) {
    return typeof args[0] === "string" ? 'eth_transactionCountByHash' : 'eth_transactionCountByNumber';
};

var uncleCountCall = function (args) {
    return typeof args[0] === "string" ? 'eth_uncleCountByHash' : 'eth_uncleCountByNumber';
};

/// @returns an array of objects describing web3.eth api methods
var methods = [
    { name: 'getBalance', call: 'eth_balanceAt', outputFormatter: formatters.convertToBigNumber},
    { name: 'getState', call: 'eth_stateAt' },
    { name: 'getStorage', call: 'eth_storageAt' },
    { name: 'getData', call: 'eth_codeAt' },
    { name: 'getBlock', call: blockCall, outputFormatter: formatters.outputBlockFormatter},
    { name: 'getUncle', call: uncleCall, outputFormatter: formatters.outputBlockFormatter},
    { name: 'getCompilers', call: 'eth_compilers' },
    { name: 'getBlockTransactionCount', call: transactionCountCall },
    { name: 'getBlockUncleCount', call: uncleCountCall },
    { name: 'getTransaction', call: transactionCall, outputFormatter: formatters.outputTransactionFormatter },
    { name: 'getTransactionCount', call: 'eth_countAt'},
    { name: 'sendTransaction', call: 'eth_transact', inputFormatter: formatters.inputTransactionFormatter },
    { name: 'call', call: 'eth_call' },
    { name: 'compile.solidity', call: 'eth_solidity' },
    { name: 'compile.lll', call: 'eth_lll' },
    { name: 'compile.serpent', call: 'eth_serpent' },
    { name: 'flush', call: 'eth_flush' },

    // deprecated methods
    { name: 'balanceAt', call: 'eth_balanceAt', newMethod: 'getBalance' },
    { name: 'stateAt', call: 'eth_stateAt', newMethod: 'getState' },
    { name: 'storageAt', call: 'eth_storageAt', newMethod: 'getStorage' },
    { name: 'countAt', call: 'eth_countAt', newMethod: 'getTransactionCount' },
    { name: 'codeAt', call: 'eth_codeAt', newMethod: 'getData' },
    { name: 'transact', call: 'eth_transact', newMethod: 'sendTransaction' },
    { name: 'block', call: blockCall, newMethod: 'getBlock' },
    { name: 'transaction', call: transactionCall, newMethod: 'getTransaction' },
    { name: 'uncle', call: uncleCall, newMethod: 'getUncle' },
    { name: 'compilers', call: 'eth_compilers', newMethod: 'getCompilers' },
    { name: 'solidity', call: 'eth_solidity', newMethod: 'compile.solidity' },
    { name: 'lll', call: 'eth_lll', newMethod: 'compile.lll' },
    { name: 'serpent', call: 'eth_serpent', newMethod: 'compile.serpent' },
    { name: 'transactionCount', call: transactionCountCall, newMethod: 'getBlockTransactionCount' },
    { name: 'uncleCount', call: uncleCountCall, newMethod: 'getBlockUncleCount' },
    { name: 'logs', call: 'eth_logs' }
];

/// @returns an array of objects describing web3.eth api properties
var properties = [
    { name: 'coinbase', getter: 'eth_coinbase', setter: 'eth_setCoinbase' },
    { name: 'listening', getter: 'eth_listening', setter: 'eth_setListening' },
    { name: 'mining', getter: 'eth_mining', setter: 'eth_setMining' },
    { name: 'gasPrice', getter: 'eth_gasPrice', outputFormatter: formatters.convertToBigNumber},
    { name: 'accounts', getter: 'eth_accounts' },
    { name: 'peerCount', getter: 'eth_peerCount' },
    { name: 'defaultBlock', getter: 'eth_defaultBlock', setter: 'eth_setDefaultBlock' },
    { name: 'blockNumber', getter: 'eth_number'},

    // deprecated properties
    { name: 'number', getter: 'eth_number', newProperty: 'blockNumber'}
];


module.exports = {
    methods: methods,
    properties: properties
};


},{"./formatters":8}],6:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file event.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2014
 */

var abi = require('./abi');
var utils = require('./utils');
var signature = require('./signature');

/// filter inputs array && returns only indexed (or not) inputs
/// @param inputs array
/// @param bool if result should be an array of indexed params on not
/// @returns array of (not?) indexed params
var filterInputs = function (inputs, indexed) {
    return inputs.filter(function (current) {
        return current.indexed === indexed;
    });
};

var inputWithName = function (inputs, name) {
    var index = utils.findIndex(inputs, function (input) {
        return input.name === name;
    });
    
    if (index === -1) {
        console.error('indexed param with name ' + name + ' not found');
        return undefined;
    }
    return inputs[index];
};

var indexedParamsToTopics = function (event, indexed) {
    // sort keys?
    return Object.keys(indexed).map(function (key) {
        var inputs = [inputWithName(filterInputs(event.inputs, true), key)];

        var value = indexed[key];
        if (value instanceof Array) {
            return value.map(function (v) {
                return abi.formatInput(inputs, [v]);
            }); 
        }
        return abi.formatInput(inputs, [value]);
    });
};

var inputParser = function (address, sign, event) {
    
    // valid options are 'earliest', 'latest', 'offset' and 'max', as defined for 'eth.filter'
    return function (indexed, options) {
        var o = options || {};
        o.address = address;
        o.topic = [];
        o.topic.push(sign);
        if (indexed) {
            o.topic = o.topic.concat(indexedParamsToTopics(event, indexed));
        }
        return o;
    };
};

var getArgumentsObject = function (inputs, indexed, notIndexed) {
    var indexedCopy = indexed.slice();
    var notIndexedCopy = notIndexed.slice();
    return inputs.reduce(function (acc, current) {
        var value;
        if (current.indexed)
            value = indexedCopy.splice(0, 1)[0];
        else
            value = notIndexedCopy.splice(0, 1)[0];

        acc[current.name] = value;
        return acc;
    }, {}); 
};
 
var outputParser = function (event) {
    
    return function (output) {
        var result = {
            event: utils.extractDisplayName(event.name),
            number: output.number,
            hash: output.hash,
            args: {}
        };

        output.topics = output.topic; // fallback for go-ethereum
        if (!output.topic) {
            return result;
        }
       
        var indexedOutputs = filterInputs(event.inputs, true);
        var indexedData = "0x" + output.topic.slice(1, output.topic.length).map(function (topic) { return topic.slice(2); }).join("");
        var indexedRes = abi.formatOutput(indexedOutputs, indexedData);

        var notIndexedOutputs = filterInputs(event.inputs, false);
        var notIndexedRes = abi.formatOutput(notIndexedOutputs, output.data);

        result.args = getArgumentsObject(event.inputs, indexedRes, notIndexedRes);

        return result;
    };
};

var getMatchingEvent = function (events, payload) {
    for (var i = 0; i < events.length; i++) {
        var sign = signature.eventSignatureFromAscii(events[i].name); 
        if (sign === payload.topic[0]) {
            return events[i];
        }
    }
    return undefined;
};


module.exports = {
    inputParser: inputParser,
    outputParser: outputParser,
    getMatchingEvent: getMatchingEvent
};


},{"./abi":1,"./signature":14,"./utils":16}],7:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file filter.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

/// Should be called to check if filter implementation is valid
/// @returns true if it is, otherwise false
var implementationIsValid = function (i) {
    return !!i && 
        typeof i.newFilter === 'function' && 
        typeof i.getLogs === 'function' && 
        typeof i.uninstallFilter === 'function' &&
        typeof i.startPolling === 'function' &&
        typeof i.stopPolling === 'function';
};

/// This method should be called on options object, to verify deprecated properties && lazy load dynamic ones
/// @param should be string or object
/// @returns options string or object
var getOptions = function (options) {
    if (typeof options === 'string') {
        return options;
    } 

    options = options || {};

    if (options.topics) {
        console.warn('"topics" is deprecated, is "topic" instead');
    }

    // evaluate lazy properties
    return {
        to: options.to,
        topic: options.topic,
        earliest: options.earliest,
        latest: options.latest,
        max: options.max,
        skip: options.skip,
        address: options.address
    };
};

/// Should be used when we want to watch something
/// it's using inner polling mechanism and is notified about changes
/// @param options are filter options
/// @param implementation, an abstract polling implementation
/// @param formatter (optional), callback function which formats output before 'real' callback 
var filter = function(options, implementation, formatter) {
    if (!implementationIsValid(implementation)) {
        console.error('filter implemenation is invalid');
        return;
    }

    options = getOptions(options);
    var callbacks = [];
    var filterId = implementation.newFilter(options);
    var onMessages = function (messages) {
        messages.forEach(function (message) {
            message = formatter ? formatter(message) : message;
            callbacks.forEach(function (callback) {
                callback(message);
            });
        });
    };

    implementation.startPolling(filterId, onMessages, implementation.uninstallFilter);

    var watch = function(callback) {
        callbacks.push(callback);
    };

    var stopWatching = function() {
        implementation.stopPolling(filterId);
        implementation.uninstallFilter(filterId);
        callbacks = [];
    };

    var get = function () {
        return implementation.getLogs(filterId);
    };
    
    return {
        watch: watch,
        stopWatching: stopWatching,
        get: get,

        // DEPRECATED methods
        changed:  function(){
            console.warn('watch().changed() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        arrived:  function(){
            console.warn('watch().arrived() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        happened:  function(){
            console.warn('watch().happened() is deprecated please use filter().watch() instead.');
            return watch.apply(this, arguments);
        },
        uninstall: function(){
            console.warn('watch().uninstall() is deprecated please use filter().stopWatching() instead.');
            return stopWatching.apply(this, arguments);
        },
        messages: function(){
            console.warn('watch().messages() is deprecated please use filter().get() instead.');
            return get.apply(this, arguments);
        },
        logs: function(){
            console.warn('watch().logs() is deprecated please use filter().get() instead.');
            return get.apply(this, arguments);
        }
    };
};

module.exports = filter;


},{}],8:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file formatters.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var utils = require('./utils');
var c = require('./const');

/// @param string string to be padded
/// @param number of characters that result string should have
/// @param sign, by default 0
/// @returns right aligned string
var padLeft = function (string, chars, sign) {
    return new Array(chars - string.length + 1).join(sign ? sign : "0") + string;
};

/// Formats input value to byte representation of int
/// If value is negative, return it's two's complement
/// If the value is floating point, round it down
/// @returns right-aligned byte representation of int
var formatInputInt = function (value) {
    /*jshint maxcomplexity:7 */
    var padding = c.ETH_PADDING * 2;
    if (utils.isBigNumber(value) || typeof value === 'number') {
        if (typeof value === 'number')
            value = new BigNumber(value);
        BigNumber.config(c.ETH_BIGNUMBER_ROUNDING_MODE);
        value = value.round();

        if (value.lessThan(0)) 
            value = new BigNumber("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16).plus(value).plus(1);
        value = value.toString(16);
    }
    else if (typeof value === 'string') {
        if (value.indexOf('0x') === 0) {
            value = value.substr(2);
        } else {
            value = formatInputInt(new BigNumber(value));
        }
    }
    else
        value = (+value).toString(16);
    return padLeft(value, padding);
};

/// Formats input value to byte representation of string
/// @returns left-algined byte representation of string
var formatInputString = function (value) {
    return utils.fromAscii(value, c.ETH_PADDING).substr(2);
};

/// Formats input value to byte representation of bool
/// @returns right-aligned byte representation bool
var formatInputBool = function (value) {
    return '000000000000000000000000000000000000000000000000000000000000000' + (value ?  '1' : '0');
};

/// Formats input value to byte representation of real
/// Values are multiplied by 2^m and encoded as integers
/// @returns byte representation of real
var formatInputReal = function (value) {
    return formatInputInt(new BigNumber(value).times(new BigNumber(2).pow(128))); 
};


/// Check if input value is negative
/// @param value is hex format
/// @returns true if it is negative, otherwise false
var signedIsNegative = function (value) {
    return (new BigNumber(value.substr(0, 1), 16).toString(2).substr(0, 1)) === '1';
};

/// Formats input right-aligned input bytes to int
/// @returns right-aligned input bytes formatted to int
var formatOutputInt = function (value) {

    value = value || "0";

    // check if it's negative number
    // it it is, return two's complement
    if (signedIsNegative(value)) {
        return new BigNumber(value, 16).minus(new BigNumber('ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff', 16)).minus(1);
    }
    return new BigNumber(value, 16);
};


/// Formats big right-aligned input bytes to uint
/// @returns right-aligned input bytes formatted to uint
var formatOutputUInt = function (value) {
    value = value || "0";
    return new BigNumber(value, 16);
};

/// @returns input bytes formatted to real
var formatOutputReal = function (value) {
    return formatOutputInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/// @returns input bytes formatted to ureal
var formatOutputUReal = function (value) {
    return formatOutputUInt(value).dividedBy(new BigNumber(2).pow(128)); 
};

/// @returns right-aligned input bytes formatted to hex
var formatOutputHash = function (value) {
    return "0x" + value;
};

/// @returns right-aligned input bytes formatted to bool
var formatOutputBool = function (value) {
    return value === '0000000000000000000000000000000000000000000000000000000000000001' ? true : false;
};

/// @returns left-aligned input bytes formatted to ascii string
var formatOutputString = function (value) {
    return utils.toAscii(value);
};

/// @returns right-aligned input bytes formatted to address
var formatOutputAddress = function (value) {
    return "0x" + value.slice(value.length - 40, value.length);
};


/// Formats the input to a big number
/// @returns a BigNumber object
var convertToBigNumber = function (value) {

    // remove the leading 0x
    if(typeof value === 'string')
        value = value.replace('0x', '');

    value = value || "0";

    return new BigNumber(value, 16);
};


/**
Formats the input of a transaction and converts all values to HEX

@returns object
*/
var inputTransactionFormatter = function(options){

    // make code -> data
    if(options.code) {
        options.data = options.code;
        delete options.code;
    }

    // make endowment -> value
    if(options.endowment) {
        options.value = options.endowment;
        delete options.endowment;
    }


    // format the following options
    /*jshint maxcomplexity:5 */
    ['gasPrice', 'value'].forEach(function(key){

        // if hex or string integer
        if(typeof options[key] === 'string') {

            // if not hex assume its a number string
            if(options[key].indexOf('0x') === -1)
                options[key] = utils.fromDecimal(options[key]);

        // if number
        } else if(typeof options[key] === 'number') {
            options[key] = utils.fromDecimal(options[key]);

        // if bignumber
        } else if(options[key] instanceof BigNumber) {
            options[key] = '0x'+ options[key].toString(16);
        }
    });

    // format gas to number
    options.gas = Number(options.gas);


    return options;
};

/**
Formats the output of a transaction to its proper values

@returns object
*/
var outputTransactionFormatter = function(tx){
    // transform to number
    tx.gas = Number(tx.gas);

    // gasPrice to bignumber
    if(typeof tx.gasPrice === 'string' && tx.gasPrice.indexOf('0x') === 0)
        tx.gasPrice = new BigNumber(tx.gasPrice, 16);
    else
        tx.gasPrice = new BigNumber(tx.gasPrice.toString(10), 10);

    // value to bignumber
    if(typeof tx.value === 'string' && tx.value.indexOf('0x') === 0)
        tx.value = new BigNumber(tx.value, 16);
    else
        tx.value = new BigNumber(tx.value.toString(10), 10);

    return tx;
};


/**
Formats the output of a block to its proper values

@returns object
*/
var outputBlockFormatter = function(block){
    /*jshint maxcomplexity:7 */

    // transform to number
    block.gasLimit = Number(block.gasLimit);
    block.gasUsed = Number(block.gasUsed);
    block.size = Number(block.size);
    block.timestamp = Number(block.timestamp);
    block.number = Number(block.number);

    // minGasPrice to bignumber
    if(block.minGasPrice) {
        if(typeof block.minGasPrice === 'string' && block.minGasPrice.indexOf('0x') === 0)
            block.minGasPrice = new BigNumber(block.minGasPrice, 16);
        else
            block.minGasPrice = new BigNumber(block.minGasPrice.toString(10), 10);
    }


    // difficulty to bignumber
    if(block.difficulty) {
        if(typeof block.difficulty === 'string' && block.difficulty.indexOf('0x') === 0)
            block.difficulty = new BigNumber(block.difficulty, 16);
        else
            block.difficulty = new BigNumber(block.difficulty.toString(10), 10);
    }


    // difficulty to bignumber
    if(block.totalDifficulty) {
        if(typeof block.totalDifficulty === 'string' && block.totalDifficulty.indexOf('0x') === 0)
            block.totalDifficulty = new BigNumber(block.totalDifficulty, 16);
        else
            block.totalDifficulty = new BigNumber(block.totalDifficulty.toString(10), 10);
    }

    return block;
};


module.exports = {
    formatInputInt: formatInputInt,
    formatInputString: formatInputString,
    formatInputBool: formatInputBool,
    formatInputReal: formatInputReal,
    formatOutputInt: formatOutputInt,
    formatOutputUInt: formatOutputUInt,
    formatOutputReal: formatOutputReal,
    formatOutputUReal: formatOutputUReal,
    formatOutputHash: formatOutputHash,
    formatOutputBool: formatOutputBool,
    formatOutputString: formatOutputString,
    formatOutputAddress: formatOutputAddress,
    convertToBigNumber: convertToBigNumber,
    inputTransactionFormatter: inputTransactionFormatter,
    outputTransactionFormatter: outputTransactionFormatter,
    outputBlockFormatter: outputBlockFormatter
};


},{"./const":2,"./utils":16}],9:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file httpsync.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

if ("build" !== 'build') {/*
        var XMLHttpRequest = require('xmlhttprequest').XMLHttpRequest; // jshint ignore:line
*/}

var HttpSyncProvider = function (host) {
    this.handlers = [];
    this.host = host || 'http://localhost:8080';
};

HttpSyncProvider.prototype.send = function (payload) {
    //var data = formatJsonRpcObject(payload);

    var request = new XMLHttpRequest();
    request.open('POST', this.host, false);
    request.send(JSON.stringify(payload));

    var result = request.responseText;
    // check request.status
    if(request.status !== 200)
        return;
    return JSON.parse(result);
};

module.exports = HttpSyncProvider;


},{}],10:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file jsonrpc.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var messageId = 1;

/// Should be called to valid json create payload object
/// @param method of jsonrpc call, required
/// @param params, an array of method params, optional
/// @returns valid jsonrpc payload object
var toPayload = function (method, params) {
    if (!method)
        console.error('jsonrpc method should be specified!');

    return {
        jsonrpc: '2.0',
        method: method,
        params: params || [],
        id: messageId++
    }; 
};

/// Should be called to check if jsonrpc response is valid
/// @returns true if response is valid, otherwise false 
var isValidResponse = function (response) {
    return !!response &&
        !response.error &&
        response.jsonrpc === '2.0' &&
        typeof response.id === 'number' &&
        response.result !== undefined; // only undefined is not valid json object
};

/// Should be called to create batch payload object
/// @param messages, an array of objects with method (required) and params (optional) fields
var toBatchPayload = function (messages) {
    return messages.map(function (message) {
        return toPayload(message.method, message.params);
    }); 
};

module.exports = {
    toPayload: toPayload,
    isValidResponse: isValidResponse,
    toBatchPayload: toBatchPayload
};



},{}],11:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file qtsync.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 * @date 2014
 */

var QtSyncProvider = function () {
};

QtSyncProvider.prototype.send = function (payload) {
    var result = navigator.qt.callMethod(JSON.stringify(payload));
    return JSON.parse(result);
};

module.exports = QtSyncProvider;


},{}],12:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file requestmanager.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

var jsonrpc = require('./jsonrpc');
var c = require('./const');

/**
 * It's responsible for passing messages to providers
 * It's also responsible for polling the ethereum node for incoming messages
 * Default poll timeout is 1 second
 */
var requestManager = function() {
    var polls = [];
    var timeout = null;
    var provider;

    var send = function (data) {
        /*jshint maxcomplexity: 6 */

        // format the input before sending
        if(typeof data.inputFormatter === 'function') {
            data.params = Array.prototype.map.call(data.params, function(item){
                return data.inputFormatter(item);
            });
        }

        var payload = jsonrpc.toPayload(data.method, data.params);
        
        if (!provider) {
            console.error('provider is not set');
            return null;
        }

        var result = provider.send(payload);

        if (!jsonrpc.isValidResponse(result)) {
            console.log(result);
            if(typeof result === 'object' && result.error && result.error.message)
                console.error(result.error.message);
            return null;
        }
        
        // format the output
        return (typeof data.outputFormatter === 'function') ? data.outputFormatter(result.result) : result.result;
    };

    var setProvider = function (p) {
        provider = p;
    };

    /*jshint maxparams:4 */
    var startPolling = function (data, pollId, callback, uninstall) {
        polls.push({data: data, id: pollId, callback: callback, uninstall: uninstall});
    };
    /*jshint maxparams:3 */

    var stopPolling = function (pollId) {
        for (var i = polls.length; i--;) {
            var poll = polls[i];
            if (poll.id === pollId) {
                polls.splice(i, 1);
            }
        }
    };

    var reset = function () {
        polls.forEach(function (poll) {
            poll.uninstall(poll.id); 
        });
        polls = [];

        if (timeout) {
            clearTimeout(timeout);
            timeout = null;
        }
        poll();
    };

    var poll = function () {
        polls.forEach(function (data) {
            var result = send(data.data);
            if (!(result instanceof Array) || result.length === 0) {
                return;
            }
            data.callback(result);
        });
        timeout = setTimeout(poll, c.ETH_POLLING_TIMEOUT);
    };
    
    poll();

    return {
        send: send,
        setProvider: setProvider,
        startPolling: startPolling,
        stopPolling: stopPolling,
        reset: reset
    };
};

module.exports = requestManager;


},{"./const":2,"./jsonrpc":10}],13:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file shh.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// @returns an array of objects describing web3.shh api methods
var methods = function () {
    return [
    { name: 'post', call: 'shh_post' },
    { name: 'newIdentity', call: 'shh_newIdentity' },
    { name: 'hasIdentity', call: 'shh_haveIdentity' },
    { name: 'newGroup', call: 'shh_newGroup' },
    { name: 'addToGroup', call: 'shh_addToGroup' },

    // deprecated
    { name: 'haveIdentity', call: 'shh_haveIdentity', newMethod: 'hasIdentity' },
    ];
};

module.exports = {
    methods: methods
};


},{}],14:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file signature.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var web3 = require('./web3'); 
var c = require('./const');

/// @param function name for which we want to get signature
/// @returns signature of function with given name
var functionSignatureFromAscii = function (name) {
    return web3.sha3(web3.fromAscii(name)).slice(0, 2 + c.ETH_SIGNATURE_LENGTH * 2);
};

/// @param event name for which we want to get signature
/// @returns signature of event with given name
var eventSignatureFromAscii = function (name) {
    return web3.sha3(web3.fromAscii(name));
};

module.exports = {
    functionSignatureFromAscii: functionSignatureFromAscii,
    eventSignatureFromAscii: eventSignatureFromAscii
};


},{"./const":2,"./web3":18}],15:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file types.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var f = require('./formatters');

/// @param expected type prefix (string)
/// @returns function which checks if type has matching prefix. if yes, returns true, otherwise false
var prefixedType = function (prefix) {
    return function (type) {
        return type.indexOf(prefix) === 0;
    };
};

/// @param expected type name (string)
/// @returns function which checks if type is matching expected one. if yes, returns true, otherwise false
var namedType = function (name) {
    return function (type) {
        return name === type;
    };
};

/// Setups input formatters for solidity types
/// @returns an array of input formatters 
var inputTypes = function () {
    
    return [
        { type: prefixedType('uint'), format: f.formatInputInt },
        { type: prefixedType('int'), format: f.formatInputInt },
        { type: prefixedType('hash'), format: f.formatInputInt },
        { type: prefixedType('string'), format: f.formatInputString }, 
        { type: prefixedType('real'), format: f.formatInputReal },
        { type: prefixedType('ureal'), format: f.formatInputReal },
        { type: namedType('address'), format: f.formatInputInt },
        { type: namedType('bool'), format: f.formatInputBool }
    ];
};

/// Setups output formaters for solidity types
/// @returns an array of output formatters
var outputTypes = function () {

    return [
        { type: prefixedType('uint'), format: f.formatOutputUInt },
        { type: prefixedType('int'), format: f.formatOutputInt },
        { type: prefixedType('hash'), format: f.formatOutputHash },
        { type: prefixedType('string'), format: f.formatOutputString },
        { type: prefixedType('real'), format: f.formatOutputReal },
        { type: prefixedType('ureal'), format: f.formatOutputUReal },
        { type: namedType('address'), format: f.formatOutputAddress },
        { type: namedType('bool'), format: f.formatOutputBool }
    ];
};

module.exports = {
    prefixedType: prefixedType,
    namedType: namedType,
    inputTypes: inputTypes,
    outputTypes: outputTypes
};


},{"./formatters":8}],16:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file utils.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

var c = require('./const');

if ("build" !== 'build') {/*
    var BigNumber = require('bignumber.js'); // jshint ignore:line
*/}

var unitMap = {
    'wei':      '1',
    'kwei':     '1000',
    'ada':      '1000',
    'mwei':     '1000000',
    'babbage':  '1000000',
    'gwei':     '1000000000',
    'shannon':  '1000000000',
    'szabo':    '1000000000000',
    'finney':   '1000000000000000',
    'ether':    '1000000000000000000',
    'kether':   '1000000000000000000000',
    'grand':    '1000000000000000000000',
    'einstein': '1000000000000000000000',
    'mether':   '1000000000000000000000000',
    'gether':   '1000000000000000000000000000',
    'tether':   '1000000000000000000000000000000'
};


/// Finds first index of array element matching pattern
/// @param array
/// @param callback pattern
/// @returns index of element
var findIndex = function (array, callback) {
    var end = false;
    var i = 0;
    for (; i < array.length && !end; i++) {
        end = callback(array[i]);
    }
    return end ? i - 1 : -1;
};

/// @returns ascii string representation of hex value prefixed with 0x
var toAscii = function(hex) {
// Find termination
    var str = "";
    var i = 0, l = hex.length;
    if (hex.substring(0, 2) === '0x') {
        i = 2;
    }
    for (; i < l; i+=2) {
        var code = parseInt(hex.substr(i, 2), 16);
        if (code === 0) {
            break;
        }

        str += String.fromCharCode(code);
    }

    return str;
};
    
var toHex = function(str) {
    var hex = "";
    for(var i = 0; i < str.length; i++) {
        var n = str.charCodeAt(i).toString(16);
        hex += n.length < 2 ? '0' + n : n;
    }

    return hex;
};

/// @returns hex representation (prefixed by 0x) of ascii string
var fromAscii = function(str, pad) {
    pad = pad === undefined ? 0 : pad;
    var hex = toHex(str);
    while (hex.length < pad*2)
        hex += "00";
    return "0x" + hex;
};

/// @returns display name for function/event eg. multiply(uint256) -> multiply
var extractDisplayName = function (name) {
    var length = name.indexOf('('); 
    return length !== -1 ? name.substr(0, length) : name;
};

/// @returns overloaded part of function/event name
var extractTypeName = function (name) {
    /// TODO: make it invulnerable
    var length = name.indexOf('(');
    return length !== -1 ? name.substr(length + 1, name.length - 1 - (length + 1)).replace(' ', '') : "";
};

/// Filters all function from input abi
/// @returns abi array with filtered objects of type 'function'
var filterFunctions = function (json) {
    return json.filter(function (current) {
        return current.type === 'function'; 
    }); 
};

/// Filters all events form input abi
/// @returns abi array with filtered objects of type 'event'
var filterEvents = function (json) {
    return json.filter(function (current) {
        return current.type === 'event';
    });
};

/// used to transform value/string to eth string
/// TODO: use BigNumber.js to parse int
/// TODO: add tests for it!
var toEth = function (str) {

    console.warn('This method is deprecated please use eth.fromWei(BigNumberOrNumber, unit) instead.');

     /*jshint maxcomplexity:7 */
    var val = typeof str === "string" ? str.indexOf('0x') === 0 ? parseInt(str.substr(2), 16) : parseInt(str.replace(/,/g,'').replace(/ /g,'')) : str;
    var unit = 0;
    var units = c.ETH_UNITS;
    while (val > 3000 && unit < units.length - 1)
    {
        val /= 1000;
        unit++;
    }
    var s = val.toString().length < val.toFixed(2).length ? val.toString() : val.toFixed(2);
    var replaceFunction = function($0, $1, $2) {
        return $1 + ',' + $2;
    };

    while (true) {
        var o = s;
        s = s.replace(/(\d)(\d\d\d[\.\,])/, replaceFunction);
        if (o === s)
            break;
    }
    return s + ' ' + units[unit];
};


var toDecimal = function (val) {
    // remove 0x and place 0, if it's required
    val = val.length > 2 ? val.substring(2) : "0";
    return (new BigNumber(val, 16).toString(10));
};

var fromDecimal = function (val) {
    return "0x" + (new BigNumber(val).toString(16));
};


/**
Takes a number of wei and converts it to any other ether unit.

Possible units are:

    - kwei/ada
    - mwei/babbage
    - gwei/shannon
    - szabo
    - finney
    - ether
    - kether/grand/einstein
    - mether
    - gether
    - tether

@method fromWei
@param {Number|String} number can be a number, number string or a HEX of a decimal
@param {String} unit the unit to convert to
@return {String|Object} When given a BigNumber object it returns one as well, otherwise a number
*/
var fromWei = function(number, unit) {
    /*jshint maxcomplexity: 6 */
    unit = unit.toLowerCase();

    var isBigNumber = true;

    if(!unitMap[unit]) {
        console.warn('This unit doesn\'t exists, please use the one of the following units' , unitMap);
        return number;
    }

    if(!number)
        return number;

    if(typeof number === 'string' && number.indexOf('0x') === 0) {
        isBigNumber = false;
        number = new BigNumber(number, 16);
    }
    
    if(!(number instanceof BigNumber)) {
        isBigNumber = false;
        number = new BigNumber(number.toString(10), 10); // toString to prevent errors, the user have to handle giving correct bignums themselves
    }

    number = number.dividedBy(new BigNumber(unitMap[unit], 10));

    return (isBigNumber) ? number : number.toString(10);
};

/**
Takes a number of a unit and converts it to wei.

Possible units are:

    - kwei/ada
    - mwei/babbage
    - gwei/shannon
    - szabo
    - finney
    - ether
    - kether/grand/einstein
    - mether
    - gether
    - tether

@method toWei
@param {Number|String|BigNumber} number can be a number, number string or a HEX of a decimal
@param {String} unit the unit to convert to
@return {String|Object} When given a BigNumber object it returns one as well, otherwise a number
*/
var toWei = function(number, unit) {
    /*jshint maxcomplexity: 6 */
    unit = unit.toLowerCase();

    var isBigNumber = true;

    if(!unitMap[unit]) {
        console.warn('This unit doesn\'t exists, please use the one of the following units' , unitMap);
        return number;
    }

    if(!number)
        return number;

    if(typeof number === 'string' && number.indexOf('0x') === 0) {
        isBigNumber = false;
        number = new BigNumber(number, 16);
    }

    if(!(number instanceof BigNumber)) {
        isBigNumber = false;
        number = new BigNumber(number.toString(10), 10);// toString to prevent errors, the user have to handle giving correct bignums themselves
    }


    number = number.times(new BigNumber(unitMap[unit], 10));

    return (isBigNumber) ? number : number.toString(10);
};


/**
Checks if the given string is a valid ethereum HEX address.

@method isAddress
@param {String} address the given HEX adress
@return {Boolean}
*/
var isAddress = function(address) {
    if(address.indexOf('0x') === 0 && address.length !== 42)
        return false;
    if(address.indexOf('0x') === -1 && address.length !== 40)
        return false;

    return /^\w+$/.test(address);
};

var isBigNumber = function (value) {
    return value instanceof BigNumber ||
        (value && value.constructor && value.constructor.name === 'BigNumber');
};


module.exports = {
    findIndex: findIndex,
    toDecimal: toDecimal,
    fromDecimal: fromDecimal,
    toAscii: toAscii,
    fromAscii: fromAscii,
    extractDisplayName: extractDisplayName,
    extractTypeName: extractTypeName,
    filterFunctions: filterFunctions,
    filterEvents: filterEvents,
    toEth: toEth,
    toWei: toWei,
    fromWei: fromWei,
    isAddress: isAddress,
    isBigNumber: isBigNumber
};


},{"./const":2}],17:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file watches.js
 * @authors:
 *   Marek Kotewicz <marek@ethdev.com>
 * @date 2015
 */

/// @returns an array of objects describing web3.eth.filter api methods
var eth = function () {
    var newFilter = function (args) {
        return typeof args[0] === 'string' ? 'eth_newFilterString' : 'eth_newFilter';
    };

    return [
    { name: 'newFilter', call: newFilter },
    { name: 'uninstallFilter', call: 'eth_uninstallFilter' },
    { name: 'getLogs', call: 'eth_filterLogs' }
    ];
};

/// @returns an array of objects describing web3.shh.watch api methods
var shh = function () {
    return [
    { name: 'newFilter', call: 'shh_newFilter' },
    { name: 'uninstallFilter', call: 'shh_uninstallFilter' },
    { name: 'getLogs', call: 'shh_getMessages' }
    ];
};

module.exports = {
    eth: eth,
    shh: shh
};


},{}],18:[function(require,module,exports){
/*
    This file is part of ethereum.js.

    ethereum.js is free software: you can redistribute it and/or modify
    it under the terms of the GNU Lesser General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    ethereum.js is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public License
    along with ethereum.js.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file web3.js
 * @authors:
 *   Jeffrey Wilcke <jeff@ethdev.com>
 *   Marek Kotewicz <marek@ethdev.com>
 *   Marian Oancea <marian@ethdev.com>
 *   Gav Wood <g@ethdev.com>
 * @date 2014
 */

// if (process.env.NODE_ENV !== 'build') {
//     var BigNumber = require('bignumber.js');
// }

var eth = require('./eth');
var db = require('./db');
var shh = require('./shh');
var watches = require('./watches');
var filter = require('./filter');
var utils = require('./utils');
var requestManager = require('./requestmanager');

/// @returns an array of objects describing web3 api methods
var web3Methods = function () {
    return [
    { name: 'sha3', call: 'web3_sha3' }
    ];
};

/// creates methods in a given object based on method description on input
/// setups api calls for these methods
var setupMethods = function (obj, methods) {
    methods.forEach(function (method) {
        // allow for object methods 'myObject.method'
        var objectMethods = method.name.split('.'),
            callFunction = function () {
                var args = Array.prototype.slice.call(arguments);
                var call = typeof method.call === 'function' ? method.call(args) : method.call;

                // show deprecated warning
                if(method.newMethod)
                    console.warn('This method is deprecated please use eth.'+ method.newMethod +'() instead.');

                return web3.manager.send({
                    method: call,
                    params: args,
                    outputFormatter: method.outputFormatter,
                    inputFormatter: method.inputFormatter
                });
            };

        if(objectMethods.length > 1) {
            if(!obj[objectMethods[0]])
                obj[objectMethods[0]] = {};

            obj[objectMethods[0]][objectMethods[1]] = callFunction;
        
        } else {

            obj[objectMethods[0]] = callFunction;
        }

    });
};

/// creates properties in a given object based on properties description on input
/// setups api calls for these properties
var setupProperties = function (obj, properties) {
    properties.forEach(function (property) {
        var proto = {};
        proto.get = function () {

            // show deprecated warning
            if(property.newProperty)
                console.warn('This property is deprecated please use eth.'+ property.newProperty +' instead.');


            return web3.manager.send({
                method: property.getter,
                outputFormatter: property.outputFormatter
            });
        };

        if (property.setter) {
            proto.set = function (val) {

                // show deprecated warning
                if(property.newProperty)
                    console.warn('This property is deprecated please use eth.'+ property.newProperty +' instead.');

                return web3.manager.send({
                    method: property.setter,
                    params: [val],
                    inputFormatter: property.inputFormatter
                });
            };
        }

        proto.enumerable = !property.newProperty;
        Object.defineProperty(obj, property.name, proto);

    });
};

/*jshint maxparams:4 */
var startPolling = function (method, id, callback, uninstall) {
    web3.manager.startPolling({
        method: method, 
        params: [id]
    }, id,  callback, uninstall); 
};
/*jshint maxparams:3 */

var stopPolling = function (id) {
    web3.manager.stopPolling(id);
};

var ethWatch = {
    startPolling: startPolling.bind(null, 'eth_changed'), 
    stopPolling: stopPolling
};

var shhWatch = {
    startPolling: startPolling.bind(null, 'shh_changed'), 
    stopPolling: stopPolling
};

/// setups web3 object, and it's in-browser executed methods
var web3 = {
    manager: requestManager(),
    providers: {},

    setProvider: function (provider) {
        web3.manager.setProvider(provider);
    },
    
    /// Should be called to reset state of web3 object
    /// Resets everything except manager
    reset: function () {
        web3.manager.reset(); 
    },

    /// @returns ascii string representation of hex value prefixed with 0x
    toAscii: utils.toAscii,

    /// @returns hex representation (prefixed by 0x) of ascii string
    fromAscii: utils.fromAscii,

    /// @returns decimal representaton of hex value prefixed by 0x
    toDecimal: utils.toDecimal,

    /// @returns hex representation (prefixed by 0x) of decimal value
    fromDecimal: utils.fromDecimal,

    /// used to transform value/string to eth string
    toEth: utils.toEth,

    toWei: utils.toWei,
    fromWei: utils.fromWei,
    isAddress: utils.isAddress,


    /// eth object prototype
    eth: {
        // DEPRECATED
        contractFromAbi: function (abi) {
            console.warn('Initiating a contract like this is deprecated please use var MyContract = eth.contract(abi); new MyContract(address); instead.');

            return function(addr) {
                // Default to address of Config. TODO: rremove prior to genesis.
                addr = addr || '0xc6d9d2cd449a754c494264e1809c50e34d64562b';
                var ret = web3.eth.contract(addr, abi);
                ret.address = addr;
                return ret;
            };
        },

        /// @param filter may be a string, object or event
        /// @param eventParams is optional, this is an object with optional event eventParams params
        /// @param options is optional, this is an object with optional event options ('max'...)
        /// TODO: fix it, 4 params? no way
        /*jshint maxparams:4 */
        filter: function (fil, eventParams, options, formatter) {

            // if its event, treat it differently
            if (fil._isEvent)
                return fil(eventParams, options);

            return filter(fil, ethWatch, formatter);
        },
        // DEPRECATED
        watch: function (fil, eventParams, options, formatter) {
            console.warn('eth.watch() is deprecated please use eth.filter() instead.');
            return this.filter(fil, eventParams, options, formatter);
        }
        /*jshint maxparams:3 */
    },

    /// db object prototype
    db: {},

    /// shh object prototype
    shh: {
        /// @param filter may be a string, object or event
        filter: function (fil) {
            return filter(fil, shhWatch);
        },
        // DEPRECATED
        watch: function (fil) {
            console.warn('shh.watch() is deprecated please use shh.filter() instead.');
            return this.filter(fil);
        }
    }
};

/// setups all api methods
setupMethods(web3, web3Methods());
setupMethods(web3.eth, eth.methods);
setupProperties(web3.eth, eth.properties);
setupMethods(web3.db, db.methods());
setupMethods(web3.shh, shh.methods());
setupMethods(ethWatch, watches.eth());
setupMethods(shhWatch, watches.shh());

module.exports = web3;


},{"./db":4,"./eth":5,"./filter":7,"./requestmanager":12,"./shh":13,"./utils":16,"./watches":17}],"web3":[function(require,module,exports){
var web3 = require('./lib/web3');
web3.providers.HttpSyncProvider = require('./lib/httpsync');
web3.providers.QtSyncProvider = require('./lib/qtsync');
web3.eth.contract = require('./lib/contract');
web3.abi = require('./lib/abi');

module.exports = web3;

},{"./lib/abi":1,"./lib/contract":3,"./lib/httpsync":9,"./lib/qtsync":11,"./lib/web3":18}]},{},["web3"])


//# sourceMappingURL=ethereum.js.map
