/*
  This file is part of ethash.

  ethash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ethash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with ethash.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file io_win32.c
 * @author Lefteris Karapetsas <lefteris@ethdev.com>
 * @date 2015
 */

#include "io.h"
#include <direct.h>
#include <errno.h>
#include <stdio.h>

enum ethash_io_rc ethash_io_prepare(char const *dirname, ethash_blockhash_t seedhash)
{
    char read_buffer[DAG_MEMO_BYTESIZE];
    char expect_buffer[DAG_MEMO_BYTESIZE];
    enum ethash_io_rc ret = ETHASH_IO_FAIL;

    // assert directory exists
    int rc = _mkdir(dirname);
    if (rc == -1 && errno != EEXIST) {
        goto end;
    }

    char *memofile = ethash_io_create_filename(dirname, DAG_MEMO_NAME, sizeof(DAG_MEMO_NAME));
    if (!memofile) {
        goto end;
    }

    // try to open memo file
    FILE *f = fopen(memofile, "rb");
    if (!f) {
        // file does not exist, so no checking happens. All is fine.
        ret = ETHASH_IO_MEMO_MISMATCH;
        goto free_memo;
    }

    if (fread(read_buffer, 1, DAG_MEMO_BYTESIZE, f) != DAG_MEMO_BYTESIZE) {
        goto close;
    }

    ethash_io_serialize_info(REVISION, seedhash, expect_buffer);
    if (memcmp(read_buffer, expect_buffer, DAG_MEMO_BYTESIZE) != 0) {
        // we have different memo contents so delete the memo file
        if (_unlink(memofile) != 0) {
            goto close;
        }
        ret = ETHASH_IO_MEMO_MISMATCH;
    }

    ret = ETHASH_IO_MEMO_MATCH;

close:
    fclose(f);
free_memo:
    free(memofile);
end:
    return ret;
}
