/*
 *  Logging support
 */

#include <stdio.h>
#include <string.h>
#include <stdarg.h>
#include "duktape.h"
#include "duk_logging.h"

/* XXX: uses stderr always for now, configurable? */

#define DUK_LOGGING_FLUSH  /* Duktape 1.x: flush stderr */

/* 3-letter log level strings. */
static const char duk__log_level_strings[] = {
	'T', 'R', 'C', 'D', 'B', 'G', 'I', 'N', 'F',
	'W', 'R', 'N', 'E', 'R', 'R', 'F', 'T', 'L'
};

/* Log method names. */
static const char *duk__log_method_names[] = {
	"trace", "debug", "info", "warn", "error", "fatal"
};

/* Constructor. */
static duk_ret_t duk__logger_constructor(duk_context *ctx) {
	duk_idx_t nargs;

	/* Calling as a non-constructor is not meaningful. */
	if (!duk_is_constructor_call(ctx)) {
		return DUK_RET_TYPE_ERROR;
	}

	nargs = duk_get_top(ctx);
	duk_set_top(ctx, 1);

	duk_push_this(ctx);

	/* [ name this ] */

	if (nargs == 0) {
		/* Automatic defaulting of logger name from caller.  This
		 * would work poorly with tail calls, but constructor calls
		 * are currently never tail calls, so tail calls are not an
		 * issue now.
		 */

		duk_inspect_callstack_entry(ctx, -2);
		if (duk_is_object(ctx, -1)) {
			if (duk_get_prop_string(ctx, -1, "function")) {
				if (duk_get_prop_string(ctx, -1, "fileName")) {
					if (duk_is_string(ctx, -1)) {
						duk_replace(ctx, 0);
					}
				}
			}
		}
		/* Leave values on stack on purpose, ignored below. */

		/* Stripping the filename might be a good idea
		 * ("/foo/bar/quux.js" -> logger name "quux"),
		 * but now used verbatim.
		 */
	}
	/* The stack is unbalanced here on purpose; we only rely on the
	 * initial two values: [ name this ].
	 */

	if (duk_is_string(ctx, 0)) {
		duk_dup(ctx, 0);
		duk_put_prop_string(ctx, 1, "n");
	} else {
		/* don't set 'n' at all, inherited value is used as name */
	}

	duk_compact(ctx, 1);

	return 0;  /* keep default instance */
}

/* Default function to format objects.  Tries to use toLogString() but falls
 * back to toString().  Any errors are propagated out without catching.
 */
static duk_ret_t duk__logger_prototype_fmt(duk_context *ctx) {
	if (duk_get_prop_string(ctx, 0, "toLogString")) {
		/* [ arg toLogString ] */

		duk_dup(ctx, 0);
		duk_call_method(ctx, 0);

		/* [ arg result ] */
		return 1;
	}

	/* [ arg undefined ] */
	duk_pop(ctx);
	duk_to_string(ctx, 0);
	return 1;
}

/* Default function to write a formatted log line.  Writes to stderr,
 * appending a newline to the log line.
 *
 * The argument is a buffer; avoid coercing the buffer to a string to
 * avoid string table traffic.
 */
static duk_ret_t duk__logger_prototype_raw(duk_context *ctx) {
	const char *data;
	duk_size_t data_len;

	data = (const char *) duk_require_buffer(ctx, 0, &data_len);
	fwrite((const void *) data, 1, data_len, stderr);
	fputc((int) '\n', stderr);
#if defined(DUK_LOGGING_FLUSH)
	fflush(stderr);
#endif
	return 0;
}

/* Log frontend shared helper, magic value indicates log level.  Provides
 * frontend functions: trace(), debug(), info(), warn(), error(), fatal().
 * This needs to have small footprint, reasonable performance, minimal
 * memory churn, etc.
 */
