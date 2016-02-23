/*
  LZ4cli - LZ4 Command Line Interface
  Copyright (C) Yann Collet 2011-2015

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
/*
  Note : this is stand-alone program.
  It is not part of LZ4 compression library, it is a user program of the LZ4 library.
  The license of LZ4 library is BSD.
  The license of xxHash library is BSD.
  The license of this compression CLI program is GPLv2.
*/

/**************************************
*  Tuning parameters
***************************************/
/* ENABLE_LZ4C_LEGACY_OPTIONS :
   Control the availability of -c0, -c1 and -hc legacy arguments
   Default : Legacy options are disabled */
/* #define ENABLE_LZ4C_LEGACY_OPTIONS */


/**************************************
*  Compiler Options
***************************************/
/* Disable some Visual warning messages */
#ifdef _MSC_VER
#  define _CRT_SECURE_NO_WARNINGS
#  define _CRT_SECURE_NO_DEPRECATE     /* VS2005 */
#  pragma warning(disable : 4127)      /* disable: C4127: conditional expression is constant */
#endif

#define _POSIX_SOURCE 1        /* for fileno() within <stdio.h> on unix */


/****************************
*  Includes
*****************************/
#include <stdio.h>    /* fprintf, getchar */
#include <stdlib.h>   /* exit, calloc, free */
#include <string.h>   /* strcmp, strlen */
#include "bench.h"    /* BMK_benchFile, BMK_SetNbIterations, BMK_SetBlocksize, BMK_SetPause */
#include "lz4io.h"    /* LZ4IO_compressFilename, LZ4IO_decompressFilename, LZ4IO_compressMultipleFilenames */


/****************************
*  OS-specific Includes
*****************************/
#if defined(MSDOS) || defined(OS2) || defined(WIN32) || defined(_WIN32)
#  include <io.h>       /* _isatty */
#  if defined(__DJGPP__)
#    include <unistd.h>
#    define _isatty isatty
#    define _fileno fileno
#  endif
#  ifdef __MINGW32__
   int _fileno(FILE *stream);   /* MINGW somehow forgets to include this prototype into <stdio.h> */
#  endif
#  define IS_CONSOLE(stdStream) _isatty(_fileno(stdStream))
#else
#  include <unistd.h>   /* isatty */
#  define IS_CONSOLE(stdStream) isatty(fileno(stdStream))
#endif


/*****************************
*  Constants
******************************/
#define COMPRESSOR_NAME "LZ4 command line interface"
#ifndef LZ4_VERSION
#  define LZ4_VERSION "r128"
#endif
#define AUTHOR "Yann Collet"
#define WELCOME_MESSAGE "*** %s %i-bits %s, by %s (%s) ***\n", COMPRESSOR_NAME, (int)(sizeof(void*)*8), LZ4_VERSION, AUTHOR, __DATE__
#define LZ4_EXTENSION ".lz4"
#define LZ4CAT "lz4cat"
#define UNLZ4 "unlz4"

#define KB *(1U<<10)
#define MB *(1U<<20)
#define GB *(1U<<30)

#define LZ4_BLOCKSIZEID_DEFAULT 7


/**************************************
*  Macros
***************************************/
#define DISPLAY(...)           fprintf(stderr, __VA_ARGS__)
#define DISPLAYLEVEL(l, ...)   if (displayLevel>=l) { DISPLAY(__VA_ARGS__); }
static unsigned displayLevel = 2;   /* 0 : no display ; 1: errors only ; 2 : downgradable normal ; 3 : non-downgradable normal; 4 : + information */


/**************************************
*  Local Variables
***************************************/
static char* programName;


