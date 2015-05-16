#include <Python.h>
#include <alloca.h>
#include <stdint.h>
#include <stdlib.h>
#include <time.h>
#include "../libethash/ethash.h"
#include "../libethash/internal.h"

#if PY_MAJOR_VERSION >= 3
#define PY_STRING_FORMAT "y#"
#define PY_CONST_STRING_FORMAT "y"
#else
#define PY_STRING_FORMAT "s#"
#define PY_CONST_STRING_FORMAT "s"
#endif

#define MIX_WORDS (ETHASH_MIX_BYTES/4)

static PyObject *
mkcache_bytes(PyObject *self, PyObject *args) {
    unsigned long block_number;
    unsigned long cache_size;

    if (!PyArg_ParseTuple(args, "k", &block_number))
        return 0;

    ethash_light_t L = ethash_light_new(block_number);
    PyObject * val = Py_BuildValue(PY_STRING_FORMAT, L->cache, L->cache_size);
    free(L->cache);
    return val;
}

/*
static PyObject *
calc_dataset_bytes(PyObject *self, PyObject *args) {
    char *cache_bytes;
    unsigned long full_size;
    int cache_size;

    if (!PyArg_ParseTuple(args, "k" PY_STRING_FORMAT, &full_size, &cache_bytes, &cache_size))
        return 0;

    if (full_size % MIX_WORDS != 0) {
        char error_message[1024];
        sprintf(error_message, "The size of data set must be a multiple of %i bytes (was %lu)", MIX_WORDS, full_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    if (cache_size % ETHASH_HASH_BYTES != 0) {
        char error_message[1024];
        sprintf(error_message, "The size of the cache must be a multiple of %i bytes (was %i)", ETHASH_HASH_BYTES, cache_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    ethash_params params;
    params.cache_size = (size_t) cache_size;
    params.full_size = (size_t) full_size;
    ethash_cache cache;
    cache.mem = (void *) cache_bytes;
    void *mem = malloc(params.full_size);
    ethash_compute_full_data(mem, &params, &cache);
    PyObject * val = Py_BuildValue(PY_STRING_FORMAT, (char *) mem, full_size);
    free(mem);
    return val;
}*/

