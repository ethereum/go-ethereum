import os
import glob
import os.path
import shutil
import subprocess
import time
import unittest
import tempfile
import re

def my_check_output(*popenargs, **kwargs):
    """
    If we had python 2.7, we should simply use subprocess.check_output.
    This is a stop-gap solution for python 2.6
    """
    if 'stdout' in kwargs:
        raise ValueError('stdout argument not allowed, it will be overridden.')
    process = subprocess.Popen(stderr=subprocess.PIPE, stdout=subprocess.PIPE,
                               *popenargs, **kwargs)
    output, unused_err = process.communicate()
    retcode = process.poll()
    if retcode:
        cmd = kwargs.get("args")
        if cmd is None:
            cmd = popenargs[0]
        raise Exception("Exit code is not 0.  It is %d.  Command: %s" %
                (retcode, cmd))
    return output

def run_err_null(cmd):
    return os.system(cmd + " 2>/dev/null ")

class LDBTestCase(unittest.TestCase):
    def setUp(self):
        self.TMP_DIR  = tempfile.mkdtemp(prefix="ldb_test_")
        self.DB_NAME = "testdb"

    def tearDown(self):
        assert(self.TMP_DIR.strip() != "/"
                and self.TMP_DIR.strip() != "/tmp"
                and self.TMP_DIR.strip() != "/tmp/") #Just some paranoia

        shutil.rmtree(self.TMP_DIR)

    def dbParam(self, dbName):
        return "--db=%s" % os.path.join(self.TMP_DIR, dbName)

    def assertRunOKFull(self, params, expectedOutput, unexpected=False,
                        isPattern=False):
        """
        All command-line params must be specified.
        Allows full flexibility in testing; for example: missing db param.

        """

        output = my_check_output("./ldb %s |grep -v \"Created bg thread\"" %
                            params, shell=True)
        if not unexpected:
            if isPattern:
                self.assertNotEqual(expectedOutput.search(output.strip()),
                                    None)
            else:
                self.assertEqual(output.strip(), expectedOutput.strip())
        else:
            if isPattern:
                self.assertEqual(expectedOutput.search(output.strip()), None)
            else:
                self.assertNotEqual(output.strip(), expectedOutput.strip())

    def assertRunFAILFull(self, params):
        """
        All command-line params must be specified.
        Allows full flexibility in testing; for example: missing db param.

        """
        try:

            my_check_output("./ldb %s >/dev/null 2>&1 |grep -v \"Created bg \
                thread\"" % params, shell=True)
        except Exception, e:
            return
        self.fail(
            "Exception should have been raised for command with params: %s" %
            params)

    def assertRunOK(self, params, expectedOutput, unexpected=False):
        """
        Uses the default test db.

        """
        self.assertRunOKFull("%s %s" % (self.dbParam(self.DB_NAME), params),
                             expectedOutput, unexpected)

    def assertRunFAIL(self, params):
        """
        Uses the default test db.
        """
        self.assertRunFAILFull("%s %s" % (self.dbParam(self.DB_NAME), params))

    def testSimpleStringPutGet(self):
        print "Running testSimpleStringPutGet..."
        self.assertRunFAIL("put x1 y1")
        self.assertRunOK("put --create_if_missing x1 y1", "OK")
        self.assertRunOK("get x1", "y1")
        self.assertRunFAIL("get x2")

        self.assertRunOK("put x2 y2", "OK")
        self.assertRunOK("get x1", "y1")
        self.assertRunOK("get x2", "y2")
        self.assertRunFAIL("get x3")

        self.assertRunOK("scan --from=x1 --to=z", "x1 : y1\nx2 : y2")
        self.assertRunOK("put x3 y3", "OK")

        self.assertRunOK("scan --from=x1 --to=z", "x1 : y1\nx2 : y2\nx3 : y3")
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3")
        self.assertRunOK("scan --from=x", "x1 : y1\nx2 : y2\nx3 : y3")

        self.assertRunOK("scan --to=x2", "x1 : y1")
        self.assertRunOK("scan --from=x1 --to=z --max_keys=1", "x1 : y1")
        self.assertRunOK("scan --from=x1 --to=z --max_keys=2",
                "x1 : y1\nx2 : y2")

        self.assertRunOK("scan --from=x1 --to=z --max_keys=3",
                "x1 : y1\nx2 : y2\nx3 : y3")
        self.assertRunOK("scan --from=x1 --to=z --max_keys=4",
                "x1 : y1\nx2 : y2\nx3 : y3")
        self.assertRunOK("scan --from=x1 --to=x2", "x1 : y1")
        self.assertRunOK("scan --from=x2 --to=x4", "x2 : y2\nx3 : y3")
        self.assertRunFAIL("scan --from=x4 --to=z") # No results => FAIL
        self.assertRunFAIL("scan --from=x1 --to=z --max_keys=foo")

        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3")

        self.assertRunOK("delete x1", "OK")
        self.assertRunOK("scan", "x2 : y2\nx3 : y3")

        self.assertRunOK("delete NonExistentKey", "OK")
        # It is weird that GET and SCAN raise exception for
        # non-existent key, while delete does not

        self.assertRunOK("checkconsistency", "OK")

    def dumpDb(self, params, dumpFile):
        return 0 == run_err_null("./ldb dump %s > %s" % (params, dumpFile))

    def loadDb(self, params, dumpFile):
        return 0 == run_err_null("cat %s | ./ldb load %s" % (dumpFile, params))

    def testStringBatchPut(self):
        print "Running testStringBatchPut..."
        self.assertRunOK("batchput x1 y1 --create_if_missing", "OK")
        self.assertRunOK("scan", "x1 : y1")
        self.assertRunOK("batchput x2 y2 x3 y3 \"x4 abc\" \"y4 xyz\"", "OK")
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 abc : y4 xyz")
        self.assertRunFAIL("batchput")
        self.assertRunFAIL("batchput k1")
        self.assertRunFAIL("batchput k1 v1 k2")

    def testCountDelimDump(self):
        print "Running testCountDelimDump..."
        self.assertRunOK("batchput x.1 x1 --create_if_missing", "OK")
        self.assertRunOK("batchput y.abc abc y.2 2 z.13c pqr", "OK")
        self.assertRunOK("dump --count_delim", "x => count:1\tsize:5\ny => count:2\tsize:12\nz => count:1\tsize:8")
        self.assertRunOK("dump --count_delim=\".\"", "x => count:1\tsize:5\ny => count:2\tsize:12\nz => count:1\tsize:8")
        self.assertRunOK("batchput x,2 x2 x,abc xabc", "OK")
        self.assertRunOK("dump --count_delim=\",\"", "x => count:2\tsize:14\nx.1 => count:1\tsize:5\ny.2 => count:1\tsize:4\ny.abc => count:1\tsize:8\nz.13c => count:1\tsize:8")

    def testCountDelimIDump(self):
        print "Running testCountDelimIDump..."
        self.assertRunOK("batchput x.1 x1 --create_if_missing", "OK")
        self.assertRunOK("batchput y.abc abc y.2 2 z.13c pqr", "OK")
        self.assertRunOK("dump --count_delim", "x => count:1\tsize:5\ny => count:2\tsize:12\nz => count:1\tsize:8")
        self.assertRunOK("dump --count_delim=\".\"", "x => count:1\tsize:5\ny => count:2\tsize:12\nz => count:1\tsize:8")
        self.assertRunOK("batchput x,2 x2 x,abc xabc", "OK")
        self.assertRunOK("dump --count_delim=\",\"", "x => count:2\tsize:14\nx.1 => count:1\tsize:5\ny.2 => count:1\tsize:4\ny.abc => count:1\tsize:8\nz.13c => count:1\tsize:8")

    def testInvalidCmdLines(self):
        print "Running testInvalidCmdLines..."
        # db not specified
        self.assertRunFAILFull("put 0x6133 0x6233 --hex --create_if_missing")
        # No param called he
        self.assertRunFAIL("put 0x6133 0x6233 --he --create_if_missing")
        # max_keys is not applicable for put
        self.assertRunFAIL("put 0x6133 0x6233 --max_keys=1 --create_if_missing")
        # hex has invalid boolean value

    def testHexPutGet(self):
        print "Running testHexPutGet..."
        self.assertRunOK("put a1 b1 --create_if_missing", "OK")
        self.assertRunOK("scan", "a1 : b1")
        self.assertRunOK("scan --hex", "0x6131 : 0x6231")
        self.assertRunFAIL("put --hex 6132 6232")
        self.assertRunOK("put --hex 0x6132 0x6232", "OK")
        self.assertRunOK("scan --hex", "0x6131 : 0x6231\n0x6132 : 0x6232")
        self.assertRunOK("scan", "a1 : b1\na2 : b2")
        self.assertRunOK("get a1", "b1")
        self.assertRunOK("get --hex 0x6131", "0x6231")
        self.assertRunOK("get a2", "b2")
        self.assertRunOK("get --hex 0x6132", "0x6232")
        self.assertRunOK("get --key_hex 0x6132", "b2")
        self.assertRunOK("get --key_hex --value_hex 0x6132", "0x6232")
        self.assertRunOK("get --value_hex a2", "0x6232")
        self.assertRunOK("scan --key_hex --value_hex",
                "0x6131 : 0x6231\n0x6132 : 0x6232")
        self.assertRunOK("scan --hex --from=0x6131 --to=0x6133",
                "0x6131 : 0x6231\n0x6132 : 0x6232")
        self.assertRunOK("scan --hex --from=0x6131 --to=0x6132",
                "0x6131 : 0x6231")
        self.assertRunOK("scan --key_hex", "0x6131 : b1\n0x6132 : b2")
        self.assertRunOK("scan --value_hex", "a1 : 0x6231\na2 : 0x6232")
        self.assertRunOK("batchput --hex 0x6133 0x6233 0x6134 0x6234", "OK")
        self.assertRunOK("scan", "a1 : b1\na2 : b2\na3 : b3\na4 : b4")
        self.assertRunOK("delete --hex 0x6133", "OK")
        self.assertRunOK("scan", "a1 : b1\na2 : b2\na4 : b4")
        self.assertRunOK("checkconsistency", "OK")

    def testTtlPutGet(self):
        print "Running testTtlPutGet..."
        self.assertRunOK("put a1 b1 --ttl --create_if_missing", "OK")
        self.assertRunOK("scan --hex", "0x6131 : 0x6231", True)
        self.assertRunOK("dump --ttl ", "a1 ==> b1", True)
        self.assertRunOK("dump --hex --ttl ",
                         "0x6131 ==> 0x6231\nKeys in range: 1")
        self.assertRunOK("scan --hex --ttl", "0x6131 : 0x6231")
        self.assertRunOK("get --value_hex a1", "0x6231", True)
        self.assertRunOK("get --ttl a1", "b1")
        self.assertRunOK("put a3 b3 --create_if_missing", "OK")
        # fails because timstamp's length is greater than value's
        self.assertRunFAIL("get --ttl a3")
        self.assertRunOK("checkconsistency", "OK")

    def testInvalidCmdLines(self):
        print "Running testInvalidCmdLines..."
        # db not specified
        self.assertRunFAILFull("put 0x6133 0x6233 --hex --create_if_missing")
        # No param called he
        self.assertRunFAIL("put 0x6133 0x6233 --he --create_if_missing")
        # max_keys is not applicable for put
        self.assertRunFAIL("put 0x6133 0x6233 --max_keys=1 --create_if_missing")
        # hex has invalid boolean value
        self.assertRunFAIL("put 0x6133 0x6233 --hex=Boo --create_if_missing")

    def testDumpLoad(self):
        print "Running testDumpLoad..."
        self.assertRunOK("batchput --create_if_missing x1 y1 x2 y2 x3 y3 x4 y4",
                "OK")
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")
        origDbPath = os.path.join(self.TMP_DIR, self.DB_NAME)

        # Dump and load without any additional params specified
        dumpFilePath = os.path.join(self.TMP_DIR, "dump1")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump1")
        self.assertTrue(self.dumpDb("--db=%s" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --create_if_missing" % loadedDbPath, dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        # Dump and load in hex
        dumpFilePath = os.path.join(self.TMP_DIR, "dump2")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump2")
        self.assertTrue(self.dumpDb("--db=%s --hex" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --hex --create_if_missing" % loadedDbPath, dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        # Dump only a portion of the key range
        dumpFilePath = os.path.join(self.TMP_DIR, "dump3")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump3")
        self.assertTrue(self.dumpDb(
            "--db=%s --from=x1 --to=x3" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --create_if_missing" % loadedDbPath, dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath, "x1 : y1\nx2 : y2")

        # Dump upto max_keys rows
        dumpFilePath = os.path.join(self.TMP_DIR, "dump4")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump4")
        self.assertTrue(self.dumpDb(
            "--db=%s --max_keys=3" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --create_if_missing" % loadedDbPath, dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3")

        # Load into an existing db, create_if_missing is not specified
        self.assertTrue(self.dumpDb("--db=%s" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb("--db=%s" % loadedDbPath, dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        # Dump and load with WAL disabled
        dumpFilePath = os.path.join(self.TMP_DIR, "dump5")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump5")
        self.assertTrue(self.dumpDb("--db=%s" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --disable_wal --create_if_missing" % loadedDbPath,
            dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        # Dump and load with lots of extra params specified
        extraParams = " ".join(["--bloom_bits=14", "--block_size=1024",
                                "--auto_compaction=true",
                                "--write_buffer_size=4194304",
                                "--file_size=2097152"])
        dumpFilePath = os.path.join(self.TMP_DIR, "dump6")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump6")
        self.assertTrue(self.dumpDb(
            "--db=%s %s" % (origDbPath, extraParams), dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s %s --create_if_missing" % (loadedDbPath, extraParams),
            dumpFilePath))
        self.assertRunOKFull("scan --db=%s" % loadedDbPath,
                "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        # Dump with count_only
        dumpFilePath = os.path.join(self.TMP_DIR, "dump7")
        loadedDbPath = os.path.join(self.TMP_DIR, "loaded_from_dump7")
        self.assertTrue(self.dumpDb(
            "--db=%s --count_only" % origDbPath, dumpFilePath))
        self.assertTrue(self.loadDb(
            "--db=%s --create_if_missing" % loadedDbPath, dumpFilePath))
        # DB should have atleast one value for scan to work
        self.assertRunOKFull("put --db=%s k1 v1" % loadedDbPath, "OK")
        self.assertRunOKFull("scan --db=%s" % loadedDbPath, "k1 : v1")

        # Dump command fails because of typo in params
        dumpFilePath = os.path.join(self.TMP_DIR, "dump8")
        self.assertFalse(self.dumpDb(
            "--db=%s --create_if_missing" % origDbPath, dumpFilePath))

    def testMiscAdminTask(self):
        print "Running testMiscAdminTask..."
        # These tests need to be improved; for example with asserts about
        # whether compaction or level reduction actually took place.
        self.assertRunOK("batchput --create_if_missing x1 y1 x2 y2 x3 y3 x4 y4",
                "OK")
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")
        origDbPath = os.path.join(self.TMP_DIR, self.DB_NAME)

        self.assertTrue(0 == run_err_null(
            "./ldb compact --db=%s" % origDbPath))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        self.assertTrue(0 == run_err_null(
            "./ldb reduce_levels --db=%s --new_levels=2" % origDbPath))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        self.assertTrue(0 == run_err_null(
            "./ldb reduce_levels --db=%s --new_levels=3" % origDbPath))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        self.assertTrue(0 == run_err_null(
            "./ldb compact --db=%s --from=x1 --to=x3" % origDbPath))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        self.assertTrue(0 == run_err_null(
            "./ldb compact --db=%s --hex --from=0x6131 --to=0x6134"
            % origDbPath))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

        #TODO(dilip): Not sure what should be passed to WAL.Currently corrupted.
        self.assertTrue(0 == run_err_null(
            "./ldb dump_wal --db=%s --walfile=%s --header" % (
                origDbPath, os.path.join(origDbPath, "LOG"))))
        self.assertRunOK("scan", "x1 : y1\nx2 : y2\nx3 : y3\nx4 : y4")

    def testCheckConsistency(self):
        print "Running testCheckConsistency..."

        dbPath = os.path.join(self.TMP_DIR, self.DB_NAME)
        self.assertRunOK("put x1 y1 --create_if_missing", "OK")
        self.assertRunOK("put x2 y2", "OK")
        self.assertRunOK("get x1", "y1")
        self.assertRunOK("checkconsistency", "OK")

        sstFilePath = my_check_output("ls %s" % os.path.join(dbPath, "*.sst"),
                                      shell=True)

        # Modify the file
        my_check_output("echo 'evil' > %s" % sstFilePath, shell=True)
        self.assertRunFAIL("checkconsistency")

        # Delete the file
        my_check_output("rm -f %s" % sstFilePath, shell=True)
        self.assertRunFAIL("checkconsistency")

    def dumpLiveFiles(self, params, dumpFile):
        return 0 == run_err_null("./ldb dump_live_files %s > %s" % (
            params, dumpFile))

    def testDumpLiveFiles(self):
        print "Running testDumpLiveFiles..."

        dbPath = os.path.join(self.TMP_DIR, self.DB_NAME)
        self.assertRunOK("put x1 y1 --create_if_missing", "OK")
        self.assertRunOK("put x2 y2", "OK")
        dumpFilePath = os.path.join(self.TMP_DIR, "dump1")
        self.assertTrue(self.dumpLiveFiles("--db=%s" % dbPath, dumpFilePath))
        self.assertRunOK("delete x1", "OK")
        self.assertRunOK("put x3 y3", "OK")
        dumpFilePath = os.path.join(self.TMP_DIR, "dump2")
        self.assertTrue(self.dumpLiveFiles("--db=%s" % dbPath, dumpFilePath))

    def getManifests(self, directory):
        return glob.glob(directory + "/MANIFEST-*")

    def copyManifests(self, src, dest):
        return 0 == run_err_null("cp " + src + " " + dest)

    def testManifestDump(self):
        print "Running testManifestDump..."
        dbPath = os.path.join(self.TMP_DIR, self.DB_NAME)
        self.assertRunOK("put 1 1 --create_if_missing", "OK")
        self.assertRunOK("put 2 2", "OK")
        self.assertRunOK("put 3 3", "OK")
        # Pattern to expect from manifest_dump.
        num = "[0-9]+"
        st = ".*"
        subpat = st + " @ " + num + ": " + num
        regex = num + ":" + num + "\[" + subpat + ".." + subpat + "\]"
        expected_pattern = re.compile(regex)
        cmd = "manifest_dump --db=%s"
        manifest_files = self.getManifests(dbPath)
        self.assertTrue(len(manifest_files) == 1)
        # Test with the default manifest file in dbPath.
        self.assertRunOKFull(cmd % dbPath, expected_pattern,
                             unexpected=False, isPattern=True)
        self.copyManifests(manifest_files[0], manifest_files[0] + "1")
        manifest_files = self.getManifests(dbPath)
        self.assertTrue(len(manifest_files) == 2)
        # Test with multiple manifest files in dbPath.
        self.assertRunFAILFull(cmd % dbPath)
        # Running it with the copy we just created should pass.
        self.assertRunOKFull((cmd + " --path=%s")
                             % (dbPath, manifest_files[1]),
                             expected_pattern, unexpected=False,
                             isPattern=True)

    def testListColumnFamilies(self):
        print "Running testListColumnFamilies..."
        dbPath = os.path.join(self.TMP_DIR, self.DB_NAME)
        self.assertRunOK("put x1 y1 --create_if_missing", "OK")
        cmd = "list_column_families %s | grep -v \"Column families\""
        # Test on valid dbPath.
        self.assertRunOKFull(cmd % dbPath, "{default}")
        # Test on empty path.
        self.assertRunFAILFull(cmd % "")



if __name__ == "__main__":
    unittest.main()
