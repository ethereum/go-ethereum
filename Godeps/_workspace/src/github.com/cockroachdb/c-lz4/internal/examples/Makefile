# ##########################################################################
# LZ4 examples - Makefile
# Copyright (C) Yann Collet 2011-2014
#
# GPL v2 License
#
# This program is free software; you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation; either version 2 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License along
# with this program; if not, write to the Free Software Foundation, Inc.,
# 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.
#
# You can contact the author at :
#  - LZ4 source repository : http://code.google.com/p/lz4/
#  - LZ4 forum froup : https://groups.google.com/forum/#!forum/lz4c
# ##########################################################################
# This makefile compile and test
# example programs, using (mostly) LZ4 streaming library,
# kindly provided by Takayuki Matsuoka
# ##########################################################################

CFLAGS ?= -O3
CFLAGS += -std=c99 -Wall -Wextra -Wundef -Wshadow -Wcast-align -Wstrict-prototypes
FLAGS  := -I../lib $(CPPFLAGS) $(CFLAGS) $(LDFLAGS)

TESTFILE= Makefile
LZ4DIR := ../lib
LZ4 = ../programs/lz4


# Define *.exe as extension for Windows systems
ifneq (,$(filter Windows%,$(OS)))
EXT =.exe
VOID = nul
else
EXT =
VOID = /dev/null
endif


default: all

all: printVersion doubleBuffer ringBuffer ringBufferHC lineCompress frameCompress

printVersion: $(LZ4DIR)/lz4.c printVersion.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT)

doubleBuffer: $(LZ4DIR)/lz4.c blockStreaming_doubleBuffer.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT)

ringBuffer  : $(LZ4DIR)/lz4.c blockStreaming_ringBuffer.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT)

ringBufferHC: $(LZ4DIR)/lz4.c $(LZ4DIR)/lz4hc.c HCStreaming_ringBuffer.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT)

lineCompress: $(LZ4DIR)/lz4.c blockStreaming_lineByLine.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT)

frameCompress: frameCompress.c
	$(CC)      $(FLAGS) $^ -o $@$(EXT) -L$(LZ4DIR) -llz4

test : all
	./printVersion$(EXT)
	./doubleBuffer$(EXT) $(TESTFILE)
	./ringBuffer$(EXT)   $(TESTFILE)
	./ringBufferHC$(EXT) $(TESTFILE)
	./lineCompress$(EXT) $(TESTFILE)
	LD_LIBRARY_PATH=$(LZ4DIR) ./frameCompress$(EXT) $(TESTFILE)
	$(LZ4) -vt $(TESTFILE).lz4

clean:
	@rm -f core *.o *.dec *-0 *-9 *-8192 *.lz4s *.lz4 \
	 printVersion$(EXT) doubleBuffer$(EXT) ringBuffer$(EXT) ringBufferHC$(EXT) \
	 lineCompress$(EXT) frameCompress$(EXT)
	@echo Cleaning completed

