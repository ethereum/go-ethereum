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
/** @file util.h
 * @author Tim Hughes <tim@twistedfury.com>
 * @date 2015
 */
#pragma once
#include <stdint.h>
#include "compiler.h"

#ifdef __cplusplus
extern "C" {
#endif

//#ifdef _MSC_VER
void debugf(char const* str, ...);
//#else
//#define debugf printf
//#endif

static inline uint32_t min_u32(uint32_t a, uint32_t b)
{
	return a < b ? a : b;
}

static inline uint32_t clamp_u32(uint32_t x, uint32_t min_, uint32_t max_)
{
	return x < min_ ? min_ : (x > max_ ? max_ : x);
}

#ifdef __cplusplus
}
#endif
