#if !defined(DUK_CONSOLE_H_INCLUDED)
#define DUK_CONSOLE_H_INCLUDED

#include "duktape.h"

/* Use a proxy wrapper to make undefined methods (console.foo()) no-ops. */
#define DUK_CONSOLE_PROXY_WRAPPER  (1 << 0)

/* Flush output after every call. */
#define DUK_CONSOLE_FLUSH          (1 << 1)

extern void duk_console_init(duk_context *ctx, duk_uint_t flags);

#endif  /* DUK_CONSOLE_H_INCLUDED */
