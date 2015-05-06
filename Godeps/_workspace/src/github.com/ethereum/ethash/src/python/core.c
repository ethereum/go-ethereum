#include <Python.h>
#include <alloca.h>
#include <stdint.h>
#include <stdlib.h>
#include <time.h>
#include "../libethash/ethash.h"

#if PY_MAJOR_VERSION >= 3
#define PY_STRING_FORMAT "y#"
#define PY_CONST_STRING_FORMAT "y"
#else
#define PY_STRING_FORMAT "s#"
#define PY_CONST_STRING_FORMAT "s"
#endif

#define MIX_WORDS (ETHASH_MIX_BYTES/4)

static PyObject *
get_cache_size(PyObject *self, PyObject *args) {
    unsigned long block_number;
    if (!PyArg_ParseTuple(args, "k", &block_number))
        return 0;
    if (block_number >= ETHASH_EPOCH_LENGTH * 2048) {
        char error_message[1024];
        sprintf(error_message, "Block number must be less than %i (was %lu)", ETHASH_EPOCH_LENGTH * 2048, block_number);

        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    return Py_BuildValue("i", ethash_get_cachesize(block_number));
}

static PyObject *
get_full_size(PyObject *self, PyObject *args) {
    unsigned long block_number;
    if (!PyArg_ParseTuple(args, "k", &block_number))
        return 0;
    if (block_number >= ETHASH_EPOCH_LENGTH * 2048) {
        char error_message[1024];
        sprintf(error_message, "Block number must be less than %i (was %lu)", ETHASH_EPOCH_LENGTH * 2048, block_number);

        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    return Py_BuildValue("i", ethash_get_datasize(block_number));
}


static PyObject *
mkcache_bytes(PyObject *self, PyObject *args) {
    char *seed;
    unsigned long cache_size;
    int seed_len;

    if (!PyArg_ParseTuple(args, "k" PY_STRING_FORMAT, &cache_size, &seed, &seed_len))
        return 0;

    if (seed_len != 32) {
        char error_message[1024];
        sprintf(error_message, "Seed must be 32 bytes long (was %i)", seed_len);

        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    ethash_params params;
    params.cache_size = (size_t) cache_size;
    ethash_cache cache;
    cache.mem = malloc(cache_size);
    ethash_mkcache(&cache, &params, (ethash_h256_t *) seed);
    PyObject * val = Py_BuildValue(PY_STRING_FORMAT, cache.mem, cache_size);
    free(cache.mem);
    return val;
}


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
}

// hashimoto_light(full_size, cache, header, nonce)
static PyObject *
hashimoto_light(PyObject *self, PyObject *args) {
    char *cache_bytes;
    char *header;
    unsigned long full_size;
    unsigned long long nonce;
    int cache_size, header_size;
    if (!PyArg_ParseTuple(args, "k" PY_STRING_FORMAT PY_STRING_FORMAT "K", &full_size, &cache_bytes, &cache_size, &header, &header_size, &nonce))
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
    if (header_size != 32) {
        char error_message[1024];
        sprintf(error_message, "Seed must be 32 bytes long (was %i)", header_size);
        PyErr_SetString(PyExc_ValueError, error_message);
        return 0;
    }

    ethash_return_value out;
    ethash_params params;
    params.cache_size = (size_t) cache_size;
    params.full_size = (size_t) full_size;
    ethash_cache cache;
    cache.mem = (void *) cache_bytes;
    ethash_light(&out, &cache, &params, (ethash_h256_t *) header, nonce);
    return Py_BuildValue("{" PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT "," PY_CONST_STRING_FORMAT ":" PY_STRING_FORMAT "}",
                         "mix digest", &out.mix_hash, 32,
                         "result", &out.result, 32);
}

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
                {"get_cache_size", get_cache_size, METH_VARARGS,
                        "get_cache_size(block_number)\n\n"
                                "Get the cache size for a given block number\n"
                                "\nExample:\n"
                                ">>> get_cache_size(0)\n"
                                "1048384"},
                {"get_full_size", get_full_size, METH_VARARGS,
                        "get_full_size(block_number)\n\n"
                                "Get the full size for a given block number\n"
                                "\nExample:\n"
                                ">>> get_full_size(0)\n"
                                "1073739904"
                },
                {"get_seedhash", get_seedhash, METH_VARARGS,
                        "get_seedhash(block_number)\n\n"
                                "Gets the seedhash for a block."},
                {"mkcache_bytes", mkcache_bytes, METH_VARARGS,
                        "mkcache_bytes(size, header)\n\n"
                                "Makes a byte array for the cache for given cache size and seed hash\n"
                                "\nExample:\n"
                                ">>> pyethash.mkcache_bytes( 1024, \"~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\").encode('hex')"
                                "\"2da2b506f21070e1143d908e867962486d6b0a02e31d468fd5e3a7143aafa76a14201f63374314e2a6aaf84ad2eb57105dea3378378965a1b3873453bb2b78f9a8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995ca8620b2ebeca41fbc773bb837b5e724d6eb2de570d99858df0d7d97067fb8103b21757873b735097b35d3bea8fd1c359a9e8a63c1540c76c9784cf8d975e995c259440b89fa3481c2c33171477c305c8e1e421f8d8f6d59585449d0034f3e421808d8da6bbd0b6378f567647cc6c4ba6c434592b198ad444e7284905b7c6adaf70bf43ec2daa7bd5e8951aa609ab472c124cf9eba3d38cff5091dc3f58409edcc386c743c3bd66f92408796ee1e82dd149eaefbf52b00ce33014a6eb3e50625413b072a58bc01da28262f42cbe4f87d4abc2bf287d15618405a1fe4e386fcdafbb171064bd99901d8f81dd6789396ce5e364ac944bbbd75a7827291c70b42d26385910cd53ca535ab29433dd5c5714d26e0dce95514c5ef866329c12e958097e84462197c2b32087849dab33e88b11da61d52f9dbc0b92cc61f742c07dbbf751c49d7678624ee60dfbe62e5e8c47a03d8247643f3d16ad8c8e663953bcda1f59d7e2d4a9bf0768e789432212621967a8f41121ad1df6ae1fa78782530695414c6213942865b2730375019105cae91a4c17a558d4b63059661d9f108362143107babe0b848de412e4da59168cce82bfbff3c99e022dd6ac1e559db991f2e3f7bb910cefd173e65ed00a8d5d416534e2c8416ff23977dbf3eb7180b75c71580d08ce95efeb9b0afe904ea12285a392aff0c8561ff79fca67f694a62b9e52377485c57cc3598d84cac0a9d27960de0cc31ff9bbfe455acaa62c8aa5d2cce96f345da9afe843d258a99c4eaf3650fc62efd81c7b81cd0d534d2d71eeda7a6e315d540b4473c80f8730037dc2ae3e47b986240cfc65ccc565f0d8cde0bc68a57e39a271dda57440b3598bee19f799611d25731a96b5dbbbefdff6f4f656161462633030d62560ea4e9c161cf78fc96a2ca5aaa32453a6c5dea206f766244e8c9d9a8dc61185ce37f1fc804459c5f07434f8ecb34141b8dcae7eae704c950b55556c5f40140c3714b45eddb02637513268778cbf937a33e4e33183685f9deb31ef54e90161e76d969587dd782eaa94e289420e7c2ee908517f5893a26fdb5873d68f92d118d4bcf98d7a4916794d6ab290045e30f9ea00ca547c584b8482b0331ba1539a0f2714fddc3a0b06b0cfbb6a607b8339c39bcfd6640b1f653e9d70ef6c985b\""},
                {"calc_dataset_bytes", calc_dataset_bytes, METH_VARARGS,
                        "calc_dataset_bytes(full_size, cache_bytes)\n\n"
                                "Makes a byte array for the dataset for a given size given cache bytes"},
                {"hashimoto_light", hashimoto_light, METH_VARARGS,
                        "hashimoto_light(full_size, cache_bytes, header, nonce)\n\n"
                                "Runs the hashimoto hashing function just using cache bytes. Takes an int (full_size), byte array (cache_bytes), another byte array (header), and an int (nonce). Returns an object containing the mix digest, and hash result."},
                {"hashimoto_full", hashimoto_full, METH_VARARGS,
                        "hashimoto_full(dataset_bytes, header, nonce)\n\n"
                                "Runs the hashimoto hashing function using the dataset bytes. Useful for testing. Returns an object containing the mix digest (byte array), and hash result (another byte array)."},
                {"mine", mine, METH_VARARGS,
                        "mine(dataset_bytes, header, difficulty_bytes)\n\n"
                                "Mine for an adequate header. Returns an object containing the mix digest (byte array), hash result (another byte array) and nonce (an int)."},
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
