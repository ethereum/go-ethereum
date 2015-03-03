/*
	Copyright 2009 NVIDIA Corporation.  All rights reserved.

	NOTICE TO LICENSEE:   

	This source code and/or documentation ("Licensed Deliverables") are subject 
	to NVIDIA intellectual property rights under U.S. and international Copyright 
	laws.  

	These Licensed Deliverables contained herein is PROPRIETARY and CONFIDENTIAL 
	to NVIDIA and is being provided under the terms and conditions of a form of 
	NVIDIA software license agreement by and between NVIDIA and Licensee ("License 
	Agreement") or electronically accepted by Licensee.  Notwithstanding any terms 
	or conditions to the contrary in the License Agreement, reproduction or 
	disclosure of the Licensed Deliverables to any third party without the express 
	written consent of NVIDIA is prohibited.     

	NOTWITHSTANDING ANY TERMS OR CONDITIONS TO THE CONTRARY IN THE LICENSE AGREEMENT, 
	NVIDIA MAKES NO REPRESENTATION ABOUT THE SUITABILITY OF THESE LICENSED 
	DELIVERABLES FOR ANY PURPOSE.  IT IS PROVIDED "AS IS" WITHOUT EXPRESS OR IMPLIED 
	WARRANTY OF ANY KIND. NVIDIA DISCLAIMS ALL WARRANTIES WITH REGARD TO THESE 
	LICENSED DELIVERABLES, INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY, 
	NONINFRINGEMENT, AND FITNESS FOR A PARTICULAR PURPOSE.   NOTWITHSTANDING ANY 
	TERMS OR CONDITIONS TO THE CONTRARY IN THE LICENSE AGREEMENT, IN NO EVENT SHALL 
	NVIDIA BE LIABLE FOR ANY SPECIAL, INDIRECT, INCIDENTAL, OR CONSEQUENTIAL DAMAGES, 
	OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS,	WHETHER 
	IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION,  ARISING OUT OF 
	OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THESE LICENSED DELIVERABLES.  

	U.S. Government End Users. These Licensed Deliverables are a "commercial item" 
	as that term is defined at  48 C.F.R. 2.101 (OCT 1995), consisting  of 
	"commercial computer  software"  and "commercial computer software documentation" 
	as such terms are  used in 48 C.F.R. 12.212 (SEPT 1995) and is provided to the 
	U.S. Government only as a commercial end item.  Consistent with 48 C.F.R.12.212 
	and 48 C.F.R. 227.7202-1 through 227.7202-4 (JUNE 1995), all U.S. Government 
	End Users acquire the Licensed Deliverables with only those rights set forth 
	herein. 

	Any use of the Licensed Deliverables in individual and commercial software must 
	include, in the user documentation and internal comments to the code, the above 
	Disclaimer and U.S. Government End Users Notice.
 */

/*
 *	cuPrintf.cu
 *
 *	This is a printf command callable from within a kernel. It is set
 *	up so that output is sent to a memory buffer, which is emptied from
 *	the host side - but only after a cudaThreadSynchronize() on the host.
 *
 *	Currently, there is a limitation of around 200 characters of output
 *	and no more than 10 arguments to a single cuPrintf() call. Issue
 *	multiple calls if longer format strings are required.
 *
 *	It requires minimal setup, and is *NOT* optimised for performance.
 *	For example, writes are not coalesced - this is because there is an
 *	assumption that people will not want to printf from every single one
 *	of thousands of threads, but only from individual threads at a time.
 *
 *	Using this is simple - it requires one host-side call to initialise
 *	everything, and then kernels can call cuPrintf at will. Sample code
 *	is the easiest way to demonstrate:
 *
	#include "cuPrintf.cu"
 	
	__global__ void testKernel(int val)
	{
		cuPrintf("Value is: %d\n", val);
	}

	int main()
	{
		cudaPrintfInit();
		testKernel<<< 2, 3 >>>(10);
		cudaPrintfDisplay(stdout, true);
		cudaPrintfEnd();
        return 0;
	}
 *
 *	See the header file, "cuPrintf.cuh" for more info, especially
 *	arguments to cudaPrintfInit() and cudaPrintfDisplay();
 */

