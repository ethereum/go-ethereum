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

#ifndef CUPRINTF_H
#define CUPRINTF_H

/*
 *	This is the header file supporting cuPrintf.cu and defining both
 *	the host and device-side interfaces. See that file for some more
 *	explanation and sample use code. See also below for details of the
 *	host-side interfaces.
 *
 *  Quick sample code:
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
 */

///////////////////////////////////////////////////////////////////////////////
// DEVICE SIDE
// External function definitions for device-side code

// Abuse of templates to simulate varargs
__device__ int cuPrintf(const char *fmt);
template <typename T1> __device__ int cuPrintf(const char *fmt, T1 arg1);
template <typename T1, typename T2> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2);
template <typename T1, typename T2, typename T3> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3);
template <typename T1, typename T2, typename T3, typename T4> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4);
template <typename T1, typename T2, typename T3, typename T4, typename T5> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5);
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6);
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7);
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8);
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8, typename T9> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8, T9 arg9);
template <typename T1, typename T2, typename T3, typename T4, typename T5, typename T6, typename T7, typename T8, typename T9, typename T10> __device__ int cuPrintf(const char *fmt, T1 arg1, T2 arg2, T3 arg3, T4 arg4, T5 arg5, T6 arg6, T7 arg7, T8 arg8, T9 arg9, T10 arg10);


//
//	cuPrintfRestrict
//
//	Called to restrict output to a given thread/block. Pass
//	the constant CUPRINTF_UNRESTRICTED to unrestrict output
//	for thread/block IDs. Note you can therefore allow
//	"all printfs from block 3" or "printfs from thread 2
//	on all blocks", or "printfs only from block 1, thread 5".
//
//	Arguments:
//		threadid - Thread ID to allow printfs from
//		blockid - Block ID to allow printfs from
//
//	NOTE: Restrictions last between invocations of
//	kernels unless cudaPrintfInit() is called again.
//
#define CUPRINTF_UNRESTRICTED	-1
__device__ void cuPrintfRestrict(int threadid, int blockid);



///////////////////////////////////////////////////////////////////////////////
// HOST SIDE
// External function definitions for host-side code

//
//	cudaPrintfInit
//
//	Call this once to initialise the printf system. If the output
//	file or buffer size needs to be changed, call cudaPrintfEnd()
//	before re-calling cudaPrintfInit().
//
//	The default size for the buffer is 1 megabyte. For CUDA
//	architecture 1.1 and above, the buffer is filled linearly and
//	is completely used;	however for architecture 1.0, the buffer
//	is divided into as many segments are there are threads, even
//	if some threads do not call cuPrintf().
//
//	Arguments:
//		bufferLen - Length, in bytes, of total space to reserve
//		            (in device global memory) for output.
//
//	Returns:
//		cudaSuccess if all is well.
//
extern "C" cudaError_t cudaPrintfInit(size_t bufferLen=1048576);   // 1-meg - that's enough for 4096 printfs by all threads put together

//
//	cudaPrintfEnd
//
//	Cleans up all memories allocated by cudaPrintfInit().
//	Call this at exit, or before calling cudaPrintfInit() again.
//
extern "C" void cudaPrintfEnd();

//
//	cudaPrintfDisplay
//
//	Dumps the contents of the output buffer to the specified
//	file pointer. If the output pointer is not specified,
//	the default "stdout" is used.
//
//	Arguments:
//		outputFP     - A file pointer to an output stream.
//		showThreadID - If "true", output strings are prefixed
//		               by "[blockid, threadid] " at output.
//
//	Returns:
//		cudaSuccess if all is well.
//
extern "C" cudaError_t cudaPrintfDisplay(void *outputFP=NULL, bool showThreadID=false);

#endif  // CUPRINTF_H
