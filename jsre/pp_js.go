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
        var last = getFields(object).pop()
        getFields(object).forEach(function (k) {
            str += indent + k + ": ";
            try {
                str += pp(object[k], indent);
            } catch (e) {
                str += pp(e, indent);
            }

            if(k !== last) {
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
        str += "\033[35m[Function]";
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
    var result = Object.getOwnPropertyNames(object);
    if (object.constructor && object.constructor.prototype) {
        result = result.concat(Object.getOwnPropertyNames(object.constructor.prototype));
    }
    return result.filter(function (field) {
        return redundantFields.indexOf(field) === -1;
    });
};

var isBigNumber = function (object) {
    return typeof BigNumber !== 'undefined' && object instanceof BigNumber;
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