#ifndef CUPRINTF_CU
#define CUPRINTF_CU

#include "cuPrintf.cuh"
#if __CUDA_ARCH__ > 100      // Atomics only used with > sm_10 architecture
#include <sm_11_atomic_functions.h>
#endif

// This is the smallest amount of memory, per-thread, which is allowed.
// It is also the largest amount of space a single printf() can take up
const static int CUPRINTF_MAX_LEN = 256;

// This structure is used internally to track block/thread output restrictions.
typedef struct __align__(8) {
	int threadid;				// CUPRINTF_UNRESTRICTED for unrestricted
	int blockid;				// CUPRINTF_UNRESTRICTED for unrestricted
} cuPrintfRestriction;

// The main storage is in a global print buffer, which has a known
// start/end/length. These are atomically updated so it works as a
// circular buffer.
// Since the only control primitive that can be used is atomicAdd(),
// we cannot wrap the pointer as such. The actual address must be
// calculated from printfBufferPtr by mod-ing with printfBufferLength.
// For sm_10 architecture, we must subdivide the buffer per-thread
// since we do not even have an atomic primitive.
__constant__ static char *globalPrintfBuffer = NULL;         // Start of circular buffer (set up by host)
__constant__ static int printfBufferLength = 0;              // Size of circular buffer (set up by host)
__device__ static cuPrintfRestriction restrictRules;         // Output restrictions
__device__ volatile static char *printfBufferPtr = NULL;     // Current atomically-incremented non-wrapped offset

// This is the header preceeding all printf entries.
// NOTE: It *must* be size-aligned to the maximum entity size (size_t)
typedef struct __align__(8) {
    unsigned short magic;                   // Magic number says we're valid
    unsigned short fmtoffset;               // Offset of fmt string into buffer
    unsigned short blockid;                 // Block ID of author
    unsigned short threadid;                // Thread ID of author
} cuPrintfHeader;

// Special header for sm_10 architecture
#define CUPRINTF_SM10_MAGIC   0xC810        // Not a valid ascii character
typedef struct __align__(16) {
    unsigned short magic;                   // sm_10 specific magic number
    unsigned short unused;
    unsigned int thread_index;              // thread ID for this buffer
    unsigned int thread_buf_len;            // per-thread buffer length
    unsigned int offset;                    // most recent printf's offset
} cuPrintfHeaderSM10;


// Because we can't write an element which is not aligned to its bit-size,
// we have to align all sizes and variables on maximum-size boundaries.
// That means sizeof(double) in this case, but we'll use (long long) for
// better arch<1.3 support
#define CUPRINTF_ALIGN_SIZE      sizeof(long long)

// All our headers are prefixed with a magic number so we know they're ready
#define CUPRINTF_SM11_MAGIC  (unsigned short)0xC811        // Not a valid ascii character