static duk_ret_t duk__logger_prototype_log_shared(duk_context *ctx) {
	duk_double_t now;
	duk_time_components comp;
	duk_small_int_t entry_lev;
	duk_small_int_t logger_lev;
	duk_int_t nargs;
	duk_int_t i;
	duk_size_t tot_len;
	const duk_uint8_t *arg_str;
	duk_size_t arg_len;
	duk_uint8_t *buf, *p;
	const duk_uint8_t *q;
	duk_uint8_t date_buf[32];  /* maximum format length is 24+1 (NUL), round up. */
	duk_size_t date_len;
	duk_small_int_t rc;

	/* XXX: sanitize to printable (and maybe ASCII) */
	/* XXX: better multiline */

	/*
	 *  Logger arguments are:
	 *
	 *    magic: log level (0-5)
	 *    this: logger
	 *    stack: plain log args
	 *
	 *  We want to minimize memory churn so a two-pass approach
	 *  is used: first pass formats arguments and computes final
	 *  string length, second pass copies strings into a buffer
	 *  allocated directly with the correct size.  If the backend
	 *  function plays nice, it won't coerce the buffer to a string
	 *  (and thus intern it).
	 */

	entry_lev = duk_get_current_magic(ctx);
	if (entry_lev < DUK_LOG_TRACE || entry_lev > DUK_LOG_FATAL) {
		/* Should never happen, check just in case. */
		return 0;
	}
	nargs = duk_get_top(ctx);

	/* [ arg1 ... argN this ] */

	/*
	 *  Log level check
	 */

	duk_push_this(ctx);

	duk_get_prop_string(ctx, -1, "l");
	logger_lev = (duk_small_int_t) duk_get_int(ctx, -1);
	if (entry_lev < logger_lev) {
		return 0;
	}
	/* log level could be popped but that's not necessary */

	now = duk_get_now(ctx);
	duk_time_to_components(ctx, now, &comp);
	sprintf((char *) date_buf, "%04d-%02d-%02dT%02d:%02d:%02d.%03dZ",
	        (int) comp.year, (int) comp.month + 1, (int) comp.day,
	        (int) comp.hours, (int) comp.minutes, (int) comp.seconds,
	        (int) comp.milliseconds);

	date_len = strlen((const char *) date_buf);

	duk_get_prop_string(ctx, -2, "n");
	duk_to_string(ctx, -1);

	/* [ arg1 ... argN this loggerLevel loggerName ] */

	/*
	 *  Pass 1
	 */

	/* Line format: <time> <entryLev> <loggerName>: <msg> */

	tot_len = 0;
	tot_len += 3 +  /* separators: space, space, colon */
	           3 +  /* level string */
	           date_len +  /* time */
	           duk_get_length(ctx, -1);  /* loggerName */

	for (i = 0; i < nargs; i++) {
		/* When formatting an argument to a string, errors may happen from multiple
		 * causes.  In general we want to catch obvious errors like a toLogString()
		 * throwing an error, but we don't currently try to catch every possible
		 * error.  In particular, internal errors (like out of memory or stack) are
		 * not caught.  Also, we expect Error toString() to not throw an error.
		 */
		if (duk_is_object(ctx, i)) {
			/* duk_pcall_prop() may itself throw an error, but we're content
			 * in catching the obvious errors (like toLogString() throwing an
			 * error).
			 */
			duk_push_string(ctx, "fmt");
			duk_dup(ctx, i);
			/* [ arg1 ... argN this loggerLevel loggerName 'fmt' arg ] */
			/* call: this.fmt(arg) */
			rc = duk_pcall_prop(ctx, -5 /*obj_index*/, 1 /*nargs*/);
			if (rc) {
				/* Keep the error as the result (coercing it might fail below,
				 * but we don't catch that now).
				 */
				;
			}
			duk_replace(ctx, i);
		}
		(void) duk_to_lstring(ctx, i, &arg_len);
		tot_len++;  /* sep (even before first one) */
		tot_len += arg_len;
	}

	/*
	 *  Pass 2
	 */

	/* XXX: Here it'd be nice if we didn't need to allocate a new fixed
	 * buffer for every write.  This would be possible if raw() took a
	 * buffer and a length.  We could then use a preallocated buffer for
	 * most log writes and request raw() to write a partial buffer.
	 */

	buf = (duk_uint8_t *) duk_push_fixed_buffer(ctx, tot_len);
	p = buf;

	memcpy((void *) p, (const void *) date_buf, (size_t) date_len);
	p += date_len;
	*p++ = (duk_uint8_t) ' ';

	q = (const duk_uint8_t *) duk__log_level_strings + (entry_lev * 3);
	memcpy((void *) p, (const void *) q, (size_t) 3);
	p += 3;

	*p++ = (duk_uint8_t) ' ';

	arg_str = (const duk_uint8_t *) duk_get_lstring(ctx, -2, &arg_len);
	memcpy((void *) p, (const void *) arg_str, (size_t) arg_len);
	p += arg_len;

	*p++ = (duk_uint8_t) ':';

	for (i = 0; i < nargs; i++) {
		*p++ = (duk_uint8_t) ' ';

		arg_str = (const duk_uint8_t *) duk_get_lstring(ctx, i, &arg_len);
		memcpy((void *) p, (const void *) arg_str, (size_t) arg_len);
		p += arg_len;
	}

	/* [ arg1 ... argN this loggerLevel loggerName buffer ] */

	/* Call this.raw(msg); look up through the instance allows user to override
	 * the raw() function in the instance or in the prototype for maximum
	 * flexibility.
	 */
	duk_push_string(ctx, "raw");
	duk_dup(ctx, -2);
	/* [ arg1 ... argN this loggerLevel loggerName buffer 'raw' buffer ] */
	duk_call_prop(ctx, -6, 1);  /* this.raw(buffer) */

	return 0;
}

