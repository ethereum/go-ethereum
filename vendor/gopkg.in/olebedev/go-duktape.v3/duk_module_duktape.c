/*
 *  Duktape 1.x compatible module loading framework
 */

#include "duktape.h"
#include "duk_module_duktape.h"

/* (v)snprintf() is missing before MSVC 2015.  Note that _(v)snprintf() does
 * NOT NUL terminate on truncation, but that's OK here.
 * http://stackoverflow.com/questions/2915672/snprintf-and-visual-studio-2010
 */
#if defined(_MSC_VER) && (_MSC_VER < 1900)
#define snprintf _snprintf
#endif

#if 0  /* Enable manually */
#define DUK__ASSERT(x) do { \
		if (!(x)) { \
			fprintf(stderr, "ASSERTION FAILED at %s:%d: " #x "\n", __FILE__, __LINE__); \
			fflush(stderr);  \
		} \
	} while (0)
#define DUK__ASSERT_TOP(ctx,val) do { \
		DUK__ASSERT(duk_get_top((ctx)) == (val)); \
	} while (0)
#else
#define DUK__ASSERT(x) do { (void) (x); } while (0)
#define DUK__ASSERT_TOP(ctx,val) do { (void) ctx; (void) (val); } while (0)
#endif

static void duk__resolve_module_id(duk_context *ctx, const char *req_id, const char *mod_id) {
	duk_uint8_t buf[DUK_COMMONJS_MODULE_ID_LIMIT];
	duk_uint8_t *p;
	duk_uint8_t *q;
	duk_uint8_t *q_last;  /* last component */
	duk_int_t int_rc;

	DUK__ASSERT(req_id != NULL);
	/* mod_id may be NULL */

	/*
	 *  A few notes on the algorithm:
	 *
	 *    - Terms are not allowed to begin with a period unless the term
	 *      is either '.' or '..'.  This simplifies implementation (and
	 *      is within CommonJS modules specification).
	 *
	 *    - There are few output bound checks here.  This is on purpose:
	 *      the resolution input is length checked and the output is never
	 *      longer than the input.  The resolved output is written directly
	 *      over the input because it's never longer than the input at any
	 *      point in the algorithm.
	 *
	 *    - Non-ASCII characters are processed as individual bytes and
	 *      need no special treatment.  However, U+0000 terminates the
	 *      algorithm; this is not an issue because U+0000 is not a
	 *      desirable term character anyway.
	 */

	/*
	 *  Set up the resolution input which is the requested ID directly
	 *  (if absolute or no current module path) or with current module
	 *  ID prepended (if relative and current module path exists).
	 *
	 *  Suppose current module is 'foo/bar' and relative path is './quux'.
	 *  The 'bar' component must be replaced so the initial input here is
	 *  'foo/bar/.././quux'.
	 */

	if (mod_id != NULL && req_id[0] == '.') {
		int_rc = snprintf((char *) buf, sizeof(buf), "%s/../%s", mod_id, req_id);
	} else {
		int_rc = snprintf((char *) buf, sizeof(buf), "%s", req_id);
	}
	if (int_rc >= (duk_int_t) sizeof(buf) || int_rc < 0) {
		/* Potentially truncated, NUL not guaranteed in any case.
		 * The (int_rc < 0) case should not occur in practice.
		 */
		goto resolve_error;
	}
	DUK__ASSERT(strlen((const char *) buf) < sizeof(buf));  /* at most sizeof(buf) - 1 */

	/*
	 *  Resolution loop.  At the top of the loop we're expecting a valid
	 *  term: '.', '..', or a non-empty identifier not starting with a period.
	 */

	p = buf;
	q = buf;
	for (;;) {
		duk_uint_fast8_t c;

		/* Here 'p' always points to the start of a term.
		 *
		 * We can also unconditionally reset q_last here: if this is
		 * the last (non-empty) term q_last will have the right value
		 * on loop exit.
		 */

		DUK__ASSERT(p >= q);  /* output is never longer than input during resolution */

		q_last = q;

		c = *p++;
		if (c == 0) {
			goto resolve_error;
		} else if (c == '.') {
			c = *p++;
			if (c == '/') {
				/* Term was '.' and is eaten entirely (including dup slashes). */
				goto eat_dup_slashes;
			}
			if (c == '.' && *p == '/') {
				/* Term was '..', backtrack resolved name by one component.
				 *  q[-1] = previous slash (or beyond start of buffer)
				 *  q[-2] = last char of previous component (or beyond start of buffer)
				 */
				p++;  /* eat (first) input slash */
				DUK__ASSERT(q >= buf);
				if (q == buf) {
					goto resolve_error;
				}
				DUK__ASSERT(*(q - 1) == '/');
				q--;  /* Backtrack to last output slash (dups already eliminated). */
				for (;;) {
					/* Backtrack to previous slash or start of buffer. */
					DUK__ASSERT(q >= buf);
					if (q == buf) {
						break;
					}
					if (*(q - 1) == '/') {
						break;
					}
					q--;
				}
				goto eat_dup_slashes;
			}
			goto resolve_error;
		} else if (c == '/') {
			/* e.g. require('/foo'), empty terms not allowed */
			goto resolve_error;
		} else {
			for (;;) {
				/* Copy term name until end or '/'. */
				*q++ = c;
				c = *p++;
				if (c == 0) {
					/* This was the last term, and q_last was
					 * updated to match this term at loop top.
					 */
					goto loop_done;
				} else if (c == '/') {
					*q++ = '/';
					break;
				} else {
					/* write on next loop */
				}
			}
		}

	 eat_dup_slashes:
		for (;;) {
			/* eat dup slashes */
			c = *p;
			if (c != '/') {
				break;
			}
			p++;
		}
	}
 loop_done:
	/* Output #1: resolved absolute name. */
	DUK__ASSERT(q >= buf);
	duk_push_lstring(ctx, (const char *) buf, (size_t) (q - buf));

	/* Output #2: last component name. */
	DUK__ASSERT(q >= q_last);
	DUK__ASSERT(q_last >= buf);
	duk_push_lstring(ctx, (const char *) q_last, (size_t) (q - q_last));
	return;

 resolve_error:
	(void) duk_type_error(ctx, "cannot resolve module id: %s", (const char *) req_id);
}

