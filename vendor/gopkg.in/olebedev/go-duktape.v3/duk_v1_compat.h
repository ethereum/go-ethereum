#if !defined(DUK_V1_COMPAT_INCLUDED)
#define DUK_V1_COMPAT_INCLUDED

#include "duktape.h"

/* Straight flag rename */
#if !defined(DUK_ENUM_INCLUDE_INTERNAL)
#define DUK_ENUM_INCLUDE_INTERNAL DUK_ENUM_INCLUDE_HIDDEN
#endif

/* Flags for duk_push_string_file_raw() */
#define DUK_STRING_PUSH_SAFE              (1 << 0)    /* no error if file does not exist */

extern void duk_dump_context_stdout(duk_context *ctx);
extern void duk_dump_context_stderr(duk_context *ctx);
extern const char *duk_push_string_file_raw(duk_context *ctx, const char *path, duk_uint_t flags);
extern void duk_eval_file(duk_context *ctx, const char *path);
extern void duk_eval_file_noresult(duk_context *ctx, const char *path);
extern duk_int_t duk_peval_file(duk_context *ctx, const char *path);
extern duk_int_t duk_peval_file_noresult(duk_context *ctx, const char *path);
extern void duk_compile_file(duk_context *ctx, duk_uint_t flags, const char *path);
extern duk_int_t duk_pcompile_file(duk_context *ctx, duk_uint_t flags, const char *path);
extern void duk_to_defaultvalue(duk_context *ctx, duk_idx_t idx, duk_int_t hint);

#define duk_push_string_file(ctx,path) \
	duk_push_string_file_raw((ctx), (path), 0)

#endif  /* DUK_V1_COMPAT_INCLUDED */