void duk_log_va(duk_context *ctx, duk_int_t level, const char *fmt, va_list ap) {
	if (level < 0) {
		level = 0;
	} else if (level > (int) (sizeof(duk__log_method_names) / sizeof(const char *)) - 1) {
		level = (int) (sizeof(duk__log_method_names) / sizeof(const char *)) - 1;
	}

	duk_push_global_stash(ctx);
	duk_get_prop_string(ctx, -1, "\xff" "logger:constructor");  /* fixed at init time */
	duk_get_prop_string(ctx, -1, "clog");
	duk_get_prop_string(ctx, -1, duk__log_method_names[level]);
	duk_dup(ctx, -2);
	duk_push_vsprintf(ctx, fmt, ap);

	/* [ ... stash Logger clog logfunc clog(=this) msg ] */

	duk_call_method(ctx, 1 /*nargs*/);

	/* [ ... stash Logger clog res ] */

	duk_pop_n(ctx, 4);
}

void duk_log(duk_context *ctx, duk_int_t level, const char *fmt, ...) {
	va_list ap;

	va_start(ap, fmt);
	duk_log_va(ctx, level, fmt, ap);
	va_end(ap);
}

void duk_logging_init(duk_context *ctx, duk_uint_t flags) {
	/* XXX: Add .name property for logger functions (useful for stack traces if they throw). */

	(void) flags;

	duk_eval_string(ctx,
		"(function(cons,prot){"
		"Object.defineProperty(Duktape,'Logger',{value:cons,writable:true,configurable:true});"
		"Object.defineProperty(cons,'prototype',{value:prot});"
		"Object.defineProperty(cons,'clog',{value:new Duktape.Logger('C'),writable:true,configurable:true});"
		"});");

	duk_push_c_function(ctx, duk__logger_constructor, DUK_VARARGS /*nargs*/);  /* Duktape.Logger */
	duk_push_object(ctx);  /* Duktape.Logger.prototype */

	/* [ ... func Duktape.Logger Duktape.Logger.prototype ] */

	duk_push_string(ctx, "name");
	duk_push_string(ctx, "Logger");
	duk_def_prop(ctx, -4, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);

	duk_dup_top(ctx);
	duk_put_prop_string(ctx, -2, "constructor");
	duk_push_int(ctx, 2);
	duk_put_prop_string(ctx, -2, "l");
	duk_push_string(ctx, "anon");
	duk_put_prop_string(ctx, -2, "n");
	duk_push_c_function(ctx, duk__logger_prototype_fmt, 1 /*nargs*/);
	duk_put_prop_string(ctx, -2, "fmt");
	duk_push_c_function(ctx, duk__logger_prototype_raw, 1 /*nargs*/);
	duk_put_prop_string(ctx, -2, "raw");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 0);  /* magic=0: trace */
	duk_put_prop_string(ctx, -2, "trace");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 1);  /* magic=1: debug */
	duk_put_prop_string(ctx, -2, "debug");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 2);  /* magic=2: info */
	duk_put_prop_string(ctx, -2, "info");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 3);  /* magic=3: warn */
	duk_put_prop_string(ctx, -2, "warn");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 4);  /* magic=4: error */
	duk_put_prop_string(ctx, -2, "error");
	duk_push_c_function(ctx, duk__logger_prototype_log_shared, DUK_VARARGS /*nargs*/);
	duk_set_magic(ctx, -1, 5);  /* magic=5: fatal */
	duk_put_prop_string(ctx, -2, "fatal");

	/* [ ... func Duktape.Logger Duktape.Logger.prototype ] */

	/* XXX: when using ROM built-ins, "Duktape" is read-only by default so
	 * setting Duktape.Logger will now fail.
	 */

	/* [ ... func Duktape.Logger Duktape.Logger.prototype ] */

	duk_call(ctx, 2);
	duk_pop(ctx);
}
