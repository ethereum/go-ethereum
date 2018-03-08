#include <stdio.h>
#include "duktape.h"
#include "duk_v1_compat.h"

/*
 *  duk_dump_context_{stdout,stderr}()
 */

void duk_dump_context_stdout(duk_context *ctx) {
	duk_push_context_dump(ctx);
	fprintf(stdout, "%s\n", duk_safe_to_string(ctx, -1));
	duk_pop(ctx);
}

void duk_dump_context_stderr(duk_context *ctx) {
	duk_push_context_dump(ctx);
	fprintf(stderr, "%s\n", duk_safe_to_string(ctx, -1));
	duk_pop(ctx);
}

/*
 *  duk_push_string_file() and duk_push_string_file_raw()
 */

const char *duk_push_string_file_raw(duk_context *ctx, const char *path, duk_uint_t flags) {
	FILE *f = NULL;
	char *buf;
	long sz;  /* ANSI C typing */

	if (!path) {
		goto fail;
	}
	f = fopen(path, "rb");
	if (!f) {
		goto fail;
	}
	if (fseek(f, 0, SEEK_END) < 0) {
		goto fail;
	}
	sz = ftell(f);
	if (sz < 0) {
		goto fail;
	}
	if (fseek(f, 0, SEEK_SET) < 0) {
		goto fail;
	}
	buf = (char *) duk_push_fixed_buffer(ctx, (duk_size_t) sz);
	if ((size_t) fread(buf, 1, (size_t) sz, f) != (size_t) sz) {
		duk_pop(ctx);
		goto fail;
	}
	(void) fclose(f);  /* ignore fclose() error */
	return duk_buffer_to_string(ctx, -1);

 fail:
	if (f) {
		(void) fclose(f);  /* ignore fclose() error */
	}

	if (flags & DUK_STRING_PUSH_SAFE) {
		duk_push_undefined(ctx);
	} else {
		(void) duk_type_error(ctx, "read file error");
	}
	return NULL;
}

/*
 *  duk_eval_file(), duk_compile_file(), and their variants
 */

void duk_eval_file(duk_context *ctx, const char *path) {
	duk_push_string_file_raw(ctx, path, 0);
	duk_push_string(ctx, path);
	duk_compile(ctx, DUK_COMPILE_EVAL);
	duk_push_global_object(ctx);  /* 'this' binding */
	duk_call_method(ctx, 0);
}

void duk_eval_file_noresult(duk_context *ctx, const char *path) {
	duk_eval_file(ctx, path);
	duk_pop(ctx);
}

duk_int_t duk_peval_file(duk_context *ctx, const char *path) {
	duk_int_t rc;

	duk_push_string_file_raw(ctx, path, DUK_STRING_PUSH_SAFE);
	duk_push_string(ctx, path);
	rc = duk_pcompile(ctx, DUK_COMPILE_EVAL);
	if (rc != 0) {
		return rc;
	}
	duk_push_global_object(ctx);  /* 'this' binding */
	rc = duk_pcall_method(ctx, 0);
	return rc;
}

duk_int_t duk_peval_file_noresult(duk_context *ctx, const char *path) {
	duk_int_t rc;

	rc = duk_peval_file(ctx, path);
	duk_pop(ctx);
	return rc;
}

void duk_compile_file(duk_context *ctx, duk_uint_t flags, const char *path) {
	duk_push_string_file_raw(ctx, path, 0);
	duk_push_string(ctx, path);
	duk_compile(ctx, flags);
}

duk_int_t duk_pcompile_file(duk_context *ctx, duk_uint_t flags, const char *path) {
	duk_int_t rc;

	duk_push_string_file_raw(ctx, path, DUK_STRING_PUSH_SAFE);
	duk_push_string(ctx, path);
	rc = duk_pcompile(ctx, flags);
	return rc;
}

/*
 *  duk_to_defaultvalue()
 */

void duk_to_defaultvalue(duk_context *ctx, duk_idx_t idx, duk_int_t hint) {
	duk_require_type_mask(ctx, idx, DUK_TYPE_MASK_OBJECT |
	                                DUK_TYPE_MASK_BUFFER |
	                                DUK_TYPE_MASK_LIGHTFUNC);
	duk_to_primitive(ctx, idx, hint);
}
