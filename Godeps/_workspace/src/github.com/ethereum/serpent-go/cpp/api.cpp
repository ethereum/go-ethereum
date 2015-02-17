#include <string>

#include "serpent/lllparser.h"
#include "serpent/bignum.h"
#include "serpent/util.h"
#include "serpent/tokenize.h"
#include "serpent/parser.h"
#include "serpent/compiler.h"

#include "cpp/api.h"

const char *compileGo(char *code, int *err)
{
    try {
        std::string c = binToHex(compile(std::string(code)));

        return c.c_str();
    }
    catch(std::string &error) {
        *err = 1;
        return error.c_str();
    }
    catch(...) {
        return "Unknown error";
    }
}