/**************************************
*  Exceptions
***************************************/
#define DEBUG 0
#define DEBUGOUTPUT(...) if (DEBUG) DISPLAY(__VA_ARGS__);
#define EXM_THROW(error, ...)                                             \
{                                                                         \
    DEBUGOUTPUT("Error defined at %s, line %i : \n", __FILE__, __LINE__); \
    DISPLAYLEVEL(1, "Error %i : ", error);                                \
    DISPLAYLEVEL(1, __VA_ARGS__);                                         \
    DISPLAYLEVEL(1, "\n");                                                \
    exit(error);                                                          \
}


/**************************************
*  Version modifiers
***************************************/
#define EXTENDED_ARGUMENTS
#define EXTENDED_HELP
#define EXTENDED_FORMAT
#define DEFAULT_COMPRESSOR   LZ4IO_compressFilename
#define DEFAULT_DECOMPRESSOR LZ4IO_decompressFilename
int LZ4IO_compressFilename_Legacy(const char* input_filename, const char* output_filename, int compressionlevel);   /* hidden function */


/*****************************
*  Functions
*****************************/
static int usage(void)
{
    DISPLAY( "Usage :\n");
    DISPLAY( "      %s [arg] [input] [output]\n", programName);
    DISPLAY( "\n");
    DISPLAY( "input   : a filename\n");
    DISPLAY( "          with no FILE, or when FILE is - or %s, read standard input\n", stdinmark);
    DISPLAY( "Arguments :\n");
    DISPLAY( " -1     : Fast compression (default) \n");
    DISPLAY( " -9     : High compression \n");
    DISPLAY( " -d     : decompression (default for %s extension)\n", LZ4_EXTENSION);
    DISPLAY( " -z     : force compression\n");
    DISPLAY( " -f     : overwrite output without prompting \n");
    DISPLAY( " -h/-H  : display help/long help and exit\n");
    return 0;
}

static int usage_advanced(void)
{
    DISPLAY(WELCOME_MESSAGE);
    usage();
    DISPLAY( "\n");
    DISPLAY( "Advanced arguments :\n");
    DISPLAY( " -V     : display Version number and exit\n");
    DISPLAY( " -v     : verbose mode\n");
    DISPLAY( " -q     : suppress warnings; specify twice to suppress errors too\n");
    DISPLAY( " -c     : force write to standard output, even if it is the console\n");
    DISPLAY( " -t     : test compressed file integrity\n");
    DISPLAY( " -m     : multiple input files (implies automatic output filenames)\n");
    DISPLAY( " -l     : compress using Legacy format (Linux kernel compression)\n");
    DISPLAY( " -B#    : Block size [4-7](default : 7)\n");
    DISPLAY( " -BD    : Block dependency (improve compression ratio)\n");
    /* DISPLAY( " -BX    : enable block checksum (default:disabled)\n");   *//* Option currently inactive */
    DISPLAY( "--no-frame-crc : disable stream checksum (default:enabled)\n");
    DISPLAY( "--content-size : compressed frame includes original size (default:not present)\n");
    DISPLAY( "--[no-]sparse  : sparse mode (default:enabled on file, disabled on stdout)\n");
    DISPLAY( "Benchmark arguments :\n");
    DISPLAY( " -b     : benchmark file(s)\n");
    DISPLAY( " -i#    : iteration loops [1-9](default : 3), benchmark mode only\n");
#if defined(ENABLE_LZ4C_LEGACY_OPTIONS)
    DISPLAY( "Legacy arguments :\n");
    DISPLAY( " -c0    : fast compression\n");
    DISPLAY( " -c1    : high compression\n");
    DISPLAY( " -hc    : high compression\n");
    DISPLAY( " -y     : overwrite output without prompting \n");
#endif /* ENABLE_LZ4C_LEGACY_OPTIONS */
    EXTENDED_HELP;
    return 0;
}