//
//  getNextPrintfBufPtr
//
//  Grabs a block of space in the general circular buffer, using an
//  atomic function to ensure that it's ours. We handle wrapping
//  around the circular buffer and return a pointer to a place which
//  can be written to.
//
//  Important notes:
//      1. We always grab CUPRINTF_MAX_LEN bytes
//      2. Because of 1, we never worry about wrapping around the end
//      3. Because of 1, printfBufferLength *must* be a factor of CUPRINTF_MAX_LEN
//
//  This returns a pointer to the place where we own.
//
__device__ static char *getNextPrintfBufPtr()
{
    // Initialisation check
    if(!printfBufferPtr)
        return NULL;

	// Thread/block restriction check
	if((restrictRules.blockid != CUPRINTF_UNRESTRICTED) && (restrictRules.blockid != (blockIdx.x + gridDim.x*blockIdx.y)))
		return NULL;
	if((restrictRules.threadid != CUPRINTF_UNRESTRICTED) && (restrictRules.threadid != (threadIdx.x + blockDim.x*threadIdx.y + blockDim.x*blockDim.y*threadIdx.z)))
		return NULL;

	// Conditional section, dependent on architecture
#if __CUDA_ARCH__ == 100
    // For sm_10 architectures, we have no atomic add - this means we must split the
    // entire available buffer into per-thread blocks. Inefficient, but what can you do.
    int thread_count = (gridDim.x * gridDim.y) * (blockDim.x * blockDim.y * blockDim.z);
    int thread_index = threadIdx.x + blockDim.x*threadIdx.y + blockDim.x*blockDim.y*threadIdx.z +
                       (blockIdx.x + gridDim.x*blockIdx.y) * (blockDim.x * blockDim.y * blockDim.z);
    
    // Find our own block of data and go to it. Make sure the per-thread length
	// is a precise multiple of CUPRINTF_MAX_LEN, otherwise we risk size and
	// alignment issues! We must round down, of course.
    unsigned int thread_buf_len = printfBufferLength / thread_count;
	thread_buf_len &= ~(CUPRINTF_MAX_LEN-1);

	// We *must* have a thread buffer length able to fit at least two printfs (one header, one real)
	if(thread_buf_len < (CUPRINTF_MAX_LEN * 2))
		return NULL;

	// Now address our section of the buffer. The first item is a header.
    char *myPrintfBuffer = globalPrintfBuffer + (thread_buf_len * thread_index);
    cuPrintfHeaderSM10 hdr = *(cuPrintfHeaderSM10 *)(void *)myPrintfBuffer;
    if(hdr.magic != CUPRINTF_SM10_MAGIC)
    {
        // If our header is not set up, initialise it
        hdr.magic = CUPRINTF_SM10_MAGIC;
        hdr.thread_index = thread_index;
        hdr.thread_buf_len = thread_buf_len;
        hdr.offset = 0;         // Note we start at 0! We pre-increment below.
        *(cuPrintfHeaderSM10 *)(void *)myPrintfBuffer = hdr;       // Write back the header

        // For initial setup purposes, we might need to init thread0's header too
        // (so that cudaPrintfDisplay() below will work). This is only run once.
        cuPrintfHeaderSM10 *tophdr = (cuPrintfHeaderSM10 *)(void *)globalPrintfBuffer;
        tophdr->thread_buf_len = thread_buf_len;
    }

    // Adjust the offset by the right amount, and wrap it if need be
    unsigned int offset = hdr.offset + CUPRINTF_MAX_LEN;
    if(offset >= hdr.thread_buf_len)
        offset = CUPRINTF_MAX_LEN;

    // Write back the new offset for next time and return a pointer to it
    ((cuPrintfHeaderSM10 *)(void *)myPrintfBuffer)->offset = offset;
    return myPrintfBuffer + offset;
#else
    // Much easier with an atomic operation!
    size_t offset = atomicAdd((unsigned int *)&printfBufferPtr, CUPRINTF_MAX_LEN) - (size_t)globalPrintfBuffer;
    offset %= printfBufferLength;
    return globalPrintfBuffer + offset;
#endif
}


//
//  writePrintfHeader
//
//  Inserts the header for containing our UID, fmt position and
//  block/thread number. We generate it dynamically to avoid
//	issues arising from requiring pre-initialisation.
//
__device__ static void writePrintfHeader(char *ptr, char *fmtptr)
{
    if(ptr)
    {
        cuPrintfHeader header;
        header.magic = CUPRINTF_SM11_MAGIC;
        header.fmtoffset = (unsigned short)(fmtptr - ptr);
        header.blockid = blockIdx.x + gridDim.x*blockIdx.y;
        header.threadid = threadIdx.x + blockDim.x*threadIdx.y + blockDim.x*blockDim.y*threadIdx.z;
        *(cuPrintfHeader *)(void *)ptr = header;
    }
}


