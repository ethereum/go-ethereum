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

/// @returns an array of objects describing web3 api methods
var web3Methods = function () {
    return [
    { name: 'sha3', call: 'web3_sha3' }
    ];
};

/// @returns an array of objects describing web3.eth api methods
var ethMethods = function () {
    var blockCall = function (args) {
        return typeof args[0] === "string" ? "eth_blockByHash" : "eth_blockByNumber";
    };

    var transactionCall = function (args) {
        return typeof args[0] === "string" ? 'eth_transactionByHash' : 'eth_transactionByNumber';
    };

    var uncleCall = function (args) {
        return typeof args[0] === "string" ? 'eth_uncleByHash' : 'eth_uncleByNumber';
    };

    var methods = [
    { name: 'balanceAt', call: 'eth_balanceAt' },
    { name: 'stateAt', call: 'eth_stateAt' },
    { name: 'storageAt', call: 'eth_storageAt' },
    { name: 'countAt', call: 'eth_countAt'},
    { name: 'codeAt', call: 'eth_codeAt' },
    { name: 'transact', call: 'eth_transact' },
    { name: 'call', call: 'eth_call' },
    { name: 'block', call: blockCall },
    { name: 'transaction', call: transactionCall },
    { name: 'uncle', call: uncleCall },
    { name: 'compilers', call: 'eth_compilers' },
    { name: 'lll', call: 'eth_lll' },
    { name: 'solidity', call: 'eth_solidity' },
    { name: 'serpent', call: 'eth_serpent' },
    { name: 'logs', call: 'eth_logs' }
    ];
    return methods;
};

/// @returns an array of objects describing web3.eth api properties
var ethProperties = function () {
    return [
    { name: 'coinbase', getter: 'eth_coinbase', setter: 'eth_setCoinbase' },
    { name: 'listening', getter: 'eth_listening', setter: 'eth_setListening' },
    { name: 'mining', getter: 'eth_mining', setter: 'eth_setMining' },
    { name: 'gasPrice', getter: 'eth_gasPrice' },
    { name: 'account', getter: 'eth_account' },
    { name: 'accounts', getter: 'eth_accounts' },
    { name: 'peerCount', getter: 'eth_peerCount' },
    { name: 'defaultBlock', getter: 'eth_defaultBlock', setter: 'eth_setDefaultBlock' },
    { name: 'number', getter: 'eth_number'}
    ];
};

/// @returns an array of objects describing web3.db api methods
var dbMethods = function () {
    return [
    { name: 'put', call: 'db_put' },
    { name: 'get', call: 'db_get' },
    { name: 'putString', call: 'db_putString' },
    { name: 'getString', call: 'db_getString' }
    ];
};

/// @returns an array of objects describing web3.shh api methods
var shhMethods = function () {
    return [
    { name: 'post', call: 'shh_post' },
    { name: 'newIdentity', call: 'shh_newIdentity' },
    { name: 'haveIdentity', call: 'shh_haveIdentity' },
    { name: 'newGroup', call: 'shh_newGroup' },
    { name: 'addToGroup', call: 'shh_addToGroup' }
    ];
};

/// @returns an array of objects describing web3.eth.watch api methods
var ethWatchMethods = function () {
    var newFilter = function (args) {
        return typeof args[0] === 'string' ? 'eth_newFilterString' : 'eth_newFilter';
    };

    return [
    { name: 'newFilter', call: newFilter },
    { name: 'uninstallFilter', call: 'eth_uninstallFilter' },
    { name: 'getMessages', call: 'eth_filterLogs' }
    ];
};

/// @returns an array of objects describing web3.shh.watch api methods
var shhWatchMethods = function () {
    return [
    { name: 'newFilter', call: 'shh_newFilter' },
    { name: 'uninstallFilter', call: 'shh_uninstallFilter' },
    { name: 'getMessage', call: 'shh_getMessages' }
    ];
};

