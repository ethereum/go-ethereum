package javascript

const jsLib = `
function pp(object) {
    var str = "";

    if(object instanceof Array) {
        str += "[ ";
        for(var i = 0, l = object.length; i < l; i++) {
            str += pp(object[i]);

            if(i < l-1) {
                str += ", ";
            }
        }
        str += " ]";
    } else if(typeof(object) === "object") {
        str += "{ ";
        var last = Object.keys(object).sort().pop()
        for(var k in object) {
            str += k + ": " + pp(object[k]);

            if(k !== last) {
                str += ", ";
            }
        }
        str += " }";
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
	    ret += pp(args[i]) + "\n";
    }
    return ret;
}

var print = prettyPrint;
`
