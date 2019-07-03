#if !defined(DUK_ALLOC_POOL_H_INCLUDED)
#define DUK_ALLOC_POOL_H_INCLUDED

#include "duktape.h"

#if defined(__cplusplus)
extern "C" {
#endif

/* 32-bit (big endian) marker used at the end of pool entries so that wasted
 * space can be detected.  Waste tracking must be enabled explicitly.
 */
#if defined(DUK_ALLOC_POOL_TRACK_WASTE)
#define DUK_ALLOC_POOL_WASTE_MARKER  0xedcb2345UL
#endif

/* Pointer compression with ROM strings/objects:
 *
 * For now, use DUK_USE_ROM_OBJECTS to signal the need for compressed ROM
 * pointers.  DUK_USE_ROM_PTRCOMP_FIRST is provided for the ROM pointer
 * compression range minimum to avoid duplication in user code.
 */
#if defined(DUK_USE_ROM_OBJECTS) && defined(DUK_USE_HEAPPTR16)
#define DUK_ALLOC_POOL_ROMPTR_COMPRESSION
#define DUK_ALLOC_POOL_ROMPTR_FIRST DUK_USE_ROM_PTRCOMP_FIRST

/* This extern declaration is provided by duktape.h, array provided by duktape.c.
 * Because duk_config.h may include this file (to get the inline functions) we
 * need to forward declare this also here.
 */
extern const void * const duk_rom_compressed_pointers[];
#endif

/* Pool configuration for a certain block size. */
typedef struct {
	unsigned int size;  /* must be divisible by 4 and >= sizeof(void *) */
	unsigned int a;     /* bytes (not count) to allocate: a*t + b, t is an arbitrary scale parameter */
	unsigned int b;
} duk_pool_config;

/* Freelist entry, must fit into the smallest block size. */
struct duk_pool_free;
typedef struct duk_pool_free duk_pool_free;
struct duk_pool_free {
	duk_pool_free *next;
};

/* Pool state for a certain block size. */
typedef struct {
	duk_pool_free *first;
	char *alloc_end;
	unsigned int size;
	unsigned int count;
#if defined(DUK_ALLOC_POOL_TRACK_HIGHWATER)
	unsigned int hwm_used_count;
#endif
} duk_pool_state;

/* Statistics for a certain pool. */
typedef struct {
	size_t used_count;
	size_t used_bytes;
	size_t free_count;
	size_t free_bytes;
	size_t waste_bytes;
	size_t hwm_used_count;
} duk_pool_stats;

/* Top level state for all pools.  Pointer to this struct is used as the allocator
 * userdata pointer.
 */
typedef struct {
	int num_pools;
	duk_pool_state *states;
#if defined(DUK_ALLOC_POOL_TRACK_HIGHWATER)
	size_t hwm_used_bytes;
	size_t hwm_waste_bytes;
#endif
} duk_pool_global;

/* Statistics for the entire set of pools. */
typedef struct {
	size_t used_bytes;
	size_t free_bytes;
	size_t waste_bytes;
	size_t hwm_used_bytes;
	size_t hwm_waste_bytes;
} duk_pool_global_stats;

/* Initialize a pool allocator, arguments:
 *   - buffer and size: continuous region to use for pool, must align to 4
 *   - config: configuration for pools in ascending block size
 *   - state: state for pools, matches config order
 *   - num_pools: number of entries in 'config' and 'state'
 *   - global: global state structure
 *
 * The 'config', 'state', and 'global' pointers must be valid beyond the init
 * call, as long as the pool is used.
 *
 * Returns a void pointer to be used as userdata for the allocator functions.
 * Concretely the return value will be "(void *) global", i.e. the global
 * state struct.  If pool init fails, the return value will be NULL.
 */
void *duk_alloc_pool_init(char *buffer,
                          size_t size,
                          const duk_pool_config *configs,
                          duk_pool_state *states,
                          int num_pools,
                          duk_pool_global *global);

/* Duktape allocation providers.  Typing matches Duktape requirements. */
void *duk_alloc_pool(void *udata, duk_size_t size);
void *duk_realloc_pool(void *udata, void *ptr, duk_size_t size);
void duk_free_pool(void *udata, void *ptr);

/* Stats. */
void duk_alloc_pool_get_pool_stats(duk_pool_state *s, duk_pool_stats *res);
void duk_alloc_pool_get_global_stats(duk_pool_global *g, duk_pool_global_stats *res);

/* Duktape pointer compression global state (assumes single pool). */
#if defined(DUK_USE_ROM_OBJECTS) && defined(DUK_USE_HEAPPTR16)
extern const void *duk_alloc_pool_romptr_low;
extern const void *duk_alloc_pool_romptr_high;
duk_uint16_t duk_alloc_pool_enc16_rom(void *ptr);
#endif
#if defined(DUK_USE_HEAPPTR16)
extern void *duk_alloc_pool_ptrcomp_base;
#endif

#if 0
duk_uint16_t duk_alloc_pool_enc16(void *ptr);
void *duk_alloc_pool_dec16(duk_uint16_t val);
#endif

/* Inlined pointer compression functions.  Gcc and clang -Os won't in
 * practice inline these without an "always inline" attribute because it's
 * more size efficient (by a few kB) to use explicit calls instead.  Having
 * these defined inline here allows performance optimized builds to inline
 * pointer compression operations.
 *
 * Pointer compression assumes there's a single globally registered memory
 * pool which makes pointer compression more efficient.  This would be easy
 * to fix by adding a userdata pointer to the compression functions and
 * plumbing the heap userdata from the compression/decompression macros.
 */

/* DUK_ALWAYS_INLINE is not a public API symbol so it may go away in even a
 * minor update.  But it's pragmatic for this extra because it handles many
 * compilers via duk_config.h detection.  Check that the macro exists so that
 * if it's gone, we can still compile.
 */
#if defined(DUK_ALWAYS_INLINE)
#define DUK__ALLOC_POOL_ALWAYS_INLINE DUK_ALWAYS_INLINE
#else
#define DUK__ALLOC_POOL_ALWAYS_INLINE /* nop */
#endif

#if defined(DUK_USE_HEAPPTR16)
static DUK__ALLOC_POOL_ALWAYS_INLINE duk_uint16_t duk_alloc_pool_enc16(void *ptr) {
	if (ptr == NULL) {
		/* With 'return 0' gcc and clang -Os generate inefficient code.
		 * For example, gcc -Os generates:
		 *
		 *   0804911d <duk_alloc_pool_enc16>:
		 *    804911d:       55                      push   %ebp
		 *    804911e:       85 c0                   test   %eax,%eax
		 *    8049120:       89 e5                   mov    %esp,%ebp
		 *    8049122:       74 0b                   je     804912f <duk_alloc_pool_enc16+0x12>
		 *    8049124:       2b 05 e4 90 07 08       sub    0x80790e4,%eax
		 *    804912a:       c1 e8 02                shr    $0x2,%eax
		 *    804912d:       eb 02                   jmp    8049131 <duk_alloc_pool_enc16+0x14>
		 *    804912f:       31 c0                   xor    %eax,%eax
		 *    8049131:       5d                      pop    %ebp
		 *    8049132:       c3                      ret
		 *
		 * The NULL path checks %eax for zero; if it is zero, a zero
		 * is unnecessarily loaded into %eax again.  The non-zero path
		 * has an unnecessary jump as a side effect of this.
		 *
		 * Using 'return (duk_uint16_t) (intptr_t) ptr;' generates similarly
		 * inefficient code; not sure how to make the result better.
		 */
		return 0;
	}
#if defined(DUK_ALLOC_POOL_ROMPTR_COMPRESSION)
	if (ptr >= duk_alloc_pool_romptr_low && ptr <= duk_alloc_pool_romptr_high) {
		/* This is complex enough now to need a separate function. */
		return duk_alloc_pool_enc16_rom(ptr);
	}
#endif
	return (duk_uint16_t) (((size_t) ((char *) ptr - (char *) duk_alloc_pool_ptrcomp_base)) >> 2);
}

static DUK__ALLOC_POOL_ALWAYS_INLINE void *duk_alloc_pool_dec16(duk_uint16_t val) {
	if (val == 0) {
		/* As with enc16 the gcc and clang -Os output is inefficient,
		 * e.g. gcc -Os:
		 *
		 *   08049133 <duk_alloc_pool_dec16>:
		 *    8049133:       55                      push   %ebp
		 *    8049134:       66 85 c0                test   %ax,%ax
		 *    8049137:       89 e5                   mov    %esp,%ebp
		 *    8049139:       74 0e                   je     8049149 <duk_alloc_pool_dec16+0x16>
		 *    804913b:       8b 15 e4 90 07 08       mov    0x80790e4,%edx
		 *    8049141:       0f b7 c0                movzwl %ax,%eax
		 *    8049144:       8d 04 82                lea    (%edx,%eax,4),%eax
		 *    8049147:       eb 02                   jmp    804914b <duk_alloc_pool_dec16+0x18>
		 *    8049149:       31 c0                   xor    %eax,%eax
		 *    804914b:       5d                      pop    %ebp
		 *    804914c:       c3                      ret
		 */
		return NULL;
	}
#if defined(DUK_ALLOC_POOL_ROMPTR_COMPRESSION)
	if (val >= DUK_ALLOC_POOL_ROMPTR_FIRST) {
		/* This is a blind lookup, could check index validity.
		 * Duktape should never decompress a pointer which would
		 * be out-of-bounds here.
		 */
		return (void *) (intptr_t) (duk_rom_compressed_pointers[val - DUK_ALLOC_POOL_ROMPTR_FIRST]);
	}
#endif
	return (void *) ((char *) duk_alloc_pool_ptrcomp_base + (((size_t) val) << 2));
}
#endif

#if defined(__cplusplus)
}
#endif  /* end 'extern "C"' wrapper */

#endif  /* DUK_ALLOC_POOL_H_INCLUDED */