//
//  cuPrintfStrncpy
//
//  This special strncpy outputs an aligned length value, followed by the
//  string. It then zero-pads the rest of the string until a 64-aligned
//  boundary. The length *includes* the padding. A pointer to the byte
//  just after the \0 is returned.
//
//  This function could overflow CUPRINTF_MAX_LEN characters in our buffer.
//  To avoid it, we must count as we output and truncate where necessary.
//
__device__ static char *cuPrintfStrncpy(char *dest, const char *src, int n, char *end)
{
    // Initialisation and overflow check
    if(!dest || !src || (dest >= end))
        return NULL;

    // Prepare to write the length specifier. We're guaranteed to have
    // at least "CUPRINTF_ALIGN_SIZE" bytes left because we only write out in
    // chunks that size, and CUPRINTF_MAX_LEN is aligned with CUPRINTF_ALIGN_SIZE.
    int *lenptr = (int *)(void *)dest;
    int len = 0;
    dest += CUPRINTF_ALIGN_SIZE;

    // Now copy the string
    while(n--)
    {
        if(dest >= end)     // Overflow check
            break;

        len++;
        *dest++ = *src;
        if(*src++ == '\0')
            break;
    }

    // Now write out the padding bytes, and we have our length.
    while((dest < end) && (((long)dest & (CUPRINTF_ALIGN_SIZE-1)) != 0))
    {
        len++;
        *dest++ = 0;
    }
    *lenptr = len;
    return (dest < end) ? dest : NULL;        // Overflow means return NULL
}


//
//  copyArg
//
//  This copies a length specifier and then the argument out to the
//  data buffer. Templates let the compiler figure all this out at
//  compile-time, making life much simpler from the programming
//  point of view. I'm assuimg all (const char *) is a string, and
//  everything else is the variable it points at. I'd love to see
//  a better way of doing it, but aside from parsing the format
//  string I can't think of one.
//
//  The length of the data type is inserted at the beginning (so that
//  the display can distinguish between float and double), and the
//  pointer to the end of the entry is returned.
//
__device__ static char *copyArg(char *ptr, const char *arg, char *end)
{
    // Initialisation check
    if(!ptr || !arg)
        return NULL;

    // strncpy does all our work. We just terminate.
    if((ptr = cuPrintfStrncpy(ptr, arg, CUPRINTF_MAX_LEN, end)) != NULL)
        *ptr = 0;

    return ptr;
}

template <typename T>
__device__ static char *copyArg(char *ptr, T &arg, char *end)
{
    // Initisalisation and overflow check. Alignment rules mean that
    // we're at least CUPRINTF_ALIGN_SIZE away from "end", so we only need
    // to check that one offset.
    if(!ptr || ((ptr+CUPRINTF_ALIGN_SIZE) >= end))
        return NULL;

    // Write the length and argument
    *(int *)(void *)ptr = sizeof(arg);
    ptr += CUPRINTF_ALIGN_SIZE;
    *(T *)(void *)ptr = arg;
    ptr += CUPRINTF_ALIGN_SIZE;
    *ptr = 0;

    return ptr;
}


//
//  cuPrintf
//
//  Templated printf functions to handle multiple arguments.
//  Note we return the total amount of data copied, not the number
//  of characters output. But then again, who ever looks at the
//  return from printf() anyway?
//
//  The format is to grab a block of circular buffer space, the
//  start of which will hold a header and a pointer to the format
//  string. We then write in all the arguments, and finally the
//  format string itself. This is to make it easy to prevent
//  overflow of our buffer (we support up to 10 arguments, each of
//  which can be 12 bytes in length - that means that only the
//  format string (or a %s) can actually overflow; so the overflow
//  check need only be in the strcpy function.
//
//  The header is written at the very last because that's what
//  makes it look like we're done.
//
//  Errors, which are basically lack-of-initialisation, are ignored
//  in the called functions because NULL pointers are passed around
//