static int usage_longhelp(void)
{
    usage_advanced();
    DISPLAY( "\n");
    DISPLAY( "Which values can get [output] ? \n");
    DISPLAY( "[output] : a filename\n");
    DISPLAY( "          '%s', or '-' for standard output (pipe mode)\n", stdoutmark);
    DISPLAY( "          '%s' to discard output (test mode)\n", NULL_OUTPUT);
    DISPLAY( "[output] can be left empty. In this case, it receives the following value : \n");
    DISPLAY( "          - if stdout is not the console, then [output] = stdout \n");
    DISPLAY( "          - if stdout is console : \n");
    DISPLAY( "               + if compression selected, output to filename%s \n", LZ4_EXTENSION);
    DISPLAY( "               + if decompression selected, output to filename without '%s'\n", LZ4_EXTENSION);
    DISPLAY( "                    > if input filename has no '%s' extension : error\n", LZ4_EXTENSION);
    DISPLAY( "\n");
    DISPLAY( "Compression levels : \n");
    DISPLAY( "There are technically 2 accessible compression levels.\n");
    DISPLAY( "-0 ... -2 => Fast compression\n");
    DISPLAY( "-3 ... -9 => High compression\n");
    DISPLAY( "\n");
    DISPLAY( "stdin, stdout and the console : \n");
    DISPLAY( "To protect the console from binary flooding (bad argument mistake)\n");
    DISPLAY( "%s will refuse to read from console, or write to console \n", programName);
    DISPLAY( "except if '-c' command is specified, to force output to console \n");
    DISPLAY( "\n");
    DISPLAY( "Simple example :\n");
    DISPLAY( "1 : compress 'filename' fast, using default output name 'filename.lz4'\n");
    DISPLAY( "          %s filename\n", programName);
    DISPLAY( "\n");
    DISPLAY( "Arguments can be appended together, or provided independently. For example :\n");
    DISPLAY( "2 : compress 'filename' in high compression mode, overwrite output if exists\n");
    DISPLAY( "          %s -f9 filename \n", programName);
    DISPLAY( "    is equivalent to :\n");
    DISPLAY( "          %s -f -9 filename \n", programName);
    DISPLAY( "\n");
    DISPLAY( "%s can be used in 'pure pipe mode', for example :\n", programName);
    DISPLAY( "3 : compress data stream from 'generator', send result to 'consumer'\n");
    DISPLAY( "          generator | %s | consumer \n", programName);
#if defined(ENABLE_LZ4C_LEGACY_OPTIONS)
    DISPLAY( "\n");
    DISPLAY( "Warning :\n");
    DISPLAY( "Legacy arguments take precedence. Therefore : \n");
    DISPLAY( "          %s -hc filename\n", programName);
    DISPLAY( "means 'compress filename in high compression mode'\n");
    DISPLAY( "It is not equivalent to :\n");
    DISPLAY( "          %s -h -c filename\n", programName);
    DISPLAY( "which would display help text and exit\n");
#endif /* ENABLE_LZ4C_LEGACY_OPTIONS */
    return 0;
}

static int badusage(void)
{
    DISPLAYLEVEL(1, "Incorrect parameters\n");
    if (displayLevel >= 1) usage();
    exit(1);
}


static void waitEnter(void)
{
    DISPLAY("Press enter to continue...\n");
    (void)getchar();
}


