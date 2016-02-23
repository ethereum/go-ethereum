ACLOCAL_AMFLAGS = -I m4

# Library.
lib_LTLIBRARIES = libsnappy.la
libsnappy_la_SOURCES = snappy.cc snappy-sinksource.cc snappy-stubs-internal.cc snappy-c.cc
libsnappy_la_LDFLAGS = -version-info $(SNAPPY_LTVERSION)

include_HEADERS = snappy.h snappy-sinksource.h snappy-stubs-public.h snappy-c.h
noinst_HEADERS = snappy-internal.h snappy-stubs-internal.h snappy-test.h

# Unit tests and benchmarks.
snappy_unittest_CPPFLAGS = $(gflags_CFLAGS) $(GTEST_CPPFLAGS)
snappy_unittest_SOURCES = snappy_unittest.cc snappy-test.cc
snappy_unittest_LDFLAGS = $(GTEST_LDFLAGS)
snappy_unittest_LDADD = libsnappy.la $(UNITTEST_LIBS) $(gflags_LIBS) $(GTEST_LIBS)
TESTS = snappy_unittest
noinst_PROGRAMS = $(TESTS)

EXTRA_DIST = autogen.sh testdata/alice29.txt testdata/asyoulik.txt testdata/baddata1.snappy testdata/baddata2.snappy testdata/baddata3.snappy testdata/geo.protodata testdata/fireworks.jpeg testdata/html testdata/html_x_4 testdata/kppkn.gtb testdata/lcet10.txt testdata/paper-100k.pdf testdata/plrabn12.txt testdata/urls.10K
dist_doc_DATA = ChangeLog COPYING INSTALL NEWS README format_description.txt framing_format.txt

libtool: $(LIBTOOL_DEPS)
	$(SHELL) ./config.status --recheck
