/*
  This file is part of ubqhash.

  ubqhash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  ubqhash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with ubqhash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file sha3.cpp
* @author Tim Hughes <tim@twistedfury.com>
* @date 2015
*/
#include <stdint.h>
#include <cryptopp/blake2.h>

extern "C" {
struct ubqhash_h256;
typedef struct ubqhash_h256 ubqhash_h256_t;
void BLAKE2B_512(uint8_t* const ret, uint8_t const* data, size_t size)
{
	CryptoPP::BLAKE2b().CalculateDigest(ret, data, size);
}
}