/* Stack indices for better readability. */
#define DUK__IDX_REQUESTED_ID   0   /* module id requested */
#define DUK__IDX_REQUIRE        1   /* current require() function */
#define DUK__IDX_REQUIRE_ID     2   /* the base ID of the current require() function, resolution base */
#define DUK__IDX_RESOLVED_ID    3   /* resolved, normalized absolute module ID */
#define DUK__IDX_LASTCOMP       4   /* last component name in resolved path */
#define DUK__IDX_DUKTAPE        5   /* Duktape object */
#define DUK__IDX_MODLOADED      6   /* Duktape.modLoaded[] module cache */
#define DUK__IDX_UNDEFINED      7   /* 'undefined', artifact of lookup */
#define DUK__IDX_FRESH_REQUIRE  8   /* new require() function for module, updated resolution base */
#define DUK__IDX_EXPORTS        9   /* default exports table */
#define DUK__IDX_MODULE         10  /* module object containing module.exports, etc */

static duk_ret_t duk__require(duk_context *ctx) {
	const char *str_req_id;  /* requested identifier */
	const char *str_mod_id;  /* require.id of current module */
	duk_int_t pcall_rc;

	/* NOTE: we try to minimize code size by avoiding unnecessary pops,
	 * so the stack looks a bit cluttered in this function.  DUK__ASSERT_TOP()
	 * assertions are used to ensure stack configuration is correct at each
	 * step.
	 */

	/*
	 *  Resolve module identifier into canonical absolute form.
	 */

	str_req_id = duk_require_string(ctx, DUK__IDX_REQUESTED_ID);
	duk_push_current_function(ctx);
	duk_get_prop_string(ctx, -1, "id");
	str_mod_id = duk_get_string(ctx, DUK__IDX_REQUIRE_ID);  /* ignore non-strings */
	duk__resolve_module_id(ctx, str_req_id, str_mod_id);
	str_req_id = NULL;
	str_mod_id = NULL;

	/* [ requested_id require require.id resolved_id last_comp ] */
	DUK__ASSERT_TOP(ctx, DUK__IDX_LASTCOMP + 1);

	/*
	 *  Cached module check.
	 *
	 *  If module has been loaded or its loading has already begun without
	 *  finishing, return the same cached value (module.exports).  The
	 *  value is registered when module load starts so that circular
	 *  references can be supported to some extent.
	 */

	duk_push_global_stash(ctx);
	duk_get_prop_string(ctx, -1, "\xff" "module:Duktape");
	duk_remove(ctx, -2);  /* Lookup stashed, original 'Duktape' object. */
	duk_get_prop_string(ctx, DUK__IDX_DUKTAPE, "modLoaded");  /* Duktape.modLoaded */
	duk_require_type_mask(ctx, DUK__IDX_MODLOADED, DUK_TYPE_MASK_OBJECT);
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODLOADED + 1);

	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	if (duk_get_prop(ctx, DUK__IDX_MODLOADED)) {
		/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded Duktape.modLoaded[id] ] */
		duk_get_prop_string(ctx, -1, "exports");  /* return module.exports */
		return 1;
	}
	DUK__ASSERT_TOP(ctx, DUK__IDX_UNDEFINED + 1);

	/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded undefined ] */

	/*
	 *  Module not loaded (and loading not started previously).
	 *
	 *  Create a new require() function with 'id' set to resolved ID
	 *  of module being loaded.  Also create 'exports' and 'module'
	 *  tables but don't register exports to the loaded table yet.
	 *  We don't want to do that unless the user module search callbacks
	 *  succeeds in finding the module.
	 */

	/* Fresh require: require.id is left configurable (but not writable)
	 * so that is not easy to accidentally tweak it, but it can still be
	 * done with Object.defineProperty().
	 *
	 * XXX: require.id could also be just made non-configurable, as there
	 * is no practical reason to touch it (at least from Ecmascript code).
	 */
	duk_push_c_function(ctx, duk__require, 1 /*nargs*/);
	duk_push_string(ctx, "name");
	duk_push_string(ctx, "require");
	duk_def_prop(ctx, DUK__IDX_FRESH_REQUIRE, DUK_DEFPROP_HAVE_VALUE);  /* not writable, not enumerable, not configurable */
	duk_push_string(ctx, "id");
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_def_prop(ctx, DUK__IDX_FRESH_REQUIRE, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_CONFIGURABLE);  /* a fresh require() with require.id = resolved target module id */

	/* Module table:
	 * - module.exports: initial exports table (may be replaced by user)
	 * - module.id is non-writable and non-configurable, as the CommonJS
	 *   spec suggests this if possible
	 * - module.filename: not set, defaults to resolved ID if not explicitly
	 *   set by modSearch() (note capitalization, not .fileName, matches Node.js)
	 * - module.name: not set, defaults to last component of resolved ID if
	 *   not explicitly set by modSearch()
	 */
	duk_push_object(ctx);  /* exports */
	duk_push_object(ctx);  /* module */
	duk_push_string(ctx, "exports");
	duk_dup(ctx, DUK__IDX_EXPORTS);
	duk_def_prop(ctx, DUK__IDX_MODULE, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_SET_WRITABLE | DUK_DEFPROP_SET_CONFIGURABLE);  /* module.exports = exports */
	duk_push_string(ctx, "id");
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);  /* resolved id: require(id) must return this same module */
	duk_def_prop(ctx, DUK__IDX_MODULE, DUK_DEFPROP_HAVE_VALUE);  /* module.id = resolved_id; not writable, not enumerable, not configurable */
	duk_compact(ctx, DUK__IDX_MODULE);  /* module table remains registered to modLoaded, minimize its size */
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 1);

	/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded undefined fresh_require exports module ] */

	/* Register the module table early to modLoaded[] so that we can
	 * support circular references even in modSearch().  If an error
	 * is thrown, we'll delete the reference.
	 */
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_dup(ctx, DUK__IDX_MODULE);
	duk_put_prop(ctx, DUK__IDX_MODLOADED);  /* Duktape.modLoaded[resolved_id] = module */

	/*
	 *  Call user provided module search function and build the wrapped
	 *  module source code (if necessary).  The module search function
	 *  can be used to implement pure Ecmacsript, pure C, and mixed
	 *  Ecmascript/C modules.
	 *
	 *  The module search function can operate on the exports table directly
	 *  (e.g. DLL code can register values to it).  It can also return a
	 *  string which is interpreted as module source code (if a non-string
	 *  is returned the module is assumed to be a pure C one).  If a module
	 *  cannot be found, an error must be thrown by the user callback.
	 *
	 *  Because Duktape.modLoaded[] already contains the module being
	 *  loaded, circular references for C modules should also work
	 *  (although expected to be quite rare).
	 */

	duk_push_string(ctx, "(function(require,exports,module){");

	/* Duktape.modSearch(resolved_id, fresh_require, exports, module). */
	duk_get_prop_string(ctx, DUK__IDX_DUKTAPE, "modSearch");  /* Duktape.modSearch */
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_dup(ctx, DUK__IDX_FRESH_REQUIRE);
	duk_dup(ctx, DUK__IDX_EXPORTS);
	duk_dup(ctx, DUK__IDX_MODULE);  /* [ ... Duktape.modSearch resolved_id last_comp fresh_require exports module ] */
	pcall_rc = duk_pcall(ctx, 4 /*nargs*/);  /* -> [ ... source ] */
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 3);

	if (pcall_rc != DUK_EXEC_SUCCESS) {
		/* Delete entry in Duktape.modLoaded[] and rethrow. */
		goto delete_rethrow;
	}

	/* If user callback did not return source code, module loading
	 * is finished (user callback initialized exports table directly).
	 */
	if (!duk_is_string(ctx, -1)) {
		/* User callback did not return source code, so module loading
		 * is finished: just update modLoaded with final module.exports
		 * and we're done.
		 */
		goto return_exports;
	}

	/* Finish the wrapped module source.  Force module.filename as the
	 * function .fileName so it gets set for functions defined within a
	 * module.  This also ensures loggers created within the module get
	 * the module ID (or overridden filename) as their default logger name.
	 * (Note capitalization: .filename matches Node.js while .fileName is
	 * used elsewhere in Duktape.)
	 */
	duk_push_string(ctx, "\n})");  /* Newline allows module last line to contain a // comment. */
	duk_concat(ctx, 3);
	if (!duk_get_prop_string(ctx, DUK__IDX_MODULE, "filename")) {
		/* module.filename for .fileName, default to resolved ID if
		 * not present.
		 */
		duk_pop(ctx);
		duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	}
	pcall_rc = duk_pcompile(ctx, DUK_COMPILE_EVAL);
	if (pcall_rc != DUK_EXEC_SUCCESS) {
		goto delete_rethrow;
	}
	pcall_rc = duk_pcall(ctx, 0);  /* -> eval'd function wrapper (not called yet) */
	if (pcall_rc != DUK_EXEC_SUCCESS) {
		goto delete_rethrow;
	}

	/* Module has now evaluated to a wrapped module function.  Force its
	 * .name to match module.name (defaults to last component of resolved
	 * ID) so that it is shown in stack traces too.  Note that we must not
	 * introduce an actual name binding into the function scope (which is
	 * usually the case with a named function) because it would affect the
	 * scope seen by the module and shadow accesses to globals of the same name.
	 * This is now done by compiling the function as anonymous and then forcing
	 * its .name without setting a "has name binding" flag.
	 */

	duk_push_string(ctx, "name");
	if (!duk_get_prop_string(ctx, DUK__IDX_MODULE, "name")) {
		/* module.name for .name, default to last component if
		 * not present.
		 */
		duk_pop(ctx);
		duk_dup(ctx, DUK__IDX_LASTCOMP);
	}
	duk_def_prop(ctx, -3, DUK_DEFPROP_HAVE_VALUE | DUK_DEFPROP_FORCE);

	/*
	 *  Call the wrapped module function.
	 *
	 *  Use a protected call so that we can update Duktape.modLoaded[resolved_id]
	 *  even if the module throws an error.
	 */

	/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded undefined fresh_require exports module mod_func ] */
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 2);

	duk_dup(ctx, DUK__IDX_EXPORTS);  /* exports (this binding) */
	duk_dup(ctx, DUK__IDX_FRESH_REQUIRE);  /* fresh require (argument) */
	duk_get_prop_string(ctx, DUK__IDX_MODULE, "exports");  /* relookup exports from module.exports in case it was changed by modSearch */
	duk_dup(ctx, DUK__IDX_MODULE);  /* module (argument) */
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 6);

	/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded undefined fresh_require exports module mod_func exports fresh_require exports module ] */

	pcall_rc = duk_pcall_method(ctx, 3 /*nargs*/);
	if (pcall_rc != DUK_EXEC_SUCCESS) {
		/* Module loading failed.  Node.js will forget the module
		 * registration so that another require() will try to load
		 * the module again.  Mimic that behavior.
		 */
		goto delete_rethrow;
	}

	/* [ requested_id require require.id resolved_id last_comp Duktape Duktape.modLoaded undefined fresh_require exports module result(ignored) ] */
	DUK__ASSERT_TOP(ctx, DUK__IDX_MODULE + 2);

	/* fall through */

 return_exports:
	duk_get_prop_string(ctx, DUK__IDX_MODULE, "exports");
	duk_compact(ctx, -1);  /* compact the exports table */
	return 1;  /* return module.exports */

 delete_rethrow:
	duk_dup(ctx, DUK__IDX_RESOLVED_ID);
	duk_del_prop(ctx, DUK__IDX_MODLOADED);  /* delete Duktape.modLoaded[resolved_id] */
	(void) duk_throw(ctx);  /* rethrow original error */
	return 0;  /* not reachable */
}

