/*
  LZ4io.h - LZ4 File/Stream Interface
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
  It is not part of LZ4 compression library, it is a user code of the LZ4 library.
  - The license of LZ4 library is BSD.
  - The license of xxHash library is BSD.
  - The license of this source file is GPLv2.
*/

#pragma once

/* ************************************************** */
/* Special input/output values                        */
/* ************************************************** */
#define NULL_OUTPUT "null"
static char const stdinmark[] = "stdin";
static char const stdoutmark[] = "stdout";
#ifdef _WIN32
static char const nulmark[] = "nul";
#else
static char const nulmark[] = "/dev/null";
#endif


/* ************************************************** */
/* ****************** Functions ********************* */
/* ************************************************** */

int LZ4IO_compressFilename  (const char* input_filename, const char* output_filename, int compressionlevel);
int LZ4IO_decompressFilename(const char* input_filename, const char* output_filename);


int LZ4IO_compressMultipleFilenames(const char** inFileNamesTable, int ifntSize, const char* suffix, int compressionlevel);
int LZ4IO_decompressMultipleFilenames(const char** inFileNamesTable, int ifntSize, const char* suffix);

/* ************************************************** */
/* ****************** Parameters ******************** */
/* ************************************************** */

/* Default setting : overwrite = 1;
   return : overwrite mode (0/1) */
int LZ4IO_setOverwrite(int yes);

/* blockSizeID : valid values : 4-5-6-7
   return : -1 if error, blockSize if OK */
int LZ4IO_setBlockSizeID(int blockSizeID);

/* Default setting : independent blocks */
typedef enum { LZ4IO_blockLinked=0, LZ4IO_blockIndependent} LZ4IO_blockMode_t;
int LZ4IO_setBlockMode(LZ4IO_blockMode_t blockMode);

/* Default setting : no block checksum */
int LZ4IO_setBlockChecksumMode(int xxhash);

/* Default setting : stream checksum enabled */
int LZ4IO_setStreamChecksumMode(int xxhash);

/* Default setting : 0 (no notification) */
int LZ4IO_setNotificationLevel(int level);

/* Default setting : 0 (disabled) */
int LZ4IO_setSparseFile(int enable);

/* Default setting : 0 (disabled) */
int LZ4IO_setContentSize(int enable);
