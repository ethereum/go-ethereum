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

/** @file sha3.cpp
* @author Tim Hughes <tim@twistedfury.com>
* @date 2015
*/
#include <stdint.h>
#include <cryptopp/sha3.h>

extern "C" {
struct ethash_blockhash;
typedef struct ethash_blockhash ethash_blockhash_t;
void SHA3_256(ethash_blockhash_t const* ret, const uint8_t *data, size_t size) {
    CryptoPP::SHA3_256().CalculateDigest((uint8_t*)ret, data, size);
}

void SHA3_512(uint8_t *const ret, const uint8_t *data, size_t size) {
  CryptoPP::SHA3_512().CalculateDigest(ret, data, size);
}
}
