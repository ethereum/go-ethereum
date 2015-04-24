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
	size_t file_size
)
{
	char mutable_name[DAG_MUTABLE_NAME_MAX_SIZE];
	enum ethash_io_rc ret = ETHASH_IO_FAIL;

	// assert directory exists
	if (!ethash_mkdir(dirname)) {
		goto end;
	}

	ethash_io_mutable_name(REVISION, &seedhash, mutable_name);
	char* tmpfile = ethash_io_create_filename(dirname, mutable_name, strlen(mutable_name));
	if (!tmpfile) {
		goto end;
	}

	// try to open the file
	FILE* f = ethash_fopen(tmpfile, "rb+");
	if (f) {
		size_t found_size;
		if (!ethash_file_size(f, &found_size)) {
			fclose(f);
			goto free_memo;
		}
		if (file_size != found_size) {
			fclose(f);
			ret = ETHASH_IO_MEMO_SIZE_MISMATCH;
			goto free_memo;
		}
	} else {
		// file does not exist, will need to be created
		f = ethash_fopen(tmpfile, "wb+");
		if (!f) {
			goto free_memo;
		}
		// make sure it's of the proper size
		if (fseek(f, file_size - 1, SEEK_SET) != 0) {
			fclose(f);
			goto free_memo;
		}
		fputc('\n', f);
		fflush(f);
		ret = ETHASH_IO_MEMO_MISMATCH;
		goto set_file;
	}

	ret = ETHASH_IO_MEMO_MATCH;
set_file:
	*output_file = f;
free_memo:
	free(tmpfile);
end:
	return ret;
}
