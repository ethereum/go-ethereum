#ifndef SRC_NODE_INTERNALS_H_
#define SRC_NODE_INTERNALS_H_

//
// This is a stripped down shim to allow node_api.cc to build outside of the node source tree.
//

#include "node_version.h"
#include "util-inl.h"
#include <stdio.h>
#include <stdint.h>
#include "uv.h"
#include "node.h"
#include <string>

// Windows 8+ does not like abort() in Release mode
#ifdef _WIN32
#define ABORT_NO_BACKTRACE() raise(SIGABRT)
#else
#define ABORT_NO_BACKTRACE() abort()
#endif

#define ABORT() node::Abort()

#ifdef __GNUC__
#define LIKELY(expr) __builtin_expect(!!(expr), 1)
#define UNLIKELY(expr) __builtin_expect(!!(expr), 0)
#define PRETTY_FUNCTION_NAME __PRETTY_FUNCTION__
#else
#define LIKELY(expr) expr
#define UNLIKELY(expr) expr
#define PRETTY_FUNCTION_NAME ""
#endif

#define STRINGIFY_(x) #x
#define STRINGIFY(x) STRINGIFY_(x)

#define CHECK(expr)                                                           \
  do {                                                                        \
    if (UNLIKELY(!(expr))) {                                                  \
      static const char* const args[] = { __FILE__, STRINGIFY(__LINE__),      \
                                          #expr, PRETTY_FUNCTION_NAME };      \
      node::Assert(&args);                                                    \
    }                                                                         \
  } while (0)

#define CHECK_EQ(a, b) CHECK((a) == (b))
#define CHECK_GE(a, b) CHECK((a) >= (b))
#define CHECK_GT(a, b) CHECK((a) > (b))
#define CHECK_LE(a, b) CHECK((a) <= (b))
#define CHECK_LT(a, b) CHECK((a) < (b))
#define CHECK_NE(a, b) CHECK((a) != (b))

#ifdef __GNUC__
#define NO_RETURN __attribute__((noreturn))
#else
#define NO_RETURN
#endif

#ifndef NODE_RELEASE
#define NODE_RELEASE "node"
#endif

#if NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 6
class CallbackScope {
  public:
    CallbackScope(void *work);
};
#endif // NODE_MAJOR_VERSION < 8

namespace node {

// Copied from Node.js' src/node_persistent.h
template <typename T>
struct ResetInDestructorPersistentTraits {
  static const bool kResetInDestructor = true;
  template <typename S, typename M>
  // Disallow copy semantics by leaving this unimplemented.
  inline static void Copy(
      const v8::Persistent<S, M>&,
      v8::Persistent<T, ResetInDestructorPersistentTraits<T>>*);
};

// v8::Persistent does not reset the object slot in its destructor.  That is
// acknowledged as a flaw in the V8 API and expected to change in the future
// but for now node::Persistent is the easier and safer alternative.
template <typename T>
using Persistent = v8::Persistent<T, ResetInDestructorPersistentTraits<T>>;

#if NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 2
typedef int async_id;

typedef struct async_context {
  node::async_id async_id;
  node::async_id trigger_async_id;
} async_context;
#endif // NODE_MAJOR_VERSION < 8.2

#if NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 6
NODE_EXTERN async_context EmitAsyncInit(v8::Isolate* isolate,
                                        v8::Local<v8::Object> resource,
                                        v8::Local<v8::String> name,
                                        async_id trigger_async_id = -1);

NODE_EXTERN void EmitAsyncDestroy(v8::Isolate* isolate,
                                  async_context asyncContext);

v8::MaybeLocal<v8::Value> MakeCallback(v8::Isolate* isolate,
                                       v8::Local<v8::Object> recv,
                                       v8::Local<v8::Function> callback,
                                       int argc,
                                       v8::Local<v8::Value>* argv,
                                       async_context asyncContext);

#if NODE_MAJOR_VERSION < 8
class AsyncResource {
  public:
    AsyncResource(v8::Isolate* isolate,
                  v8::Local<v8::Object> object,
                  const char *name);
};
#endif // node version below 8

#endif // node version below 8.6

// The slightly odd function signature for Assert() is to ease
// instruction cache pressure in calls from ASSERT and CHECK.
NO_RETURN void Abort();
NO_RETURN void Assert(const char* const (*args)[4]);
void DumpBacktrace(FILE* fp);

template <typename T, size_t N>
constexpr size_t arraysize(const T(&)[N]) { return N; }

NO_RETURN void FatalError(const char* location, const char* message);

}  // namespace node

#if NODE_MAJOR_VERSION < 8
#define NewTarget This
#endif // NODE_MAJOR_VERSION < 8

#if NODE_MAJOR_VERSION < 6
namespace v8 {
  namespace Private {
    v8::Local<v8::Name> ForApi(v8::Isolate* isolate, v8::Local<v8::String> key);
  }
}
#define GetPrivate(context, key) Get((context), (key))
#define SetPrivate(context, key, value)                                 \
  DefineOwnProperty((context), (key), (value),                          \
                    static_cast<v8::PropertyAttribute>(v8::DontEnum |   \
                                                       v8::DontDelete | \
                                                       v8::ReadOnly))
#endif // NODE_MAJOR_VERSION < 6

#endif  // SRC_NODE_INTERNALS_H_
