#if !defined(DUK_PRINT_ALERT_H_INCLUDED)
#define DUK_PRINT_ALERT_H_INCLUDED

#include "duktape.h"

#if defined(__cplusplus)
extern "C" {
#endif

/* No flags at the moment. */

extern void duk_print_alert_init(duk_context *ctx, duk_uint_t flags);

#if defined(__cplusplus)
}
#endif  /* end 'extern "C"' wrapper */

#endif /* DUK_PRINT_ALERT_H_INCLUDED */
