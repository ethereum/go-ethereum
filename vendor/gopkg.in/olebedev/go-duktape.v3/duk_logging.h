#if !defined(DUK_LOGGING_H_INCLUDED)
#define DUK_LOGGING_H_INCLUDED

#include "duktape.h"

#if defined(__cplusplus)
extern "C" {
#endif

/* Log levels. */
#define DUK_LOG_TRACE                     0
#define DUK_LOG_DEBUG                     1
#define DUK_LOG_INFO                      2
#define DUK_LOG_WARN                      3
#define DUK_LOG_ERROR                     4
#define DUK_LOG_FATAL                     5

/* No flags at the moment. */

extern void duk_logging_init(duk_context *ctx, duk_uint_t flags);
extern void duk_log_va(duk_context *ctx, duk_int_t level, const char *fmt, va_list ap);
extern void duk_log(duk_context *ctx, duk_int_t level, const char *fmt, ...);

#if defined(__cplusplus)
}
#endif  /* end 'extern "C"' wrapper */

#endif  /* DUK_LOGGING_H_INCLUDED */
