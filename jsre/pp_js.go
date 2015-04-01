package jsre

const pp_js = `
function pp(object, indent) {
    var str = "";
    /*
    var o = object;
    try {
	object = JSON.stringify(object)
	object = JSON.parse(object);
   } catch(e) {
	object = o;
   }
   */

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
        str += "\033[31m" + "Error";
    } else if(typeof(object) === "object") {
        str += "{\n";
        indent += "  ";
        var last = Object.getOwnPropertyNames(object).pop()
        Object.getOwnPropertyNames(object).forEach(function (k) {
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
