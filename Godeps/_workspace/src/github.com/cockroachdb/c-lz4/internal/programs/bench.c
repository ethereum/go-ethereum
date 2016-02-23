/*
    bench.c - Demo program to benchmark open-source compression algorithms
    Copyright (C) Yann Collet 2012-2015

    GPL v2 License

    This program is free software; you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation; either version 2 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License along
    with this program; if not, write to the Free Software Foundation, Inc.,
    51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

    You can contact the author at :
    - LZ4 source repository : https://github.com/Cyan4973/lz4
    - LZ4 public forum : https://groups.google.com/forum/#!forum/lz4c
*/

/**************************************
*  Compiler Options
***************************************/
#if defined(_MSC_VER) || defined(_WIN32)
#  define _CRT_SECURE_NO_WARNINGS
#  define _CRT_SECURE_NO_DEPRECATE     /* VS2005 */
#  define BMK_LEGACY_TIMER 1           /* S_ISREG & gettimeofday() are not supported by MSVC */
#endif

/* Unix Large Files support (>4GB) */
#define _FILE_OFFSET_BITS 64
#if (defined(__sun__) && (!defined(__LP64__)))   /* Sun Solaris 32-bits requires specific definitions */
#  define _LARGEFILE_SOURCE
#elif ! defined(__LP64__)                        /* No point defining Large file for 64 bit */
#  define _LARGEFILE64_SOURCE
#endif


/**************************************
*  Includes
***************************************/
#include <stdlib.h>      /* malloc */
#include <stdio.h>       /* fprintf, fopen, ftello64 */
#include <sys/types.h>   /* stat64 */
#include <sys/stat.h>    /* stat64 */

/* Use ftime() if gettimeofday() is not available on your target */
#if defined(BMK_LEGACY_TIMER)
#  include <sys/timeb.h>   /* timeb, ftime */
#else
#  include <sys/time.h>    /* gettimeofday */
#endif

#include "lz4.h"
#define COMPRESSOR0 LZ4_compress_local
static int LZ4_compress_local(const char* src, char* dst, int srcSize, int dstSize, int clevel) { (void)clevel; return LZ4_compress_default(src, dst, srcSize, dstSize); }
#include "lz4hc.h"
#define COMPRESSOR1 LZ4_compress_HC
#define DEFAULTCOMPRESSOR COMPRESSOR0

#include "xxhash.h"


/**************************************
*  Compiler specifics
***************************************/
#if !defined(S_ISREG)
#  define S_ISREG(x) (((x) & S_IFMT) == S_IFREG)
#endif


/**************************************
*  Basic Types
***************************************/
#if defined (__STDC_VERSION__) && __STDC_VERSION__ >= 199901L   /* C99 */
# include <stdint.h>
  typedef uint8_t  BYTE;
  typedef uint16_t U16;
  typedef uint32_t U32;
  typedef  int32_t S32;
  typedef uint64_t U64;
#else
  typedef unsigned char       BYTE;
  typedef unsigned short      U16;
  typedef unsigned int        U32;
  typedef   signed int        S32;
  typedef unsigned long long  U64;
#endif


/**************************************
*  Constants
***************************************/
#define NBLOOPS    3
#define TIMELOOP   2000

#define KB *(1 <<10)
#define MB *(1 <<20)
#define GB *(1U<<30)

#define MAX_MEM             (2 GB - 64 MB)
#define DEFAULT_CHUNKSIZE   (4 MB)


/**************************************
*  Local structures
***************************************/
struct chunkParameters
{
    U32   id;
    char* origBuffer;
    char* compressedBuffer;
    int   origSize;
    int   compressedSize;
};

struct compressionParameters
{
    int (*compressionFunction)(const char* src, char* dst, int srcSize, int dstSize, int cLevel);
    int (*decompressionFunction)(const char* src, char* dst, int dstSize);
};


/**************************************
*  MACRO
***************************************/
#define DISPLAY(...) fprintf(stderr, __VA_ARGS__)


