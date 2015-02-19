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

if (process.env.NODE_ENV !== 'build') {
    var BigNumber = require('bignumber.js');
}

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
        obj[method.name] = function () {
            var args = Array.prototype.slice.call(arguments);
            var call = typeof method.call === 'function' ? method.call(args) : method.call;
            return web3.manager.send({
                method: call,
                params: args
            });
        };
    });
};

/// creates properties in a given object based on properties description on input
/// setups api calls for these properties
var setupProperties = function (obj, properties) {
    properties.forEach(function (property) {
        var proto = {};
        proto.get = function () {
            return web3.manager.send({
                method: property.getter
            });
        };

        if (property.setter) {
            proto.set = function (val) {
                return web3.manager.send({
                    method: property.setter,
                    params: [val]
                });
            };
        }
        Object.defineProperty(obj, property.name, proto);
    });
};

var startPolling = function (method, id, callback, uninstall) {
    web3.manager.startPolling({
        method: method, 
        params: [id]
    }, id,  callback, uninstall); 
};

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

    /// @returns ascii string representation of hex value prefixed with 0x
    toAscii: utils.toAscii,

    /// @returns hex representation (prefixed by 0x) of ascii string
    fromAscii: utils.fromAscii,

    /// @returns decimal representaton of hex value prefixed by 0x
    toDecimal: function (val) {
        // remove 0x and place 0, if it's required
        val = val.length > 2 ? val.substring(2) : "0";
        return (new BigNumber(val, 16).toString(10));
    },

    /// @returns hex representation (prefixed by 0x) of decimal value
    fromDecimal: function (val) {
        return "0x" + (new BigNumber(val).toString(16));
    },

    /// used to transform value/string to eth string
    toEth: utils.toEth,

    /// eth object prototype
    eth: {
        contractFromAbi: function (abi) {
            return function(addr) {
                // Default to address of Config. TODO: rremove prior to genesis.
                addr = addr || '0xc6d9d2cd449a754c494264e1809c50e34d64562b';
                var ret = web3.eth.contract(addr, abi);
                ret.address = addr;
                return ret;
            };
        },

        /// @param filter may be a string, object or event
        /// @param indexed is optional, this is an object with optional event indexed params
        /// @param options is optional, this is an object with optional event options ('max'...)
        /// TODO: fix it, 4 params? no way
        watch: function (fil, indexed, options, formatter) {
            if (fil._isEvent) {
                return fil(indexed, options);
            }
            return filter(fil, ethWatch, formatter);
        }
    },

    /// db object prototype
    db: {},

    /// shh object prototype
    shh: {
        /// @param filter may be a string, object or event
        watch: function (fil) {
            return filter(fil, shhWatch);
        }
    },
    setProvider: function (provider) {
        web3.manager.setProvider(provider);
    },
    
    /// Should be called to reset state of web3 object
    /// Resets everything except manager
    reset: function () {
        web3.manager.reset(); 
    }
};

/// setups all api methods
setupMethods(web3, web3Methods());
setupMethods(web3.eth, eth.methods());
setupProperties(web3.eth, eth.properties());
setupMethods(web3.db, db.methods());
setupMethods(web3.shh, shh.methods());
setupMethods(ethWatch, watches.eth());
setupMethods(shhWatch, watches.shh());

module.exports = web3;

