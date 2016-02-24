NATIVE_JAVA_CLASSES = org.rocksdb.AbstractCompactionFilter\
        org.rocksdb.AbstractComparator\
	org.rocksdb.AbstractSlice\
	org.rocksdb.BackupEngine\
	org.rocksdb.BackupableDB\
	org.rocksdb.BackupableDBOptions\
	org.rocksdb.BlockBasedTableConfig\
	org.rocksdb.BloomFilter\
	org.rocksdb.Checkpoint\
	org.rocksdb.ColumnFamilyHandle\
	org.rocksdb.ColumnFamilyOptions\
	org.rocksdb.Comparator\
	org.rocksdb.ComparatorOptions\
	org.rocksdb.DBOptions\
	org.rocksdb.DirectComparator\
	org.rocksdb.DirectSlice\
	org.rocksdb.Env\
	org.rocksdb.FlushOptions\
	org.rocksdb.Filter\
	org.rocksdb.GenericRateLimiterConfig\
	org.rocksdb.HashLinkedListMemTableConfig\
	org.rocksdb.HashSkipListMemTableConfig\
	org.rocksdb.Logger\
	org.rocksdb.MergeOperator\
	org.rocksdb.Options\
	org.rocksdb.PlainTableConfig\
	org.rocksdb.ReadOptions\
	org.rocksdb.RemoveEmptyValueCompactionFilter\
	org.rocksdb.RestoreBackupableDB\
	org.rocksdb.RestoreOptions\
	org.rocksdb.RocksDB\
	org.rocksdb.RocksEnv\
	org.rocksdb.RocksIterator\
	org.rocksdb.RocksMemEnv\
	org.rocksdb.SkipListMemTableConfig\
	org.rocksdb.Slice\
	org.rocksdb.Statistics\
	org.rocksdb.TransactionLogIterator\
	org.rocksdb.TtlDB\
	org.rocksdb.VectorMemTableConfig\
	org.rocksdb.Snapshot\
	org.rocksdb.StringAppendOperator\
	org.rocksdb.WriteBatch\
	org.rocksdb.WriteBatch.Handler\
	org.rocksdb.WriteOptions\
	org.rocksdb.WriteBatchWithIndex\
	org.rocksdb.WBWIRocksIterator

NATIVE_JAVA_TEST_CLASSES = org.rocksdb.WriteBatchTest\
    org.rocksdb.WriteBatchTestInternalHelper

ROCKSDB_MAJOR = $(shell egrep "ROCKSDB_MAJOR.[0-9]" ../include/rocksdb/version.h | cut -d ' ' -f 3)
ROCKSDB_MINOR = $(shell egrep "ROCKSDB_MINOR.[0-9]" ../include/rocksdb/version.h | cut -d ' ' -f 3)
ROCKSDB_PATCH = $(shell egrep "ROCKSDB_PATCH.[0-9]" ../include/rocksdb/version.h | cut -d ' ' -f 3)

NATIVE_INCLUDE = ./include
ARCH := $(shell getconf LONG_BIT)
ROCKSDB_JAR = rocksdbjni-$(ROCKSDB_MAJOR).$(ROCKSDB_MINOR).$(ROCKSDB_PATCH)-linux$(ARCH).jar
ifeq ($(PLATFORM), OS_MACOSX)
ROCKSDB_JAR = rocksdbjni-$(ROCKSDB_MAJOR).$(ROCKSDB_MINOR).$(ROCKSDB_PATCH)-osx.jar
endif