// All printf variants basically do the same thing, setting up the
// buffer, writing all arguments, then finalising the header. For
// clarity, we'll pack the code into some big macros.
#define CUPRINTF_PREAMBLE \
    char *start, *end, *bufptr, *fmtstart; \
    if((start = getNextPrintfBufPtr()) == NULL) return 0; \
    end = start + CUPRINTF_MAX_LEN; \
    bufptr = start + sizeof(cuPrintfHeader);

// Posting an argument is easy
#define CUPRINTF_ARG(argname) \
	bufptr = copyArg(bufptr, argname, end);

// After args are done, record start-of-fmt and write the fmt and header
#define CUPRINTF_POSTAMBLE \
    fmtstart = bufptr; \
    end = cuPrintfStrncpy(bufptr, fmt, CUPRINTF_MAX_LEN, end); \
    writePrintfHeader(start, end ? fmtstart : NULL); \
    return end ? (int)(end - start) : 0;

__device__ int cuPrintf(const char *fmt)
{
	CUPRINTF_PREAMBLE;

	CUPRINTF_POSTAMBLE;
}
template <typename T1> __device__ int cuPrintf(const char *fmt, T1 arg1)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);
	CUPRINTF_ARG(arg6);
	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);
	CUPRINTF_ARG(arg6);
	CUPRINTF_ARG(arg7);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8)
{
	CUPRINTF_PREAMBLE;

	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);
	CUPRINTF_ARG(arg6);
	CUPRINTF_ARG(arg7);
	CUPRINTF_ARG(arg8);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8, typename T9> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8, T9 arg9)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);
	CUPRINTF_ARG(arg6);
	CUPRINTF_ARG(arg7);
	CUPRINTF_ARG(arg8);
	CUPRINTF_ARG(arg9);

	CUPRINTF_POSTAMBLE;
}
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8, typename T9, typename T10> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8, T9 arg9, T10 arg10)
{
	CUPRINTF_PREAMBLE;
	    
	CUPRINTF_ARG(arg1);
	CUPRINTF_ARG(arg2);
	CUPRINTF_ARG(arg3);
	CUPRINTF_ARG(arg4);
	CUPRINTF_ARG(arg5);
	CUPRINTF_ARG(arg6);
	CUPRINTF_ARG(arg7);
	CUPRINTF_ARG(arg8);
	CUPRINTF_ARG(arg9);
	CUPRINTF_ARG(arg10);

	CUPRINTF_POSTAMBLE;
}
#undef CUPRINTF_PREAMBLE
#undef CUPRINTF_ARG
#undef CUPRINTF_POSTAMBLE


//
//	cuPrintfRestrict
//
//	Called to restrict output to a given thread/block.
//	We store the info in "restrictRules", which is set up at
//	init time by the host. It's not the cleanest way to do this
//	because it means restrictions will last between
//	invocations, but given the output-pointer continuity,
//	I feel this is reasonable.
//
__device__ void cuPrintfRestrict(int threadid, int blockid)
{
    int thread_count = blockDim.x * blockDim.y * blockDim.z;
	if(((threadid < thread_count) && (threadid >= 0)) || (threadid == CUPRINTF_UNRESTRICTED))
		restrictRules.threadid = threadid;

	int block_count = gridDim.x * gridDim.y;
	if(((blockid < block_count) && (blockid >= 0)) || (blockid == CUPRINTF_UNRESTRICTED))
		restrictRules.blockid = blockid;
}


///////////////////////////////////////////////////////////////////////////////
// HOST SIDE

#include <stdio.h>
static FILE *printf_fp;

static char *printfbuf_start=NULL;
static char *printfbuf_device=NULL;
static int printfbuf_len=0;