/**************************************
*  Benchmark Parameters
***************************************/
static int chunkSize = DEFAULT_CHUNKSIZE;
static int nbIterations = NBLOOPS;
static int BMK_pause = 0;

void BMK_setBlocksize(int bsize) { chunkSize = bsize; }

void BMK_setNbIterations(int nbLoops)
{
    nbIterations = nbLoops;
    DISPLAY("- %i iterations -\n", nbIterations);
}

void BMK_setPause(void) { BMK_pause = 1; }


/*********************************************************
*  Private functions
**********************************************************/

#if defined(BMK_LEGACY_TIMER)

static int BMK_GetMilliStart(void)
{
  /* Based on Legacy ftime()
     Rolls over every ~ 12.1 days (0x100000/24/60/60)
     Use GetMilliSpan to correct for rollover */
  struct timeb tb;
  int nCount;
  ftime( &tb );
  nCount = (int) (tb.millitm + (tb.time & 0xfffff) * 1000);
  return nCount;
}

#else

static int BMK_GetMilliStart(void)
{
  /* Based on newer gettimeofday()
     Use GetMilliSpan to correct for rollover */
  struct timeval tv;
  int nCount;
  gettimeofday(&tv, NULL);
  nCount = (int) (tv.tv_usec/1000 + (tv.tv_sec & 0xfffff) * 1000);
  return nCount;
}

#endif


static int BMK_GetMilliSpan( int nTimeStart )
{
  int nSpan = BMK_GetMilliStart() - nTimeStart;
  if ( nSpan < 0 )
    nSpan += 0x100000 * 1000;
  return nSpan;
}


static size_t BMK_findMaxMem(U64 requiredMem)
{
    size_t step = 64 MB;
    BYTE* testmem=NULL;

    requiredMem = (((requiredMem >> 26) + 1) << 26);
    requiredMem += 2*step;
    if (requiredMem > MAX_MEM) requiredMem = MAX_MEM;

    while (!testmem)
    {
        if (requiredMem > step) requiredMem -= step;
        else requiredMem >>= 1;
        testmem = (BYTE*) malloc ((size_t)requiredMem);
    }
    free (testmem);

    /* keep some space available */
    if (requiredMem > step) requiredMem -= step;
    else requiredMem >>= 1;

    return (size_t)requiredMem;
}


static U64 BMK_GetFileSize(const char* infilename)
{
    int r;
#if defined(_MSC_VER)
    struct _stat64 statbuf;
    r = _stat64(infilename, &statbuf);
#else
    struct stat statbuf;
    r = stat(infilename, &statbuf);
#endif
    if (r || !S_ISREG(statbuf.st_mode)) return 0;   /* No good... */
    return (U64)statbuf.st_size;
}


/*********************************************************
*  Public function
**********************************************************/