JAVA_TESTS = org.rocksdb.BackupableDBOptionsTest\
        org.rocksdb.BackupEngineTest\
	org.rocksdb.BackupableDBTest\
	org.rocksdb.BlockBasedTableConfigTest\
	org.rocksdb.CheckPointTest\
	org.rocksdb.ColumnFamilyOptionsTest\
	org.rocksdb.ColumnFamilyTest\
	org.rocksdb.ComparatorOptionsTest\
	org.rocksdb.ComparatorTest\
	org.rocksdb.CompressionOptionsTest\
	org.rocksdb.DBOptionsTest\
	org.rocksdb.DirectComparatorTest\
	org.rocksdb.DirectSliceTest\
	org.rocksdb.util.EnvironmentTest\
	org.rocksdb.FilterTest\
	org.rocksdb.FlushTest\
	org.rocksdb.InfoLogLevelTest\
	org.rocksdb.KeyMayExistTest\
        org.rocksdb.LoggerTest\
	org.rocksdb.MemTableTest\
	org.rocksdb.MergeTest\
	org.rocksdb.MixedOptionsTest\
        org.rocksdb.NativeLibraryLoaderTest\
	org.rocksdb.OptionsTest\
	org.rocksdb.PlainTableConfigTest\
	org.rocksdb.ReadOnlyTest\
	org.rocksdb.ReadOptionsTest\
	org.rocksdb.RocksDBTest\
	org.rocksdb.RocksEnvTest\
	org.rocksdb.RocksIteratorTest\
	org.rocksdb.RocksMemEnvTest\
	org.rocksdb.util.SizeUnitTest\
	org.rocksdb.SliceTest\
	org.rocksdb.SnapshotTest\
	org.rocksdb.TransactionLogIteratorTest\
	org.rocksdb.TtlDBTest\
	org.rocksdb.StatisticsCollectorTest\
	org.rocksdb.WriteBatchHandlerTest\
	org.rocksdb.WriteBatchTest\
	org.rocksdb.WriteOptionsTest\
	org.rocksdb.WriteBatchWithIndexTest

MAIN_SRC = src/main/java
TEST_SRC = src/test/java
OUTPUT = target
MAIN_CLASSES = $(OUTPUT)/classes
TEST_CLASSES = $(OUTPUT)/test-classes
JAVADOC = $(OUTPUT)/apidocs

BENCHMARK_MAIN_SRC = benchmark/src/main/java
BENCHMARK_OUTPUT = benchmark/target
BENCHMARK_MAIN_CLASSES = $(BENCHMARK_OUTPUT)/classes

SAMPLES_MAIN_SRC = samples/src/main/java
SAMPLES_OUTPUT = samples/target
SAMPLES_MAIN_CLASSES = $(SAMPLES_OUTPUT)/classes

JAVA_TEST_LIBDIR = test-libs
JAVA_JUNIT_JAR = $(JAVA_TEST_LIBDIR)/junit-4.12.jar
JAVA_HAMCR_JAR = $(JAVA_TEST_LIBDIR)/hamcrest-core-1.3.jar
JAVA_MOCKITO_JAR = $(JAVA_TEST_LIBDIR)/mockito-all-1.10.19.jar
JAVA_CGLIB_JAR = $(JAVA_TEST_LIBDIR)/cglib-2.2.2.jar
JAVA_ASSERTJ_JAR = $(JAVA_TEST_LIBDIR)/assertj-core-1.7.1.jar
JAVA_TESTCLASSPATH = $(JAVA_JUNIT_JAR):$(JAVA_HAMCR_JAR):$(JAVA_MOCKITO_JAR):$(JAVA_CGLIB_JAR):$(JAVA_ASSERTJ_JAR)

MVN_LOCAL = ~/.m2/repository