//
//  outputPrintfData
//
//  Our own internal function, which takes a pointer to a data buffer
//  and passes it through libc's printf for output.
//
//  We receive the formate string and a pointer to where the data is
//  held. We then run through and print it out.
//
//  Returns 0 on failure, 1 on success
//
static int outputPrintfData(char *fmt, char *data)
{
    // Format string is prefixed by a length that we don't need
    fmt += CUPRINTF_ALIGN_SIZE;

    // Now run through it, printing everything we can. We must
    // run to every % character, extract only that, and use printf
    // to format it.
    char *p = strchr(fmt, '%');
    while(p != NULL)
    {
        // Print up to the % character
        *p = '\0';
        fputs(fmt, printf_fp);
        *p = '%';           // Put back the %

        // Now handle the format specifier
        char *format = p++;         // Points to the '%'
        p += strcspn(p, "%cdiouxXeEfgGaAnps");
        if(*p == '\0')              // If no format specifier, print the whole thing
        {
            fmt = format;
            break;
        }

        // Cut out the format bit and use printf to print it. It's prefixed
        // by its length.
        int arglen = *(int *)data;
        if(arglen > CUPRINTF_MAX_LEN)
        {
            fputs("Corrupt printf buffer data - aborting\n", printf_fp);
            return 0;
        }

        data += CUPRINTF_ALIGN_SIZE;
        
        char specifier = *p++;
        char c = *p;        // Store for later
        *p = '\0';
        switch(specifier)
        {
            // These all take integer arguments
            case 'c':
            case 'd':
            case 'i':
            case 'o':
            case 'u':
            case 'x':
            case 'X':
            case 'p':
                fprintf(printf_fp, format, *((int *)data));
                break;

            // These all take double arguments
            case 'e':
            case 'E':
            case 'f':
            case 'g':
            case 'G':
            case 'a':
            case 'A':
                if(arglen == 4)     // Float vs. Double thing
                    fprintf(printf_fp, format, *((float *)data));
                else
                    fprintf(printf_fp, format, *((double *)data));
                break;

            // Strings are handled in a special way
            case 's':
                fprintf(printf_fp, format, (char *)data);
                break;

            // % is special
            case '%':
                fprintf(printf_fp, "%%");
                break;

            // Everything else is just printed out as-is
            default:
                fprintf(printf_fp, format);
                break;
        }
        data += CUPRINTF_ALIGN_SIZE;         // Move on to next argument
        *p = c;                     // Restore what we removed
        fmt = p;                    // Adjust fmt string to be past the specifier
        p = strchr(fmt, '%');       // and get the next specifier
    }

    // Print out the last of the string
    fputs(fmt, printf_fp);
    return 1;
}


//
//  doPrintfDisplay
//
//  This runs through the blocks of CUPRINTF_MAX_LEN-sized data, calling the
//  print function above to display them. We've got this separate from
//  cudaPrintfDisplay() below so we can handle the SM_10 architecture
//  partitioning.
//
static int doPrintfDisplay(int headings, int clear, char *bufstart, char *bufend, char *bufptr, char *endptr)
{
    // Grab, piece-by-piece, each output element until we catch
    // up with the circular buffer end pointer
    int printf_count=0;
    char printfbuf_local[CUPRINTF_MAX_LEN+1];
    printfbuf_local[CUPRINTF_MAX_LEN] = '\0';

    while(bufptr != endptr)
    {
        // Wrap ourselves at the end-of-buffer
        if(bufptr == bufend)
            bufptr = bufstart;

        // Adjust our start pointer to within the circular buffer and copy a block.
        cudaMemcpy(printfbuf_local, bufptr, CUPRINTF_MAX_LEN, cudaMemcpyDeviceToHost);

        // If the magic number isn't valid, then this write hasn't gone through
        // yet and we'll wait until it does (or we're past the end for non-async printfs).
        cuPrintfHeader *hdr = (cuPrintfHeader *)printfbuf_local;
        if((hdr->magic != CUPRINTF_SM11_MAGIC) || (hdr->fmtoffset >= CUPRINTF_MAX_LEN))
        {
            //fprintf(printf_fp, "Bad magic number in printf header\n");
            break;
        }

        // Extract all the info and get this printf done
        if(headings)
            fprintf(printf_fp, "[%d, %d]: ", hdr->blockid, hdr->threadid);
        if(hdr->fmtoffset == 0)
            fprintf(printf_fp, "printf buffer overflow\n");
        else if(!outputPrintfData(printfbuf_local+hdr->fmtoffset, printfbuf_local+sizeof(cuPrintfHeader)))
            break;
        printf_count++;

        // Clear if asked
        if(clear)
            cudaMemset(bufptr, 0, CUPRINTF_MAX_LEN);

        // Now advance our start location, because we're done, and keep copying
        bufptr += CUPRINTF_MAX_LEN;
    }

    return printf_count;
}