int BMK_benchFiles(const char** fileNamesTable, int nbFiles, int cLevel)
{
  int fileIdx=0;
  char* orig_buff;
  struct compressionParameters compP;
  int cfunctionId;

  U64 totals = 0;
  U64 totalz = 0;
  double totalc = 0.;
  double totald = 0.;


  /* Init */
  if (cLevel <= 3) cfunctionId = 0; else cfunctionId = 1;
  switch (cfunctionId)
  {
#ifdef COMPRESSOR0
  case 0 : compP.compressionFunction = COMPRESSOR0; break;
#endif
#ifdef COMPRESSOR1
  case 1 : compP.compressionFunction = COMPRESSOR1; break;
#endif
  default : compP.compressionFunction = DEFAULTCOMPRESSOR;
  }
  compP.decompressionFunction = LZ4_decompress_fast;

  /* Loop for each file */
  while (fileIdx<nbFiles)
  {
      FILE*  inFile;
      const char*  inFileName;
      U64    inFileSize;
      size_t benchedSize;
      int nbChunks;
      int maxCompressedChunkSize;
      size_t readSize;
      char* compressedBuffer; int compressedBuffSize;
      struct chunkParameters* chunkP;
      U32 crcOrig;

      /* Check file existence */
      inFileName = fileNamesTable[fileIdx++];
      inFile = fopen( inFileName, "rb" );
      if (inFile==NULL) { DISPLAY( "Pb opening %s\n", inFileName); return 11; }

      /* Memory allocation & restrictions */
      inFileSize = BMK_GetFileSize(inFileName);
      if (inFileSize==0) { DISPLAY( "file is empty\n"); fclose(inFile); return 11; }
      benchedSize = (size_t) BMK_findMaxMem(inFileSize * 2) / 2;
      if (benchedSize==0) { DISPLAY( "not enough memory\n"); fclose(inFile); return 11; }
      if ((U64)benchedSize > inFileSize) benchedSize = (size_t)inFileSize;
      if (benchedSize < inFileSize)
      {
          DISPLAY("Not enough memory for '%s' full size; testing %i MB only...\n", inFileName, (int)(benchedSize>>20));
      }

      /* Alloc */
      chunkP = (struct chunkParameters*) malloc(((benchedSize / (size_t)chunkSize)+1) * sizeof(struct chunkParameters));
      orig_buff = (char*)malloc((size_t)benchedSize);
      nbChunks = (int) ((int)benchedSize / chunkSize) + 1;
      maxCompressedChunkSize = LZ4_compressBound(chunkSize);
      compressedBuffSize = nbChunks * maxCompressedChunkSize;
      compressedBuffer = (char*)malloc((size_t)compressedBuffSize);

      if (!orig_buff || !compressedBuffer)
      {
        DISPLAY("\nError: not enough memory!\n");
        free(orig_buff);
        free(compressedBuffer);
        free(chunkP);
        fclose(inFile);
        return 12;
      }

      /* Init chunks data */
      {
          int i;
          size_t remaining = benchedSize;
          char* in = orig_buff;
          char* out = compressedBuffer;
          for (i=0; i<nbChunks; i++)
          {
              chunkP[i].id = i;
              chunkP[i].origBuffer = in; in += chunkSize;
              if ((int)remaining > chunkSize) { chunkP[i].origSize = chunkSize; remaining -= chunkSize; } else { chunkP[i].origSize = (int)remaining; remaining = 0; }
              chunkP[i].compressedBuffer = out; out += maxCompressedChunkSize;
              chunkP[i].compressedSize = 0;
          }
      }

      /* Fill input buffer */
      DISPLAY("Loading %s...       \r", inFileName);
      readSize = fread(orig_buff, 1, benchedSize, inFile);
      fclose(inFile);

      if (readSize != benchedSize)
      {
        DISPLAY("\nError: problem reading file '%s' !!    \n", inFileName);
        free(orig_buff);
        free(compressedBuffer);
        free(chunkP);
        return 13;
      }

      /* Calculating input Checksum */
      crcOrig = XXH32(orig_buff, (unsigned int)benchedSize,0);


      /* Bench */
      {
        int loopNb, chunkNb;
        size_t cSize=0;
        double fastestC = 100000000., fastestD = 100000000.;
        double ratio=0.;
        U32 crcCheck=0;

        DISPLAY("\r%79s\r", "");
        for (loopNb = 1; loopNb <= nbIterations; loopNb++)
        {
          int nbLoops;
          int milliTime;

          /* Compression */
          DISPLAY("%1i-%-14.14s : %9i ->\r", loopNb, inFileName, (int)benchedSize);
          { size_t i; for (i=0; i<benchedSize; i++) compressedBuffer[i]=(char)i; }     /* warmimg up memory */

          nbLoops = 0;
          milliTime = BMK_GetMilliStart();
          while(BMK_GetMilliStart() == milliTime);
          milliTime = BMK_GetMilliStart();
          while(BMK_GetMilliSpan(milliTime) < TIMELOOP)
          {
            for (chunkNb=0; chunkNb<nbChunks; chunkNb++)
                chunkP[chunkNb].compressedSize = compP.compressionFunction(chunkP[chunkNb].origBuffer, chunkP[chunkNb].compressedBuffer, chunkP[chunkNb].origSize, maxCompressedChunkSize, cLevel);
            nbLoops++;
          }
          milliTime = BMK_GetMilliSpan(milliTime);

          nbLoops += !nbLoops;   /* avoid division by zero */
          if ((double)milliTime < fastestC*nbLoops) fastestC = (double)milliTime/nbLoops;
          cSize=0; for (chunkNb=0; chunkNb<nbChunks; chunkNb++) cSize += chunkP[chunkNb].compressedSize;
          ratio = (double)cSize/(double)benchedSize*100.;

          DISPLAY("%1i-%-14.14s : %9i -> %9i (%5.2f%%),%7.1f MB/s\r", loopNb, inFileName, (int)benchedSize, (int)cSize, ratio, (double)benchedSize / fastestC / 1000.);

          /* Decompression */
          { size_t i; for (i=0; i<benchedSize; i++) orig_buff[i]=0; }     /* zeroing area, for CRC checking */

          nbLoops = 0;
          milliTime = BMK_GetMilliStart();
          while(BMK_GetMilliStart() == milliTime);
          milliTime = BMK_GetMilliStart();
          while(BMK_GetMilliSpan(milliTime) < TIMELOOP)
          {
            for (chunkNb=0; chunkNb<nbChunks; chunkNb++)
                chunkP[chunkNb].compressedSize = LZ4_decompress_fast(chunkP[chunkNb].compressedBuffer, chunkP[chunkNb].origBuffer, chunkP[chunkNb].origSize);
            nbLoops++;
          }
          milliTime = BMK_GetMilliSpan(milliTime);

          nbLoops += !nbLoops;   /* avoid division by zero */
          if ((double)milliTime < fastestD*nbLoops) fastestD = (double)milliTime/nbLoops;
          DISPLAY("%1i-%-14.14s : %9i -> %9i (%5.2f%%),%7.1f MB/s ,%7.1f MB/s \r", loopNb, inFileName, (int)benchedSize, (int)cSize, ratio, (double)benchedSize / fastestC / 1000., (double)benchedSize / fastestD / 1000.);

          /* CRC Checking */
          crcCheck = XXH32(orig_buff, (unsigned int)benchedSize,0);
          if (crcOrig!=crcCheck) { DISPLAY("\n!!! WARNING !!! %14s : Invalid Checksum : %x != %x\n", inFileName, (unsigned)crcOrig, (unsigned)crcCheck); break; }
        }

        if (crcOrig==crcCheck)
        {
            if (ratio<100.)
                DISPLAY("%-16.16s : %9i -> %9i (%5.2f%%),%7.1f MB/s ,%7.1f MB/s \n", inFileName, (int)benchedSize, (int)cSize, ratio, (double)benchedSize / fastestC / 1000., (double)benchedSize / fastestD / 1000.);
            else
                DISPLAY("%-16.16s : %9i -> %9i (%5.1f%%),%7.1f MB/s ,%7.1f MB/s  \n", inFileName, (int)benchedSize, (int)cSize, ratio, (double)benchedSize / fastestC / 1000., (double)benchedSize / fastestD / 1000.);
        }
        totals += benchedSize;
        totalz += cSize;
        totalc += fastestC;
        totald += fastestD;
      }

      free(orig_buff);
      free(compressedBuffer);
      free(chunkP);
  }

  if (nbFiles > 1)
        DISPLAY("%-16.16s :%10llu ->%10llu (%5.2f%%), %6.1f MB/s , %6.1f MB/s\n", "  TOTAL", (long long unsigned int)totals, (long long unsigned int)totalz, (double)totalz/(double)totals*100., (double)totals/totalc/1000., (double)totals/totald/1000.);

  if (BMK_pause) { DISPLAY("\npress enter...\n"); (void)getchar(); }

  return 0;
}



