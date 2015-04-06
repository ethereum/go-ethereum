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
/** @file io.c
 * @author Lefteris Karapetsas <lefteris@ethdev.com>
 * @date 2015
 */
#include "io.h"
#include <string.h>
#include <stdio.h>

// silly macro to save some typing
#define PASS_ARR(c_) (c_), sizeof(c_)

static bool ethash_io_write_file(char const *dirname,
                                 char const* filename,
                                 size_t filename_length,
                                 void const* data,
                                 size_t data_size)
{
    bool ret = false;
    char *fullname = ethash_io_create_filename(dirname, filename, filename_length);
    if (!fullname) {
        return false;
    }
    FILE *f = fopen(fullname, "wb");
    if (!f) {
        goto free_name;
    }
    if (data_size != fwrite(data, 1, data_size, f)) {
        goto close;
    }

    ret = true;
close:
    fclose(f);
free_name:
    free(fullname);
    return ret;
}

bool ethash_io_write(char const *dirname,
                     ethash_params const* params,
                     ethash_blockhash_t seedhash,
                     void const* cache,
                     uint8_t **data,
                     uint64_t *data_size)
{
    char info_buffer[DAG_MEMO_BYTESIZE];
    // allocate the bytes
    uint8_t *temp_data_ptr = malloc((size_t)params->full_size);
    if (!temp_data_ptr) {
        goto end;
    }
    ethash_compute_full_data(temp_data_ptr, params, cache);

    if (!ethash_io_write_file(dirname, PASS_ARR(DAG_FILE_NAME), temp_data_ptr, (size_t)params->full_size)) {
        goto fail_free;
    }

    ethash_io_serialize_info(REVISION, seedhash, info_buffer);
    if (!ethash_io_write_file(dirname, PASS_ARR(DAG_MEMO_NAME), info_buffer, DAG_MEMO_BYTESIZE)) {
        goto fail_free;
    }

    *data = temp_data_ptr;
    *data_size = params->full_size;
    return true;

fail_free:
    free(temp_data_ptr);
end:
    return false;
}

#undef PASS_ARR