/// creates methods in a given object based on method description on input
/// setups api calls for these methods
var setupMethods = function (obj, methods) {
    methods.forEach(function (method) {
        obj[method.name] = function () {
            var args = Array.prototype.slice.call(arguments);
            var call = typeof method.call === 'function' ? method.call(args) : method.call;
            return web3.provider.send({
                call: call,
                args: args
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
            return web3.provider.send({
                call: property.getter
            });
        };

        if (property.setter) {
            proto.set = function (val) {
                return web3.provider.send({
                    call: property.setter,
                    args: [val]
                });
            };
        }
        Object.defineProperty(obj, property.name, proto);
    });
};

// TODO: import from a dependency, don't duplicate.
// TODO: use bignumber for that!
var hexToDec = function (hex) {
    return parseInt(hex, 16).toString();
};

var decToHex = function (dec) {
    return parseInt(dec).toString(16);
};

/// setups web3 object, and it's in-browser executed methods
var web3 = {
    _callbacks: {},
    _events: {},
    providers: {},

    toHex: function(str) {
        var hex = "";
        for(var i = 0; i < str.length; i++) {
            var n = str.charCodeAt(i).toString(16);
            hex += n.length < 2 ? '0' + n : n;
        }

        return hex;
    },

    /// @returns ascii string representation of hex value prefixed with 0x
    toAscii: function(hex) {
        // Find termination
        var str = "";
        var i = 0, l = hex.length;
        if (hex.substring(0, 2) === '0x')
            i = 2;
        for(; i < l; i+=2) {
            var code = parseInt(hex.substr(i, 2), 16);
            if(code === 0) {
                break;
            }

            str += String.fromCharCode(code);
        }

        return str;
    },

    /// @returns hex representation (prefixed by 0x) of ascii string
    fromAscii: function(str, pad) {
        pad = pad === undefined ? 0 : pad;
        var hex = this.toHex(str);
        while(hex.length < pad*2)
            hex += "00";
        return "0x" + hex;
    },

    /// @returns decimal representaton of hex value prefixed by 0x
    toDecimal: function (val) {
        return hexToDec(val.substring(2));
    },

    /// @returns hex representation (prefixed by 0x) of decimal value
    fromDecimal: function (val) {
        return "0x" + decToHex(val);
    },

    /// used to transform value/string to eth string
    toEth: function(str) {
        var val = typeof str === "string" ? str.indexOf('0x') === 0 ? parseInt(str.substr(2), 16) : parseInt(str) : str;
        var unit = 0;
        var units = [ 'wei', 'Kwei', 'Mwei', 'Gwei', 'szabo', 'finney', 'ether', 'grand', 'Mether', 'Gether', 'Tether', 'Pether', 'Eether', 'Zether', 'Yether', 'Nether', 'Dether', 'Vether', 'Uether' ];
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
    },

    /// eth object prototype
    eth: {
        watch: function (params) {
            return new web3.filter(params, ethWatch);
        }
    },

    /// db object prototype
    db: {},

    /// shh object prototype
    shh: {
        watch: function (params) {
            return new web3.filter(params, shhWatch);
        }
    },

    /// @returns true if provider is installed
    haveProvider: function() {
        return !!web3.provider.provider;
    }
};

/// setups all api methods
setupMethods(web3, web3Methods());
setupMethods(web3.eth, ethMethods());
setupProperties(web3.eth, ethProperties());
setupMethods(web3.db, dbMethods());
setupMethods(web3.shh, shhMethods());

var ethWatch = {
    changed: 'eth_changed'
};

setupMethods(ethWatch, ethWatchMethods());

var shhWatch = {
    changed: 'shh_changed'
};

setupMethods(shhWatch, shhWatchMethods());

web3.setProvider = function(provider) {
    //provider.onmessage = messageHandler; // there will be no async calls, to remove
    web3.provider.set(provider);
};

module.exports = web3;