int main(int argc, char** argv)
{
    int i,
        cLevel=0,
        decode=0,
        bench=0,
        legacy_format=0,
        forceStdout=0,
        forceCompress=0,
        main_pause=0,
        multiple_inputs=0,
        operationResult=0;
    const char* input_filename=0;
    const char* output_filename=0;
    char* dynNameSpace=0;
    const char** inFileNames = NULL;
    unsigned ifnIdx=0;
    char nullOutput[] = NULL_OUTPUT;
    char extension[] = LZ4_EXTENSION;
    int  blockSize;

    /* Init */
    programName = argv[0];
    LZ4IO_setOverwrite(0);
    blockSize = LZ4IO_setBlockSizeID(LZ4_BLOCKSIZEID_DEFAULT);

    /* lz4cat predefined behavior */
    if (!strcmp(programName, LZ4CAT)) { decode=1; forceStdout=1; output_filename=stdoutmark; displayLevel=1; }
    if (!strcmp(programName, UNLZ4)) { decode=1; }

    /* command switches */
    for(i=1; i<argc; i++)
    {
        char* argument = argv[i];

        if(!argument) continue;   /* Protection if argument empty */

        /* long commands (--long-word) */
        if (!strcmp(argument, "--compress")) { forceCompress = 1; continue; }
        if ((!strcmp(argument, "--decompress"))
         || (!strcmp(argument, "--uncompress"))) { decode = 1; continue; }
        if (!strcmp(argument, "--multiple")) { multiple_inputs = 1; if (inFileNames==NULL) inFileNames = (const char**)malloc(argc * sizeof(char*)); continue; }
        if (!strcmp(argument, "--test")) { decode = 1; LZ4IO_setOverwrite(1); output_filename=nulmark; continue; }
        if (!strcmp(argument, "--force")) { LZ4IO_setOverwrite(1); continue; }
        if (!strcmp(argument, "--no-force")) { LZ4IO_setOverwrite(0); continue; }
        if ((!strcmp(argument, "--stdout"))
         || (!strcmp(argument, "--to-stdout"))) { forceStdout=1; output_filename=stdoutmark; displayLevel=1; continue; }
        if (!strcmp(argument, "--frame-crc")) { LZ4IO_setStreamChecksumMode(1); continue; }
        if (!strcmp(argument, "--no-frame-crc")) { LZ4IO_setStreamChecksumMode(0); continue; }
        if (!strcmp(argument, "--content-size")) { LZ4IO_setContentSize(1); continue; }
        if (!strcmp(argument, "--no-content-size")) { LZ4IO_setContentSize(0); continue; }
        if (!strcmp(argument, "--sparse")) { LZ4IO_setSparseFile(2); continue; }
        if (!strcmp(argument, "--no-sparse")) { LZ4IO_setSparseFile(0); continue; }
        if (!strcmp(argument, "--verbose")) { displayLevel=4; continue; }
        if (!strcmp(argument, "--quiet")) { if (displayLevel) displayLevel--; continue; }
        if (!strcmp(argument, "--version")) { DISPLAY(WELCOME_MESSAGE); return 0; }
        if (!strcmp(argument, "--keep")) { continue; }   /* keep source file (default anyway; just for xz/lzma compatibility) */


        /* Short commands (note : aggregated short commands are allowed) */
        if (argument[0]=='-')
        {
            /* '-' means stdin/stdout */
            if (argument[1]==0)
            {
                if (!input_filename) input_filename=stdinmark;
                else output_filename=stdoutmark;
            }

            while (argument[1]!=0)
            {
                argument ++;

#if defined(ENABLE_LZ4C_LEGACY_OPTIONS)
                /* Legacy arguments (-c0, -c1, -hc, -y, -s) */
                if ((argument[0]=='c') && (argument[1]=='0')) { cLevel=0; argument++; continue; }  /* -c0 (fast compression) */
                if ((argument[0]=='c') && (argument[1]=='1')) { cLevel=9; argument++; continue; }  /* -c1 (high compression) */
                if ((argument[0]=='h') && (argument[1]=='c')) { cLevel=9; argument++; continue; }  /* -hc (high compression) */
                if (*argument=='y') { LZ4IO_setOverwrite(1); continue; }                           /* -y (answer 'yes' to overwrite permission) */
#endif /* ENABLE_LZ4C_LEGACY_OPTIONS */

                if ((*argument>='0') && (*argument<='9'))
                {
                    cLevel = 0;
                    while ((*argument >= '0') && (*argument <= '9'))
                    {
                        cLevel *= 10;
                        cLevel += *argument - '0';
                        argument++;
                    }
                    argument--;
                    continue;
                }

                switch(argument[0])
                {
                    /* Display help */
                case 'V': DISPLAY(WELCOME_MESSAGE); goto _cleanup;   /* Version */
                case 'h': usage_advanced(); goto _cleanup;
                case 'H': usage_longhelp(); goto _cleanup;

                    /* Compression (default) */
                case 'z': forceCompress = 1; break;

                    /* Use Legacy format (ex : Linux kernel compression) */
                case 'l': legacy_format = 1; blockSize = 8 MB; break;

                    /* Decoding */
                case 'd': decode=1; break;

                    /* Force stdout, even if stdout==console */
                case 'c': forceStdout=1; output_filename=stdoutmark; displayLevel=1; break;

                    /* Test integrity */
                case 't': decode=1; LZ4IO_setOverwrite(1); output_filename=nulmark; break;

                    /* Overwrite */
                case 'f': LZ4IO_setOverwrite(1); break;

                    /* Verbose mode */
                case 'v': displayLevel=4; break;

                    /* Quiet mode */
                case 'q': if (displayLevel) displayLevel--; break;

                    /* keep source file (default anyway, so useless) (for xz/lzma compatibility) */
                case 'k': break;

                    /* Modify Block Properties */
                case 'B':
                    while (argument[1]!=0)
                    {
                        int exitBlockProperties=0;
                        switch(argument[1])
                        {
                        case '4':
                        case '5':
                        case '6':
                        case '7':
                        {
                            int B = argument[1] - '0';
                            blockSize = LZ4IO_setBlockSizeID(B);
                            BMK_setBlocksize(blockSize);
                            argument++;
                            break;
                        }
                        case 'D': LZ4IO_setBlockMode(LZ4IO_blockLinked); argument++; break;
                        case 'X': LZ4IO_setBlockChecksumMode(1); argument ++; break;   /* currently disabled */
                        default : exitBlockProperties=1;
                        }
                        if (exitBlockProperties) break;
                    }
                    break;

                    /* Benchmark */
                case 'b': bench=1; multiple_inputs=1;
                    if (inFileNames == NULL)
                        inFileNames = (const char**) malloc(argc * sizeof(char*));
                    break;

                    /* Treat non-option args as input files.  See https://code.google.com/p/lz4/issues/detail?id=151 */
                case 'm': multiple_inputs=1;
                    if (inFileNames == NULL)
                        inFileNames = (const char**) malloc(argc * sizeof(char*));
                    break;

                    /* Modify Nb Iterations (benchmark only) */
                case 'i':
                    {
                        unsigned iters = 0;
                        while ((argument[1] >='0') && (argument[1] <='9'))
                        {
                            iters *= 10;
                            iters += argument[1] - '0';
                            argument++;
                        }
                        BMK_setNbIterations(iters);
                    }
                    break;

                    /* Pause at the end (hidden option) */
                case 'p': main_pause=1; BMK_setPause(); break;

                    /* Specific commands for customized versions */
                EXTENDED_ARGUMENTS;

                    /* Unrecognised command */
                default : badusage();
                }
            }
            continue;
        }

        /* Store in *inFileNames[] if -m is used. */
        if (multiple_inputs) { inFileNames[ifnIdx++]=argument; continue; }

        /* Store first non-option arg in input_filename to preserve original cli logic. */
        if (!input_filename) { input_filename=argument; continue; }

        /* Second non-option arg in output_filename to preserve original cli logic. */
        if (!output_filename)
        {
            output_filename=argument;
            if (!strcmp (output_filename, nullOutput)) output_filename = nulmark;
            continue;
        }

        /* 3rd non-option arg should not exist */
        DISPLAYLEVEL(1, "Warning : %s won't be used ! Do you want multiple input files (-m) ? \n", argument);
    }

    DISPLAYLEVEL(3, WELCOME_MESSAGE);
    if (!decode) DISPLAYLEVEL(4, "Blocks size : %i KB\n", blockSize>>10);

    /* No input filename ==> use stdin */
    if (multiple_inputs) input_filename = inFileNames[0], output_filename = (const char*)(inFileNames[0]);
    if(!input_filename) { input_filename=stdinmark; }

    /* Check if input is defined as console; trigger an error in this case */
    if (!strcmp(input_filename, stdinmark) && IS_CONSOLE(stdin) ) badusage();

    /* Check if benchmark is selected */
    if (bench)
    {
        int bmkResult = BMK_benchFiles(inFileNames, ifnIdx, cLevel);
        free((void*)inFileNames);
        return bmkResult;
    }

    /* No output filename ==> try to select one automatically (when possible) */
    while (!output_filename)
    {
        if (!IS_CONSOLE(stdout)) { output_filename=stdoutmark; break; }   /* Default to stdout whenever possible (i.e. not a console) */
        if ((!decode) && !(forceCompress))   /* auto-determine compression or decompression, based on file extension */
        {
            size_t l = strlen(input_filename);
            if (!strcmp(input_filename+(l-4), LZ4_EXTENSION)) decode=1;
        }
        if (!decode)   /* compression to file */
        {
            size_t l = strlen(input_filename);
            dynNameSpace = (char*)calloc(1,l+5);
			if (dynNameSpace==NULL) exit(1);
            strcpy(dynNameSpace, input_filename);
            strcat(dynNameSpace, LZ4_EXTENSION);
            output_filename = dynNameSpace;
            DISPLAYLEVEL(2, "Compressed filename will be : %s \n", output_filename);
            break;
        }
        /* decompression to file (automatic name will work only if input filename has correct format extension) */
        {
            size_t outl;
            size_t inl = strlen(input_filename);
            dynNameSpace = (char*)calloc(1,inl+1);
            strcpy(dynNameSpace, input_filename);
            outl = inl;
            if (inl>4)
                while ((outl >= inl-4) && (input_filename[outl] ==  extension[outl-inl+4])) dynNameSpace[outl--]=0;
            if (outl != inl-5) { DISPLAYLEVEL(1, "Cannot determine an output filename\n"); badusage(); }
            output_filename = dynNameSpace;
            DISPLAYLEVEL(2, "Decoding file %s \n", output_filename);
        }
    }

    /* Check if output is defined as console; trigger an error in this case */
    if (!strcmp(output_filename,stdoutmark) && IS_CONSOLE(stdout) && !forceStdout) badusage();

    /* Downgrade notification level in pure pipe mode (stdin + stdout) and multiple file mode */
    if (!strcmp(input_filename, stdinmark) && !strcmp(output_filename,stdoutmark) && (displayLevel==2)) displayLevel=1;
    if ((multiple_inputs) && (displayLevel==2)) displayLevel=1;


    /* IO Stream/File */
    LZ4IO_setNotificationLevel(displayLevel);
    if (decode)
    {
      if (multiple_inputs)
        operationResult = LZ4IO_decompressMultipleFilenames(inFileNames, ifnIdx, LZ4_EXTENSION);
      else
        DEFAULT_DECOMPRESSOR(input_filename, output_filename);
    }
    else
    {
      /* compression is default action */
      if (legacy_format)
      {
        DISPLAYLEVEL(3, "! Generating compressed LZ4 using Legacy format (deprecated) ! \n");
        LZ4IO_compressFilename_Legacy(input_filename, output_filename, cLevel);
      }
      else
      {
        if (multiple_inputs)
          operationResult = LZ4IO_compressMultipleFilenames(inFileNames, ifnIdx, LZ4_EXTENSION, cLevel);
        else
          DEFAULT_COMPRESSOR(input_filename, output_filename, cLevel);
      }
    }

_cleanup:
    if (main_pause) waitEnter();
    free(dynNameSpace);
    free((void*)inFileNames);
    return operationResult;
}
