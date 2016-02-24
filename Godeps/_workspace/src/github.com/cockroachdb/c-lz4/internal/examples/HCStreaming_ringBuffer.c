// LZ4 HC streaming API example : ring buffer
// Based on previous work from Takayuki Matsuoka


/**************************************
 * Compiler Options
 **************************************/
#ifdef _MSC_VER    /* Visual Studio */
#  define _CRT_SECURE_NO_WARNINGS // for MSVC
#  define snprintf sprintf_s
#endif

#define GCC_VERSION (__GNUC__ * 100 + __GNUC_MINOR__)
#ifdef __GNUC__
#  pragma GCC diagnostic ignored "-Wmissing-braces"   /* GCC bug 53119 : doesn't accept { 0 } as initializer (https://gcc.gnu.org/bugzilla/show_bug.cgi?id=53119) */
#endif


/**************************************
 * Includes
 **************************************/
#include "lz4hc.h"
#include "lz4.h"

#include <stdio.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

enum {
    MESSAGE_MAX_BYTES   = 1024,
    RING_BUFFER_BYTES   = 1024 * 8 + MESSAGE_MAX_BYTES,
    DEC_BUFFER_BYTES    = RING_BUFFER_BYTES + MESSAGE_MAX_BYTES   // Intentionally larger to test unsynchronized ring buffers
};


size_t write_int32(FILE* fp, int32_t i) {
    return fwrite(&i, sizeof(i), 1, fp);
}

size_t write_bin(FILE* fp, const void* array, int arrayBytes) {
    return fwrite(array, 1, arrayBytes, fp);
}

size_t read_int32(FILE* fp, int32_t* i) {
    return fread(i, sizeof(*i), 1, fp);
}

size_t read_bin(FILE* fp, void* array, int arrayBytes) {
    return fread(array, 1, arrayBytes, fp);
}


void test_compress(FILE* outFp, FILE* inpFp)
{
    LZ4_streamHC_t lz4Stream_body = { 0 };
    LZ4_streamHC_t* lz4Stream = &lz4Stream_body;

    static char inpBuf[RING_BUFFER_BYTES];
    int inpOffset = 0;

    for(;;)
    {
        // Read random length ([1,MESSAGE_MAX_BYTES]) data to the ring buffer.
        char* const inpPtr = &inpBuf[inpOffset];
        const int randomLength = (rand() % MESSAGE_MAX_BYTES) + 1;
        const int inpBytes = (int) read_bin(inpFp, inpPtr, randomLength);
        if (0 == inpBytes) break;

        {
            char cmpBuf[LZ4_COMPRESSBOUND(MESSAGE_MAX_BYTES)];
            const int cmpBytes = LZ4_compressHC_continue(lz4Stream, inpPtr, cmpBuf, inpBytes);

            if(cmpBytes <= 0) break;
            write_int32(outFp, cmpBytes);
            write_bin(outFp, cmpBuf, cmpBytes);

            inpOffset += inpBytes;

            // Wraparound the ringbuffer offset
            if(inpOffset >= RING_BUFFER_BYTES - MESSAGE_MAX_BYTES)
                inpOffset = 0;
        }
    }

    write_int32(outFp, 0);
}


void test_decompress(FILE* outFp, FILE* inpFp)
{
    static char decBuf[DEC_BUFFER_BYTES];
    int decOffset = 0;
    LZ4_streamDecode_t lz4StreamDecode_body = { 0 };
    LZ4_streamDecode_t* lz4StreamDecode = &lz4StreamDecode_body;

    for(;;)
    {
        int  cmpBytes = 0;
        char cmpBuf[LZ4_COMPRESSBOUND(MESSAGE_MAX_BYTES)];

        {
            const size_t r0 = read_int32(inpFp, &cmpBytes);
            size_t r1;
            if(r0 != 1 || cmpBytes <= 0)
                break;

            r1 = read_bin(inpFp, cmpBuf, cmpBytes);
            if(r1 != (size_t) cmpBytes)
                break;
        }

        {
            char* const decPtr = &decBuf[decOffset];
            const int decBytes = LZ4_decompress_safe_continue(
                lz4StreamDecode, cmpBuf, decPtr, cmpBytes, MESSAGE_MAX_BYTES);
            if(decBytes <= 0)
                break;

            decOffset += decBytes;
            write_bin(outFp, decPtr, decBytes);

            // Wraparound the ringbuffer offset
            if(decOffset >= DEC_BUFFER_BYTES - MESSAGE_MAX_BYTES)
                decOffset = 0;
        }
    }
}


// Compare 2 files content
// return 0 if identical
// return ByteNb>0 if different
size_t compare(FILE* f0, FILE* f1)
{
    size_t result = 1;

    for (;;)
    {
        char b0[65536];
        char b1[65536];
        const size_t r0 = fread(b0, 1, sizeof(b0), f0);
        const size_t r1 = fread(b1, 1, sizeof(b1), f1);

        if ((r0==0) && (r1==0)) return 0;   // success

        if (r0 != r1)
        {
            size_t smallest = r0;
            if (r1<r0) smallest = r1;
            result += smallest;
            break;
        }

        if (memcmp(b0, b1, r0))
        {
            unsigned errorPos = 0;
            while ((errorPos < r0) && (b0[errorPos]==b1[errorPos])) errorPos++;
            result += errorPos;
            break;
        }

        result += sizeof(b0);
    }

    return result;
}


int main(int argc, char** argv)
{
    char inpFilename[256] = { 0 };
    char lz4Filename[256] = { 0 };
    char decFilename[256] = { 0 };
    unsigned fileID = 1;
    unsigned pause = 0;


    if(argc < 2) {
        printf("Please specify input filename\n");
        return 0;
    }

    if (!strcmp(argv[1], "-p")) pause = 1, fileID = 2;

    snprintf(inpFilename, 256, "%s", argv[fileID]);
    snprintf(lz4Filename, 256, "%s.lz4s-%d", argv[fileID], 9);
    snprintf(decFilename, 256, "%s.lz4s-%d.dec", argv[fileID], 9);

    printf("input   = [%s]\n", inpFilename);
    printf("lz4     = [%s]\n", lz4Filename);
    printf("decoded = [%s]\n", decFilename);

    // compress
    {
        FILE* inpFp = fopen(inpFilename, "rb");
        FILE* outFp = fopen(lz4Filename, "wb");

        test_compress(outFp, inpFp);

        fclose(outFp);
        fclose(inpFp);
    }

    // decompress
    {
        FILE* inpFp = fopen(lz4Filename, "rb");
        FILE* outFp = fopen(decFilename, "wb");

        test_decompress(outFp, inpFp);

        fclose(outFp);
        fclose(inpFp);
    }

    // verify
    {
        FILE* inpFp = fopen(inpFilename, "rb");
        FILE* decFp = fopen(decFilename, "rb");

        const size_t cmp = compare(inpFp, decFp);
        if(0 == cmp) {
            printf("Verify : OK\n");
        } else {
            printf("Verify : NG : error at pos %u\n", (unsigned)cmp-1);
        }

        fclose(decFp);
        fclose(inpFp);
    }

    if (pause)
    {
        printf("Press enter to continue ...\n");
        getchar();
    }

    return 0;
}
