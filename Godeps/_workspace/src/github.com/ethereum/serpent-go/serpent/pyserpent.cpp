#include <Python.h>
#include "structmember.h"

#include <stdlib.h>
#include <stdio.h>
#include <iostream>
#include "funcs.h"

#define PYMETHOD(name, FROM, method, TO) \
    static PyObject * name(PyObject *, PyObject *args) { \
        try { \
        FROM(med) \
        return TO(method(med)); \
        } \
        catch (std::string e) { \
           PyErr_SetString(PyExc_Exception, e.c_str()); \
           return NULL; \
        } \
    }

#define FROMSTR(v) \
    const char *command; \
    int len; \
    if (!PyArg_ParseTuple(args, "s#", &command, &len)) \
        return NULL; \
    std::string v = std::string(command, len); \

#define FROMNODE(v) \
    PyObject *node; \
    if (!PyArg_ParseTuple(args, "O", &node)) \
        return NULL; \
    Node v = cppifyNode(node);

#define FROMLIST(v) \
    PyObject *node; \
    if (!PyArg_ParseTuple(args, "O", &node)) \
        return NULL; \
    std::vector<Node> v = cppifyNodeList(node);

// Convert metadata into python wrapper form [file, ln, ch]
PyObject* pyifyMetadata(Metadata m) {
    PyObject* a = PyList_New(0);
    PyList_Append(a, Py_BuildValue("s#", m.file.c_str(), m.file.length()));
    PyList_Append(a, Py_BuildValue("i", m.ln));
    PyList_Append(a, Py_BuildValue("i", m.ch));
    return a;
}

// Convert node into python wrapper form 
// [token=0/astnode=1, val, metadata, args]
PyObject* pyifyNode(Node n) {
    PyObject* a = PyList_New(0);
    PyList_Append(a, Py_BuildValue("i", n.type == ASTNODE));
    PyList_Append(a, Py_BuildValue("s#", n.val.c_str(), n.val.length()));
    PyList_Append(a, pyifyMetadata(n.metadata));
    for (unsigned i = 0; i < n.args.size(); i++)
        PyList_Append(a, pyifyNode(n.args[i]));
    return a;
}

// Convert string into python wrapper form
PyObject* pyifyString(std::string s) {
    return Py_BuildValue("s#", s.c_str(), s.length());
}

// Convert list of nodes into python wrapper form
PyObject* pyifyNodeList(std::vector<Node> n) {
    PyObject* a = PyList_New(0);
    for (unsigned i = 0; i < n.size(); i++)
        PyList_Append(a, pyifyNode(n[i]));
    return a;
}

// Convert pyobject int into normal form
int cppifyInt(PyObject* o) {
    int out;
    if (!PyArg_Parse(o, "i", &out))
        err("Argument should be integer", Metadata());
    return out;
}

// Convert pyobject string into normal form
std::string cppifyString(PyObject* o) {
    const char *command;
    if (!PyArg_Parse(o, "s", &command))
        err("Argument should be string", Metadata());
    return std::string(command);
}

// Convert metadata from python wrapper form
Metadata cppifyMetadata(PyObject* o) {
    std::string file = cppifyString(PyList_GetItem(o, 0));
    int ln = cppifyInt(PyList_GetItem(o, 1));
    int ch = cppifyInt(PyList_GetItem(o, 2));
    return Metadata(file, ln, ch);
}

// Convert node from python wrapper form
Node cppifyNode(PyObject* o) {
    Node n;
    int isAstNode = cppifyInt(PyList_GetItem(o, 0));
    n.type = isAstNode ? ASTNODE : TOKEN;
    n.val = cppifyString(PyList_GetItem(o, 1));
    n.metadata = cppifyMetadata(PyList_GetItem(o, 2));
    std::vector<Node> args;
    for (int i = 3; i < PyList_Size(o); i++) {
        args.push_back(cppifyNode(PyList_GetItem(o, i)));
    }
    n.args = args;
    return n;
}

//Convert list of nodes into normal form
std::vector<Node> cppifyNodeList(PyObject* o) {
    std::vector<Node> out;
    for (int i = 0; i < PyList_Size(o); i++) {
        out.push_back(cppifyNode(PyList_GetItem(o,i)));
    }
    return out;
}

PYMETHOD(ps_compile, FROMSTR, compile, pyifyString)
PYMETHOD(ps_compile_chunk, FROMSTR, compileChunk, pyifyString)
PYMETHOD(ps_compile_to_lll, FROMSTR, compileToLLL, pyifyNode)
PYMETHOD(ps_compile_chunk_to_lll, FROMSTR, compileChunkToLLL, pyifyNode)
PYMETHOD(ps_compile_lll, FROMNODE, compileLLL, pyifyString)
PYMETHOD(ps_parse, FROMSTR, parseSerpent, pyifyNode)
PYMETHOD(ps_rewrite, FROMNODE, rewrite, pyifyNode)
PYMETHOD(ps_rewrite_chunk, FROMNODE, rewriteChunk, pyifyNode)
PYMETHOD(ps_pretty_compile, FROMSTR, prettyCompile, pyifyNodeList)
PYMETHOD(ps_pretty_compile_chunk, FROMSTR, prettyCompileChunk, pyifyNodeList)
PYMETHOD(ps_pretty_compile_lll, FROMNODE, prettyCompileLLL, pyifyNodeList)
PYMETHOD(ps_serialize, FROMLIST, serialize, pyifyString)
PYMETHOD(ps_deserialize, FROMSTR, deserialize, pyifyNodeList)
PYMETHOD(ps_parse_lll, FROMSTR, parseLLL, pyifyNode)


static PyMethodDef PyextMethods[] = {
    {"compile",  ps_compile, METH_VARARGS,
        "Compile code."},
    {"compile_chunk",  ps_compile_chunk, METH_VARARGS,
        "Compile code chunk (no wrappers)."},
    {"compile_to_lll",  ps_compile_to_lll, METH_VARARGS,
        "Compile code to LLL."},
    {"compile_chunk_to_lll",  ps_compile_chunk_to_lll, METH_VARARGS,
        "Compile code chunk to LLL (no wrappers)."},
    {"compile_lll",  ps_compile_lll, METH_VARARGS,
        "Compile LLL to EVM."},
    {"parse",  ps_parse, METH_VARARGS,
        "Parse serpent"},
    {"rewrite",  ps_rewrite, METH_VARARGS,
        "Rewrite parsed serpent to LLL"},
    {"rewrite_chunk",  ps_rewrite_chunk, METH_VARARGS,
        "Rewrite parsed serpent to LLL (no wrappers)"},
    {"pretty_compile",  ps_pretty_compile, METH_VARARGS,
        "Compile to EVM opcodes"},
    {"pretty_compile_chunk",  ps_pretty_compile_chunk, METH_VARARGS,
        "Compile chunk to EVM opcodes (no wrappers)"},
    {"pretty_compile_lll",  ps_pretty_compile_lll, METH_VARARGS,
        "Compile LLL to EVM opcodes"},
    {"serialize",  ps_serialize, METH_VARARGS,
        "Convert EVM opcodes to bin"},
    {"deserialize",  ps_deserialize, METH_VARARGS,
        "Convert EVM bin to opcodes"},
    {"parse_lll",  ps_parse_lll, METH_VARARGS,
        "Parse LLL"},
    {NULL, NULL, 0, NULL}        /* Sentinel */
};

PyMODINIT_FUNC initserpent_pyext(void)
{
     Py_InitModule( "serpent_pyext", PyextMethods );
}
