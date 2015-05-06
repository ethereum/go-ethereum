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

enum ethash_io_rc ethash_io_prepare(
	char const* dirname,
	ethash_h256_t const seedhash,
	FILE** output_file,
	uint64_t file_size,
	bool force_create
)
{
	char mutable_name[DAG_MUTABLE_NAME_MAX_SIZE];
	enum ethash_io_rc ret = ETHASH_IO_FAIL;

	// assert directory exists
	if (!ethash_mkdir(dirname)) {
		goto end;
	}

	ethash_io_mutable_name(ETHASH_REVISION, &seedhash, mutable_name);
	char* tmpfile = ethash_io_create_filename(dirname, mutable_name, strlen(mutable_name));
	if (!tmpfile) {
		goto end;
	}

	FILE *f;
	if (!force_create) {
		// try to open the file
		f = ethash_fopen(tmpfile, "rb+");
		if (f) {
			size_t found_size;
			if (!ethash_file_size(f, &found_size)) {
				fclose(f);
				goto free_memo;
			}
			if (file_size != found_size - ETHASH_DAG_MAGIC_NUM_SIZE) {
				fclose(f);
				ret = ETHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}
			// compare the magic number, no need to care about endianess since it's local
			uint64_t magic_num;
			if (fread(&magic_num, ETHASH_DAG_MAGIC_NUM_SIZE, 1, f) != 1) {
				// I/O error
				fclose(f);
				ret = ETHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}
			if (magic_num != ETHASH_DAG_MAGIC_NUM) {
				fclose(f);
				ret = ETHASH_IO_MEMO_SIZE_MISMATCH;
				goto free_memo;
			}
			ret = ETHASH_IO_MEMO_MATCH;
			goto set_file;
		}
	}
	
	// file does not exist, will need to be created
	f = ethash_fopen(tmpfile, "wb+");
	if (!f) {
		goto free_memo;
	}
	// make sure it's of the proper size
	if (fseek(f, (long int)(file_size + ETHASH_DAG_MAGIC_NUM_SIZE - 1), SEEK_SET) != 0) {
		fclose(f);
		goto free_memo;
	}
	fputc('\n', f);
	fflush(f);
	ret = ETHASH_IO_MEMO_MISMATCH;
	goto set_file;

	ret = ETHASH_IO_MEMO_MATCH;
set_file:
	*output_file = f;
free_memo:
	free(tmpfile);
end:
	return ret;
}
