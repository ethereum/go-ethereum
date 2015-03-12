#include <Python.h>
#include <alloca.h>
#include <stdint.h>
#include "../libethash/ethash.h"

static PyObject*
get_cache_size(PyObject* self, PyObject* args)
{
    unsigned long block_number;
    if (!PyArg_ParseTuple(args, "k", &block_number))
      return 0;
    return Py_BuildValue("i", ethash_get_cachesize(block_number));
}

static PyObject*
get_full_size(PyObject* self, PyObject* args)
{
    unsigned long block_number;
    if (!PyArg_ParseTuple(args, "k", &block_number))
      return 0;
    return Py_BuildValue("i", ethash_get_datasize(block_number));
}


static PyObject*
mkcache(PyObject* self, PyObject* args)
{
    char * seed;
    unsigned long cache_size;
    int seed_len;

    if (!PyArg_ParseTuple(args, "ks#", &cache_size, &seed, &seed_len))
      return 0;

    if (seed_len != 32)
    {
      PyErr_SetString(PyExc_ValueError,
                      "Seed must be 32 bytes long");
      return 0;
    }

    printf("cache size: %lu\n", cache_size);
    ethash_params params;
    params.cache_size = (size_t) cache_size;
    ethash_cache cache;
    cache.mem = alloca(cache_size);
    ethash_mkcache(&cache, &params, (uint8_t *) seed);
    return PyString_FromStringAndSize(cache.mem, cache_size);
}


static PyMethodDef CoreMethods[] =
{
     {"get_cache_size", get_cache_size, METH_VARARGS, "Get the cache size for a given block number"},
     {"get_full_size", get_full_size, METH_VARARGS, "Get the full size for a given block number"},
     {"mkcache", mkcache, METH_VARARGS, "Makes the cache for given parameters and seed hash"},
     {NULL, NULL, 0, NULL}
};

PyMODINIT_FUNC
initcore(void)
{
     (void) Py_InitModule("core", CoreMethods);
}
