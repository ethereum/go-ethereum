import json
from cffi import FFI
ffi = FFI()
ffi.cdef(
    """
    struct wrapper_call_return {
    char* data;
    int len;
    };
    extern int open_database(char* datadir);
    extern void close_database();
    extern struct wrapper_call_return wrapper_call(char* cargs, int clen);
    """
)
geth_lib = ffi.dlopen("./build/bin/read-only-lib.so")
directory = "/Users/hao/Library/Ethereum/"
directory = directory.encode("utf-8")
geth_lib.open_database(directory)
jsonmsg = {"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", True],"id":1}
jsonmsg = json.dumps(jsonmsg)
jsonmsg = jsonmsg.encode('utf-8')
res = geth_lib.wrapper_call(jsonmsg, len(jsonmsg))
ans = ffi.unpack(res.data, res.len)
print(ans)
geth_lib.close_database()