clean:
	rm -rf include/*
	rm -rf test-libs/
	rm -rf $(OUTPUT)
	rm -rf $(BENCHMARK_OUTPUT)
	rm -rf $(SAMPLES_OUTPUT)


javadocs:
	mkdir -p $(JAVADOC)
	javadoc -d $(JAVADOC) -sourcepath $(MAIN_SRC) -subpackages org

javalib: java java_test javadocs

java:
	mkdir -p $(MAIN_CLASSES)
	javac -d $(MAIN_CLASSES)\
		$(MAIN_SRC)/org/rocksdb/util/*.java\
		$(MAIN_SRC)/org/rocksdb/*.java
	@cp ../HISTORY.md ./HISTORY-CPP.md
	@rm -f ./HISTORY-CPP.md
	javah -cp $(MAIN_CLASSES) -d $(NATIVE_INCLUDE) -jni $(NATIVE_JAVA_CLASSES)

sample: java
	mkdir -p $(SAMPLES_MAIN_CLASSES)
	javac -cp $(MAIN_CLASSES) -d $(SAMPLES_MAIN_CLASSES) $(SAMPLES_MAIN_SRC)/RocksDBSample.java
	@rm -rf /tmp/rocksdbjni
	@rm -rf /tmp/rocksdbjni_not_found
	java -ea -Xcheck:jni -Djava.library.path=target -cp $(MAIN_CLASSES):$(SAMPLES_MAIN_CLASSES) RocksDBSample /tmp/rocksdbjni
	@rm -rf /tmp/rocksdbjni
	@rm -rf /tmp/rocksdbjni_not_found

column_family_sample: java
	mkdir -p $(SAMPLES_MAIN_CLASSES)
	javac -cp $(MAIN_CLASSES) -d $(SAMPLES_MAIN_CLASSES) $(SAMPLES_MAIN_SRC)/RocksDBColumnFamilySample.java
	@rm -rf /tmp/rocksdbjni
	java -ea -Xcheck:jni -Djava.library.path=target -cp $(MAIN_CLASSES):$(SAMPLES_MAIN_CLASSES) RocksDBColumnFamilySample /tmp/rocksdbjni
	@rm -rf /tmp/rocksdbjni

resolve_test_deps:
	test -s "$(JAVA_TEST_LIBDIR)" || mkdir -p "$(JAVA_TEST_LIBDIR)"
	test -s "$(JAVA_JUNIT_JAR)" || cp $(MVN_LOCAL)/junit/junit/4.12/junit-4.12.jar $(JAVA_TEST_LIBDIR) || curl -k -L -o $(JAVA_JUNIT_JAR) http://search.maven.org/remotecontent?filepath=junit/junit/4.12/junit-4.12.jar
	test -s "$(JAVA_HAMCR_JAR)" || cp $(MVN_LOCAL)/org/hamcrest/hamcrest-core/1.3/hamcrest-core-1.3.jar $(JAVA_TEST_LIBDIR) || curl -k -L -o $(JAVA_HAMCR_JAR) http://search.maven.org/remotecontent?filepath=org/hamcrest/hamcrest-core/1.3/hamcrest-core-1.3.jar
	test -s "$(JAVA_MOCKITO_JAR)" || cp $(MVN_LOCAL)/org/mockito/mockito-all/1.10.19/mockito-all-1.10.19.jar $(JAVA_TEST_LIBDIR) || curl -k -L -o "$(JAVA_MOCKITO_JAR)" http://search.maven.org/remotecontent?filepath=org/mockito/mockito-all/1.10.19/mockito-all-1.10.19.jar
	test -s "$(JAVA_CGLIB_JAR)" || cp $(MVN_LOCAL)/cglib/cglib/2.2.2/cglib-2.2.2.jar $(JAVA_TEST_LIBDIR) || curl -k -L -o "$(JAVA_CGLIB_JAR)" http://search.maven.org/remotecontent?filepath=cglib/cglib/2.2.2/cglib-2.2.2.jar
	test -s "$(JAVA_ASSERTJ_JAR)" || cp $(MVN_LOCAL)/org/assertj/assertj-core/1.7.1/assertj-core-1.7.1.jar $(JAVA_TEST_LIBDIR) || curl -k -L -o "$(JAVA_ASSERTJ_JAR)" http://central.maven.org/maven2/org/assertj/assertj-core/1.7.1/assertj-core-1.7.1.jar

java_test: resolve_test_deps
	mkdir -p $(TEST_CLASSES)
	javac -cp $(MAIN_CLASSES):$(JAVA_TESTCLASSPATH) -d $(TEST_CLASSES)\
		$(TEST_SRC)/org/rocksdb/test/*.java\
		$(TEST_SRC)/org/rocksdb/util/*.java\
		$(TEST_SRC)/org/rocksdb/*.java
	javah -cp $(MAIN_CLASSES):$(TEST_CLASSES) -d $(NATIVE_INCLUDE) -jni $(NATIVE_JAVA_TEST_CLASSES)

test: java resolve_test_deps java_test
	java -ea -Xcheck:jni -Djava.library.path=target -cp "$(MAIN_CLASSES):$(TEST_CLASSES):$(JAVA_TESTCLASSPATH):target/*" org.rocksdb.test.RocksJunitRunner $(JAVA_TESTS)

db_bench: java
	mkdir -p $(BENCHMARK_MAIN_CLASSES)
	javac -cp $(MAIN_CLASSES) -d $(BENCHMARK_MAIN_CLASSES) $(BENCHMARK_MAIN_SRC)/org/rocksdb/benchmark/*.java
