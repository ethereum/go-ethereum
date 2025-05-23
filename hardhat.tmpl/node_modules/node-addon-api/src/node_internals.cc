#include "node_internals.h"
#include <stdlib.h>
#include <cstdarg>
#include <vector>
#include "uv.h"

#if defined(_MSC_VER)
#define getpid GetCurrentProcessId
#else
#include <unistd.h>  // getpid
#endif

#if NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 6
CallbackScope::CallbackScope(void *work) {
}
#endif // NODE_MAJOR_VERSION < 8

namespace node {

#if NODE_MAJOR_VERSION < 8

async_context EmitAsyncInit(v8::Isolate* isolate,
                            v8::Local<v8::Object> resource,
                            v8::Local<v8::String> name,
                            async_id trigger_async_id) {
  return async_context();
}

void EmitAsyncDestroy(v8::Isolate* isolate,
                      async_context asyncContext) {
}

AsyncResource::AsyncResource(v8::Isolate* isolate,
                             v8::Local<v8::Object> object,
                             const char *name) {
}

#endif // NODE_MAJOR_VERSION < 8

#if NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 6

v8::MaybeLocal<v8::Value> MakeCallback(v8::Isolate* isolate,
                                       v8::Local<v8::Object> recv,
                                       v8::Local<v8::Function> callback,
                                       int argc,
                                       v8::Local<v8::Value>* argv,
                                       async_context asyncContext) {
  return node::MakeCallback(isolate, recv, callback, argc, argv);
}

#endif // NODE_MAJOR_VERSION < 8 || NODE_MAJOR_VERSION == 8 && NODE_MINOR_VERSION < 6

static void PrintErrorString(const char* format, ...) {
  va_list ap;
  va_start(ap, format);
#ifdef _WIN32
  HANDLE stderr_handle = GetStdHandle(STD_ERROR_HANDLE);

  // Check if stderr is something other than a tty/console
  if (stderr_handle == INVALID_HANDLE_VALUE ||
      stderr_handle == nullptr ||
      uv_guess_handle(_fileno(stderr)) != UV_TTY) {
    vfprintf(stderr, format, ap);
    va_end(ap);
    return;
  }

  // Fill in any placeholders
  int n = _vscprintf(format, ap);
  std::vector<char> out(n + 1);
  vsprintf(out.data(), format, ap);

  // Get required wide buffer size
  n = MultiByteToWideChar(CP_UTF8, 0, out.data(), -1, nullptr, 0);

  std::vector<wchar_t> wbuf(n);
  MultiByteToWideChar(CP_UTF8, 0, out.data(), -1, wbuf.data(), n);

  // Don't include the null character in the output
  CHECK_GT(n, 0);
  WriteConsoleW(stderr_handle, wbuf.data(), n - 1, nullptr, nullptr);
#else
  vfprintf(stderr, format, ap);
#endif
  va_end(ap);
}

void DumpBacktrace(FILE* fp) {
}

NO_RETURN void Abort() {
  DumpBacktrace(stderr);
  fflush(stderr);
  ABORT_NO_BACKTRACE();
}

NO_RETURN void Assert(const char* const (*args)[4]) {
  auto filename = (*args)[0];
  auto linenum = (*args)[1];
  auto message = (*args)[2];
  auto function = (*args)[3];

  char exepath[256];
  size_t exepath_size = sizeof(exepath);
  if (uv_exepath(exepath, &exepath_size))
    snprintf(exepath, sizeof(exepath), "node");

  char pid[12] = {0};
  snprintf(pid, sizeof(pid), "[%u]", getpid());

  fprintf(stderr, "%s%s: %s:%s:%s%s Assertion `%s' failed.\n",
          exepath, pid, filename, linenum,
          function, *function ? ":" : "", message);
  fflush(stderr);

  Abort();
}

static void OnFatalError(const char* location, const char* message) {
  if (location) {
    PrintErrorString("FATAL ERROR: %s %s\n", location, message);
  } else {
    PrintErrorString("FATAL ERROR: %s\n", message);
  }
  fflush(stderr);
  ABORT();
}

NO_RETURN void FatalError(const char* location, const char* message) {
  OnFatalError(location, message);
  // to suppress compiler warning
  ABORT();
}

}  // namespace node

#if NODE_MAJOR_VERSION < 6
v8::Local<v8::Name> v8::Private::ForApi(v8::Isolate* isolate,
                                         v8::Local<v8::String> key) {
  return v8::Symbol::ForApi(isolate, key);
}
#endif // NODE_MAJOR_VERSION < 6
