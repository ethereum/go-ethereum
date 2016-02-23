# Copyright (c) 2015, Louis P. Santillan <lpsantil@gmail.com>
# All rights reserved.
# See LICENSE for licensing details.
DESTDIR ?= /opt/local

# Pulled the code below from lib/Makefile.  Might be nicer to derive this somehow without sed
# Version numbers
VERSION ?= 129
RELEASE ?= r$(VERSION)
LIBVER_MAJOR=$(shell sed -n '/define LZ4_VERSION_MAJOR/s/.*[[:blank:]]\([0-9][0-9]*\).*/\1/p' < lib/lz4.h)
LIBVER_MINOR=$(shell sed -n '/define LZ4_VERSION_MINOR/s/.*[[:blank:]]\([0-9][0-9]*\).*/\1/p' < lib/lz4.h)
LIBVER_PATCH=$(shell sed -n '/define LZ4_VERSION_RELEASE/s/.*[[:blank:]]\([0-9][0-9]*\).*/\1/p' < lib/lz4.h)
LIBVER=$(LIBVER_MAJOR).$(LIBVER_MINOR).$(LIBVER_PATCH)

######################################################################

CROSS ?= i586-pc-msdosdjgpp
CC = $(CROSS)-gcc
AR = $(CROSS)-ar
LD = $(CROSS)-gcc

CFLAGS ?= -O3 -std=gnu99 -Wall -Wextra -Wundef -Wshadow -Wcast-qual -Wcast-align -Wstrict-prototypes -pedantic -DLZ4_VERSION=\"$(RELEASE)\"
LDFLAGS ?= -s
SRC = programs/bench.c programs/lz4io.c programs/lz4cli.c
OBJ = $(SRC:.c=.o)
SDEPS = $(SRC:.c=.d)
IDIR = lib
EDIR = .
EXE = lz4.exe
LNK = lz4
LDIR = lib
LSRC = lib/lz4.c lib/lz4hc.c lib/lz4frame.c lib/xxhash.c
INC = $(LSRC:.c=.h)
LOBJ = $(LSRC:.c=.o)
LSDEPS = $(LSRC:.c=.d)
LIB = $(LDIR)/lib$(LNK).a

# Since LDFLAGS defaults to "-s", probably better to override unless
# you have a default you would like to maintain
ifeq ($(WITH_DEBUG), 1)
	CFLAGS += -g
	LDFLAGS += -g
endif

# Since LDFLAGS defaults to "-s", probably better to override unless
# you have a default you would like to maintain
ifeq ($(WITH_PROFILING), 1)
	CFLAGS += -pg
	LDFLAGS += -pg
endif

%.o: %.c $(INC) Makefile
	$(CC) $(CFLAGS) -MMD -MP -I$(IDIR) -c $< -o $@

%.exe: %.o $(LIB) Makefile
	$(LD) $< -L$(LDIR) -l$(LNK) $(LDFLAGS) $(LIBDEP) -o $@

######################################################################
######################## DO NOT MODIFY BELOW #########################
######################################################################

.PHONY: all install uninstall showconfig gstat gpush

all: $(LIB) $(EXE)

$(LIB): $(LOBJ)
	$(AR) -rcs $@ $^

$(EXE): $(LOBJ) $(OBJ)
	$(LD) $(LDFLAGS) $(LOBJ) $(OBJ) -o $(EDIR)/$@

clean:
	rm -f $(OBJ) $(EXE) $(LOBJ) $(LIB) *.tmp $(SDEPS) $(LSDEPS) $(TSDEPS)

install: $(INC) $(LIB) $(EXE)
	mkdir -p $(DESTDIR)/bin $(DESTDIR)/include $(DESTDIR)/lib
	rm -f .footprint
	echo $(DESTDIR)/bin/$(EXE) >> .footprint
	cp -v $(EXE) $(DESTDIR)/bin/
	@for T in $(LIB); \
	do ( \
		echo $(DESTDIR)/$$T >> .footprint; \
		cp -v --parents $$T $(DESTDIR) \
	); done
	@for T in $(INC); \
	do ( \
		echo $(DESTDIR)/include/`basename -a $$T` >> .footprint; \
		cp -v $$T $(DESTDIR)/include/ \
	); done

uninstall: .footprint
	@for T in $(shell cat .footprint); do rm -v $$T; done

-include $(SDEPS) $(LSDEPS)

showconfig:
	@echo "PWD="$(PWD)
	@echo "VERSION="$(VERSION)
	@echo "RELEASE="$(RELEASE)
	@echo "LIBVER_MAJOR="$(LIBVER_MAJOR)
	@echo "LIBVER_MINOR="$(LIBVER_MINOR)
	@echo "LIBVER_PATCH="$(LIBVER_PATCH)
	@echo "LIBVER="$(LIBVER)
	@echo "CROSS="$(CROSS)
	@echo "CC="$(CC)
	@echo "AR="$(AR)
	@echo "LD="$(LD)
	@echo "DESTDIR="$(DESTDIR)
	@echo "CFLAGS="$(CFLAGS)
	@echo "LDFLAGS="$(LDFLAGS)
	@echo "SRC="$(SRC)
	@echo "OBJ="$(OBJ)
	@echo "IDIR="$(IDIR)
	@echo "INC="$(INC)
	@echo "EDIR="$(EDIR)
	@echo "EXE="$(EXE)
	@echo "LDIR="$(LDIR)
	@echo "LSRC="$(LSRC)
	@echo "LOBJ="$(LOBJ)
	@echo "LNK="$(LNK)
	@echo "LIB="$(LIB)
	@echo "SDEPS="$(SDEPS)
	@echo "LSDEPS="$(LSDEPS)

gstat:
	git status

gpush:
	git commit
	git push
