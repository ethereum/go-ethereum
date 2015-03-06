/*
  This file is part of cpp-ethereum.

  cpp-ethereum is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  cpp-ethereum is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with cpp-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/** @file util.c
 * @author Tim Hughes <tim@twistedfury.com>
 * @date 2015
 */
#include <stdarg.h>
#include <stdio.h>
#include "util.h"

#ifdef _MSC_VER

// foward declare without all of Windows.h
__declspec(dllimport) void __stdcall OutputDebugStringA(const char* lpOutputString);

void debugf(const char *str, ...)
{
	va_list args;
    va_start(args, str);

	char buf[1<<16];
	_vsnprintf_s(buf, sizeof(buf), sizeof(buf), str, args);
	buf[sizeof(buf)-1] = '\0';
	OutputDebugStringA(buf);
}

#endif
