from cffi import FFI
import json

if __name__ == "__main__":
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
    jsonMsg = json.dumps({"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", False],"id":1}).encode("ascii")
    geth_lib = ffi.dlopen("./build/bin/read-only-lib.so")
    geth_lib.open_database("/Users/hao/Library/Ethereum/".encode("ascii"))
    res = geth_lib.wrapper_call(jsonMsg, len(jsonMsg))
    res = ffi.string(res.data, res.len)
    print(res)
    geth_lib.close_database()