// hashimoto_light(full_size, cache, header, nonce)
static PyObject *
hashimoto_light(PyObject *self, PyObject *args) {
    char *cache_bytes;
    char *header;
    unsigned long block_number;
    unsigned long long nonce;
    int cache_size, header_size;
    if (!PyArg_ParseTuple(args, "k" PY_STRING_FORMAT PY_STRING_FORMAT "K", &block_number, &cache_bytes, &cache_size, &header, &header_size, &nonce))
        return 0;
    if (header_size != 32) {
        char error_message[1024];
        sprintf(error_message, "Seed must be 32 bytes long (was %i)", header_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }
    struct ethash_light *s;
    s = calloc(sizeof(*s), 1);
    s->cache = cache_bytes;
    s->cache_size = cache_size;
    s->block_number = block_number;
    struct ethash_h256 *h;
    h = calloc(sizeof(*h), 1);
    for (int i = 0; i < 32; i++) h->b[i] = header[i];
    struct ethash_return_value out = ethash_light_compute(s, *h, nonce);
    return Py_BuildValue("{" PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT "," PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT "}",
                         "mix digest", &out.mix_hash, 32,
                         "result", &out.result, 32);
}
/*
// hashimoto_full(dataset, header, nonce)
static PyObject *
hashimoto_full(PyObject *self, PyObject *args) {
    char *full_bytes;
    char *header;
    unsigned long long nonce;
    int full_size, header_size;

    if (!PyArg_ParseTuple(args, PY_STRING_FORMAT PY_STRING_FORMAT "K", &full_bytes, &full_size, &header, &header_size, &nonce))
        return 0;

    if (full_size % MIX_WORDS != 0) {
        char error_message[1024];
        sprintf(error_message, "The size of data set must be a multiple of %i bytes (was %i)", MIX_WORDS, full_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    if (header_size != 32) {
        char error_message[1024];
        sprintf(error_message, "Header must be 32 bytes long (was %i)", header_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }


    ethash_return_value out;
    ethash_params params;
    params.full_size = (size_t) full_size;
    ethash_full(&out, (void *) full_bytes, &params, (ethash_h256_t *) header, nonce);
    return Py_BuildValue("{" PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT ", " PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT "}",
                         "mix digest", &out.mix_hash, 32,
                         "result", &out.result, 32);
}

// mine(dataset_bytes, header, difficulty_bytes)
static PyObject *
mine(PyObject *self, PyObject *args) {
    char *full_bytes;
    char *header;
    char *difficulty;
    srand(time(0));
    uint64_t nonce = ((uint64_t) rand()) << 32 | rand();
    int full_size, header_size, difficulty_size;

    if (!PyArg_ParseTuple(args, PY_STRING_FORMAT PY_STRING_FORMAT PY_STRING_FORMAT, &full_bytes, &full_size, &header, &header_size, &difficulty, &difficulty_size))
        return 0;

    if (full_size % MIX_WORDS != 0) {
        char error_message[1024];
        sprintf(error_message, "The size of data set must be a multiple of %i bytes (was %i)", MIX_WORDS, full_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    if (header_size != 32) {
        char error_message[1024];
        sprintf(error_message, "Header must be 32 bytes long (was %i)", header_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    if (difficulty_size != 32) {
        char error_message[1024];
        sprintf(error_message, "Difficulty must be an array of 32 bytes (only had %i)", difficulty_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    ethash_return_value out;
    ethash_params params;
    params.full_size = (size_t) full_size;

    // TODO: Multi threading?
    do {
        ethash_full(&out, (void *) full_bytes, &params, (const ethash_h256_t *) header, nonce++);
        // TODO: disagrees with the spec https://github.com/ethereum/wiki/wiki/Ethash#mining
    } while (!ethash_check_difficulty(&out.result, (const ethash_h256_t *) difficulty));

    return Py_BuildValue("{" PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT ", " PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT ", " PY_CONST_STRING_FORMAT ":K}",
            "mix digest", &out.mix_hash, 32,
            "result", &out.result, 32,
            "nonce", nonce);
}
*/

//get_seedhash(block_number)
static PyObject *
get_seedhash(PyObject *self, PyObject *args) {
    unsigned long block_number;
    if (!PyArg_ParseTuple(args, "k", &block_number))
        return 0;
    if (block_number >= ETHASH_EPOCH_LENGTH * 2048) {
        char error_message[1024];
        sprintf(error_message, "Block number must be less than %i (was %lu)", ETHASH_EPOCH_LENGTH * 2048, block_number);

        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }
    ethash_h256_t seedhash = ethash_get_seedhash(block_number);
    return Py_BuildValue(PY_STRING_FORMAT, (char *) &seedhash, 32);
}

static PyMethodDef PyethashMethods[] =
        {
                {"get_seedhash", get_seedhash, METH_VARARGS,
                        "get_seedhash(block_number)\n\n"
                                "Gets the seedhash for a block."},
                {"mkcache_bytes", mkcache_bytes, METH_VARARGS,
                        "mkcache_bytes(block_number)\n\n"
                                "Makes a byte array for the cache for given block number\n"},
                /*{"calc_dataset_bytes", calc_dataset_bytes, METH_VARARGS,
                        "calc_dataset_bytes(full_size, cache_bytes)\n\n"
                                "Makes a byte array for the dataset for a given size given cache bytes"},*/
                {"hashimoto_light", hashimoto_light, METH_VARARGS,
                        "hashimoto_light(block_number, cache_bytes, header, nonce)\n\n"
                                "Runs the hashimoto hashing function just using cache bytes. Takes an int (full_size), byte array (cache_bytes), another byte array (header), and an int (nonce). Returns an object containing the mix digest, and hash result."},
                /*{"hashimoto_full", hashimoto_full, METH_VARARGS,
                        "hashimoto_full(dataset_bytes, header, nonce)\n\n"
                                "Runs the hashimoto hashing function using the dataset bytes. Useful for testing. Returns an object containing the mix digest (byte array), and hash result (another byte array)."},
                {"mine", mine, METH_VARARGS,
                        "mine(dataset_bytes, header, difficulty_bytes)\n\n"
                                "Mine for an adequate header. Returns an object containing the mix digest (byte array), hash result (another byte array) and nonce (an int)."},*/
                {NULL, NULL, 0, NULL}
        };

#if PY_MAJOR_VERSION >= 3
static struct PyModuleDef PyethashModule = {
    PyModuleDef_HEAD_INIT,
    "pyethash",
    "...",
    -1,
    PyethashMethods
};

PyMODINIT_FUNC PyInit_pyethash(void) {
    PyObject *module =  PyModule_Create(&PyethashModule);
    // Following Spec: https://github.com/ethereum/wiki/wiki/Ethash#definitions
    PyModule_AddIntConstant(module, "REVISION", (long) ETHASH_REVISION);
    PyModule_AddIntConstant(module, "DATASET_BYTES_INIT", (long) ETHASH_DATASET_BYTES_INIT);
    PyModule_AddIntConstant(module, "DATASET_BYTES_GROWTH", (long) ETHASH_DATASET_BYTES_GROWTH);
    PyModule_AddIntConstant(module, "CACHE_BYTES_INIT", (long) ETHASH_CACHE_BYTES_INIT);
    PyModule_AddIntConstant(module, "CACHE_BYTES_GROWTH", (long) ETHASH_CACHE_BYTES_GROWTH);
    PyModule_AddIntConstant(module, "EPOCH_LENGTH", (long) ETHASH_EPOCH_LENGTH);
    PyModule_AddIntConstant(module, "MIX_BYTES", (long) ETHASH_MIX_BYTES);
    PyModule_AddIntConstant(module, "HASH_BYTES", (long) ETHASH_HASH_BYTES);
    PyModule_AddIntConstant(module, "DATASET_PARENTS", (long) ETHASH_DATASET_PARENTS);
    PyModule_AddIntConstant(module, "CACHE_ROUNDS", (long) ETHASH_CACHE_ROUNDS);
    PyModule_AddIntConstant(module, "ACCESSES", (long) ETHASH_ACCESSES);
    return module;
}
#else
PyMODINIT_FUNC
initpyethash(void) {
    PyObject *module = Py_InitModule("pyethash", PyethashMethods);
    // Following Spec: https://github.com/ethereum/wiki/wiki/Ethash#definitions
    PyModule_AddIntConstant(module, "REVISION", (long) ETHASH_REVISION);
    PyModule_AddIntConstant(module, "DATASET_BYTES_INIT", (long) ETHASH_DATASET_BYTES_INIT);
    PyModule_AddIntConstant(module, "DATASET_BYTES_GROWTH", (long) ETHASH_DATASET_BYTES_GROWTH);
    PyModule_AddIntConstant(module, "CACHE_BYTES_INIT", (long) ETHASH_CACHE_BYTES_INIT);
    PyModule_AddIntConstant(module, "CACHE_BYTES_GROWTH", (long) ETHASH_CACHE_BYTES_GROWTH);
    PyModule_AddIntConstant(module, "EPOCH_LENGTH", (long) ETHASH_EPOCH_LENGTH);
    PyModule_AddIntConstant(module, "MIX_BYTES", (long) ETHASH_MIX_BYTES);
    PyModule_AddIntConstant(module, "HASH_BYTES", (long) ETHASH_HASH_BYTES);
    PyModule_AddIntConstant(module, "DATASET_PARENTS", (long) ETHASH_DATASET_PARENTS);
    PyModule_AddIntConstant(module, "CACHE_ROUNDS", (long) ETHASH_CACHE_ROUNDS);
    PyModule_AddIntConstant(module, "ACCESSES", (long) ETHASH_ACCESSES);
}
#endif