void duk_module_duktape_init(duk_context *ctx) {
	/* Stash 'Duktape' in case it's modified. */
	duk_push_global_stash(ctx);
	duk_get_global_string(ctx, "Duktape");
	duk_put_prop_string(ctx, -2, "\xff" "module:Duktape");
	duk_pop(ctx);

	/* Register `require` as a global function. */
	duk_eval_string(ctx,
		"(function(req){"
		"var D=Object.defineProperty;"
		"D(req,'name',{value:'require'});"
		"D(this,'require',{value:req,writable:true,configurable:true});"
		"D(Duktape,'modLoaded',{value:Object.create(null),writable:true,configurable:true});"
		"})");
	duk_push_c_function(ctx, duk__require, 1 /*nargs*/);
	duk_call(ctx, 1);
	duk_pop(ctx);
}

#undef DUK__ASSERT
#undef DUK__ASSERT_TOP
#undef DUK__IDX_REQUESTED_ID
#undef DUK__IDX_REQUIRE
#undef DUK__IDX_REQUIRE_ID
#undef DUK__IDX_RESOLVED_ID
#undef DUK__IDX_LASTCOMP
#undef DUK__IDX_DUKTAPE
#undef DUK__IDX_MODLOADED
#undef DUK__IDX_UNDEFINED
#undef DUK__IDX_FRESH_REQUIRE
#undef DUK__IDX_EXPORTS
#undef DUK__IDX_MODULE
