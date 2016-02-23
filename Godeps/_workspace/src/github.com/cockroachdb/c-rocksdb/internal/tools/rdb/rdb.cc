#ifndef BUILDING_NODE_EXTENSION
#define BUILDING_NODE_EXTENSION
#endif

#include <v8.h>
#include <node.h>
#include "db_wrapper.h"

using namespace v8;

void InitAll(Handle<Object> exports) {
  DBWrapper::Init(exports);
}

NODE_MODULE(rdb, InitAll)
