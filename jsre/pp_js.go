// Copyright 2014 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package jsre

const pp_js = `
function pp(object, indent) {
    try {
        JSON.stringify(object)
    } catch(e) {
        return pp(e, indent);
    }

    var str = "";
    if(object instanceof Array) {
        str += "[";
        for(var i = 0, l = object.length; i < l; i++) {
            str += pp(object[i], indent);

            if(i < l-1) {
                str += ", ";
            }
        }
        str += " ]";
    } else if (object instanceof Error) {
        str += "\033[31m" + "Error:\033[0m " + object.message;
    } else if (isBigNumber(object)) {
        str += "\033[32m'" + object.toString(10) + "'";
    } else if(typeof(object) === "object") {
        str += "{\n";
        indent += "  ";

        var fields = getFields(object);
        var last   = fields[fields.length - 1];
        fields.forEach(function (key) {
            str += indent + key + ": ";
            try {
                str += pp(object[key], indent);
            } catch (e) {
                str += pp(e, indent);
            }
            if(key !== last) {
                str += ",";
            }
            str += "\n";
        });
        str += indent.substr(2, indent.length) + "}";
    } else if(typeof(object) === "string") {
        str += "\033[32m'" + object + "'";
    } else if(typeof(object) === "undefined") {
        str += "\033[1m\033[30m" + object;
    } else if(typeof(object) === "number") {
        str += "\033[31m" + object;
    } else if(typeof(object) === "function") {
        str += "\033[35m" + object.toString().split(" {")[0];
    } else {
        str += object;
    }

    str += "\033[0m";

    return str;
}

var redundantFields = [
    'valueOf',
    'toString',
    'toLocaleString',
    'hasOwnProperty',
    'isPrototypeOf',
    'propertyIsEnumerable',
    'constructor'
];

var getFields = function (object) {
    var members = Object.getOwnPropertyNames(object);
    if (object.constructor && object.constructor.prototype) {
        members = members.concat(Object.getOwnPropertyNames(object.constructor.prototype));
    }

    var fields = members.filter(function (member) {
        return !isMemberFunction(object, member)
    }).sort()
    var funcs = members.filter(function (member) {
        return isMemberFunction(object, member)
    }).sort()

    var results = fields.concat(funcs);
    return results.filter(function (field) {
        return redundantFields.indexOf(field) === -1;
    });
};

var isMemberFunction = function(object, member) {
    try {
        return typeof(object[member]) === "function";
    } catch(e) {
        return false;
    }
}

var isBigNumber = function (object) {
    var result = typeof BigNumber !== 'undefined' && object instanceof BigNumber;

    if (!result) {
    	if(typeof(object) === "object") {
			result = object.constructor.toString().indexOf("function BigNumber(") == 0;
		}
    }

    return result
};

function prettyPrint(/* */) {
    var args = arguments;
    var ret = "";
    for(var i = 0, l = args.length; i < l; i++) {
	    ret += pp(args[i], "") + "\n";
    }
    return ret;
}

var print = prettyPrint;
`