//
//  cudaPrintfInit
//
//  Takes a buffer length to allocate, creates the memory on the device and
//  returns a pointer to it for when a kernel is called. It's up to the caller
//  to free it.
//
extern "C" cudaError_t cudaPrintfInit(size_t bufferLen)
{
    // Fix up bufferlen to be a multiple of CUPRINTF_MAX_LEN
    bufferLen = (bufferLen < CUPRINTF_MAX_LEN) ? CUPRINTF_MAX_LEN : bufferLen;
    if((bufferLen % CUPRINTF_MAX_LEN) > 0)
        bufferLen += (CUPRINTF_MAX_LEN - (bufferLen % CUPRINTF_MAX_LEN));
    printfbuf_len = (int)bufferLen;

    // Allocate a print buffer on the device and zero it
    if(cudaMalloc((void **)&printfbuf_device, printfbuf_len) != cudaSuccess)
		return cudaErrorInitializationError;
    cudaMemset(printfbuf_device, 0, printfbuf_len);
    printfbuf_start = printfbuf_device;         // Where we start reading from

	// No restrictions to begin with
	cuPrintfRestriction restrict;
	restrict.threadid = restrict.blockid = CUPRINTF_UNRESTRICTED;
	cudaMemcpyToSymbol(restrictRules, &restrict, sizeof(restrict));

    // Initialise the buffer and the respective lengths/pointers.
    cudaMemcpyToSymbol(globalPrintfBuffer, &printfbuf_device, sizeof(char *));
    cudaMemcpyToSymbol(printfBufferPtr, &printfbuf_device, sizeof(char *));
    cudaMemcpyToSymbol(printfBufferLength, &printfbuf_len, sizeof(printfbuf_len));

    return cudaSuccess;
}


//
//  cudaPrintfEnd
//
//  Frees up the memory which we allocated
//
extern "C" void cudaPrintfEnd()
{
    if(!printfbuf_start || !printfbuf_device)
        return;

    cudaFree(printfbuf_device);
    printfbuf_start = printfbuf_device = NULL;
}


