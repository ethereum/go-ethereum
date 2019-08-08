#if !defined(DUK_CONSOLE_H_INCLUDED)
#define DUK_CONSOLE_H_INCLUDED

#include "duktape.h"

#if defined(__cplusplus)
extern "C" {
#endif

/* Use a proxy wrapper to make undefined methods (console.foo()) no-ops. */
#define DUK_CONSOLE_PROXY_WRAPPER  (1U << 0)

/* Flush output after every call. */
#define DUK_CONSOLE_FLUSH          (1U << 1)

/* Send output to stdout only (default is mixed stdout/stderr). */
#define DUK_CONSOLE_STDOUT_ONLY    (1U << 2)

/* Send output to stderr only (default is mixed stdout/stderr). */
#define DUK_CONSOLE_STDERR_ONLY    (1U << 3)

/* Initialize the console system */
extern void duk_console_init(duk_context *ctx, duk_uint_t flags);

#if defined(__cplusplus)
}
#endif  /* end 'extern "C"' wrapper */

#endif  /* DUK_CONSOLE_H_INCLUDED */