//
//  cudaPrintfDisplay
//
//  Each call to this function dumps the entire current contents
//	of the printf buffer to the pre-specified FILE pointer. The
//	circular "start" pointer is advanced so that subsequent calls
//	dumps only new stuff.
//
//  In the case of async memory access (via streams), call this
//  repeatedly to keep trying to empty the buffer. If it's a sync
//  access, then the whole buffer should empty in one go.
//
//	Arguments:
//		outputFP     - File descriptor to output to (NULL => stdout)
//		showThreadID - If true, prints [block,thread] before each line
//
extern "C" cudaError_t cudaPrintfDisplay(void *outputFP, bool showThreadID)
{
	printf_fp = (FILE *)((outputFP == NULL) ? stdout : outputFP);

    // For now, we force "synchronous" mode which means we're not concurrent
	// with kernel execution. This also means we don't need clearOnPrint.
	// If you're patching it for async operation, here's where you want it.
    bool sync_printfs = true;
	bool clearOnPrint = false;

    // Initialisation check
    if(!printfbuf_start || !printfbuf_device || !printf_fp)
        return cudaErrorMissingConfiguration;

    // To determine which architecture we're using, we read the
    // first short from the buffer - it'll be the magic number
    // relating to the version.
    unsigned short magic;
    cudaMemcpy(&magic, printfbuf_device, sizeof(unsigned short), cudaMemcpyDeviceToHost);

    // For SM_10 architecture, we've split our buffer into one-per-thread.
    // That means we must do each thread block separately. It'll require
    // extra reading. We also, for now, don't support async printfs because
    // that requires tracking one start pointer per thread.
    if(magic == CUPRINTF_SM10_MAGIC)
    {
        sync_printfs = true;
	    clearOnPrint = false;
        int blocklen = 0;
        char *blockptr = printfbuf_device;
        while(blockptr < (printfbuf_device + printfbuf_len))
        {
            cuPrintfHeaderSM10 hdr;
            cudaMemcpy(&hdr, blockptr, sizeof(hdr), cudaMemcpyDeviceToHost);

            // We get our block-size-step from the very first header
            if(hdr.thread_buf_len != 0)
                blocklen = hdr.thread_buf_len;

            // No magic number means no printfs from this thread
            if(hdr.magic != CUPRINTF_SM10_MAGIC)
            {
                if(blocklen == 0)
                {
                    fprintf(printf_fp, "No printf headers found at all!\n");
                    break;                              // No valid headers!
                }
                blockptr += blocklen;
                continue;
            }

            // "offset" is non-zero then we can print the block contents
            if(hdr.offset > 0)
            {
                // For synchronous printfs, we must print from endptr->bufend, then from start->end
                if(sync_printfs)
                    doPrintfDisplay(showThreadID, clearOnPrint, blockptr+CUPRINTF_MAX_LEN, blockptr+hdr.thread_buf_len, blockptr+hdr.offset+CUPRINTF_MAX_LEN, blockptr+hdr.thread_buf_len);
                doPrintfDisplay(showThreadID, clearOnPrint, blockptr+CUPRINTF_MAX_LEN, blockptr+hdr.thread_buf_len, blockptr+CUPRINTF_MAX_LEN, blockptr+hdr.offset+CUPRINTF_MAX_LEN);
            }

            // Move on to the next block and loop again
            blockptr += hdr.thread_buf_len;
        }
    }
    // For SM_11 and up, everything is a single buffer and it's simple
    else if(magic == CUPRINTF_SM11_MAGIC)
    {
	    // Grab the current "end of circular buffer" pointer.
        char *printfbuf_end = NULL;
        cudaMemcpyFromSymbol(&printfbuf_end, printfBufferPtr, sizeof(char *));

        // Adjust our starting and ending pointers to within the block
        char *bufptr = ((printfbuf_start - printfbuf_device) % printfbuf_len) + printfbuf_device;
        char *endptr = ((printfbuf_end - printfbuf_device) % printfbuf_len) + printfbuf_device;

        // For synchronous (i.e. after-kernel-exit) printf display, we have to handle circular
        // buffer wrap carefully because we could miss those past "end".
        if(sync_printfs)
            doPrintfDisplay(showThreadID, clearOnPrint, printfbuf_device, printfbuf_device+printfbuf_len, endptr, printfbuf_device+printfbuf_len);
        doPrintfDisplay(showThreadID, clearOnPrint, printfbuf_device, printfbuf_device+printfbuf_len, bufptr, endptr);

        printfbuf_start = printfbuf_end;
    }
    else
        ;//printf("Bad magic number in cuPrintf buffer header\n");

    // If we were synchronous, then we must ensure that the memory is cleared on exit
    // otherwise another kernel launch with a different grid size could conflict.
    if(sync_printfs)
        cudaMemset(printfbuf_device, 0, printfbuf_len);

    return cudaSuccess;
}

// Cleanup
#undef CUPRINTF_MAX_LEN
#undef CUPRINTF_ALIGN_SIZE
#undef CUPRINTF_SM10_MAGIC
#undef CUPRINTF_SM11_MAGIC

#endif
