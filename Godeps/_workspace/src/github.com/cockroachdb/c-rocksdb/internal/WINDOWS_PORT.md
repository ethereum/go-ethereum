# Microsoft Contribution Notes

## Contributors
* Alexander Zinoviev https://github.com/zinoale
* Dmitri Smirnov https://github.com/yuslepukhin
* Praveen Rao  https://github.com/PraveenSinghRao
* Sherlock Huang  https://github.com/SherlockNoMad

## Introduction
RocksDB is a well proven open source key-value persistent store, optimized for fast storage. It provides scalability with number of CPUs and storage IOPS, to support IO-bound, in-memory and write-once workloads, most importantly, to be flexible to allow for innovation.

As Microsoft Bing team we have been continuously pushing hard to improve the scalability, efficiency of platform and eventually benefit Bing end-user satisfaction.  We would like to explore the opportunity to embrace open source, RocksDB here, to use, enhance and customize for our usage, and also contribute back to the RocksDB community. Herein, we are pleased to offer this RocksDB port for Windows platform.

These notes describe some decisions and changes we had to make with regards to porting RocksDB on Windows. We hope this will help both reviewers and users of the Windows port.
We are open for comments and improvements.

## OS specifics
All of the porting, testing and benchmarking was done on Windows Server 2012 R2 Datacenter 64-bit but to the best of our knowledge there is not a specific API we used during porting that is unsupported on other Windows OS after Vista.

## Porting goals
We strive to achieve the following goals:
* make use of the existing porting interface of RocksDB
* make minimum [WY2]modifications within platform independent code.
* make all unit test pass both in debug and release builds. 
  * Note: latest introduction of SyncPoint seems to disable running db_test in Release.
* make performance on par with published benchmarks accounting for HW differences
* we would like to keep the port code inline with the master branch with no forking

## Build system
We have chosen CMake as a widely accepted build system to build the Windows port. It is very fast and convenient. 

At the same time it generates Visual Studio projects that are both usable from a command line and IDE.

The top-level CMakeLists.txt file contains description of all targets and build rules. It also provides brief instructions on how to build the software for Windows. One more build related file is thirdparty.inc that also resides on the top level. This file must be edited to point to actual third party libraries location.
We think that it would be beneficial to merge the existing make-based build system and the new cmake-based build system into a single one to use on all platforms.

All building and testing was done for 64-bit. We have not conducted any testing for 32-bit and early reports indicate that it will not run on 32-bit.

## C++ and STL notes
We had to make some minimum changes within the portable files that either account for OS differences or the shortcomings of C++11 support in the current version of the MS compiler. Most or all of them are expected to be fixed in the upcoming compiler releases.

We plan to use this port for our business purposes here at Bing and this provided business justification for this port. This also means, we do not have at present to choose the compiler version at will.

* Certain headers that are not present and not necessary on Windows were simply `#ifndef OS_WIN` in a few places (`unistd.h`)
* All posix specific headers were replaced to port/port.h which worked well
* Replaced `dirent.h` for `port/dirent.h` (very few places) with the implementation of the relevant interfaces within `rocksdb::port` namespace
* Replaced `sys/time.h` to `port/sys_time.h` (few places) implemented equivalents within `rocksdb::port`
* `printf %z` specification is not supported on Windows. To imitate existing standards we came up with a string macro `ROCKSDB_PRIszt` which expands to `%z` on posix systems and to Iu on windows.
* in class member initialization were moved to a __ctors in some cases
* `constexpr` is not supported. We had to replace `std::numeric_limits<>::max/min()` to its C macros for constants. Sometimes we had to make class members `static const` and place a definition within a .cc file.
* `constexpr` for functions was replaced to a template specialization (1 place)
* Union members that have non-trivial constructors were replaced to `char[]` in one place along with bug fixes (spatial experimental feature)
* Zero-sized arrays are deemed a non-standard extension which we converted to 1 size array and that should work well for the purposes of these classes.
* `std::chrono` lacks nanoseconds support (fixed in the upcoming release of the STL) and we had to use `QueryPerfCounter()` within env_win.cc
* Function local statics initialization is still not safe. Used `std::once` to mitigate within WinEnv.

## Windows Environments notes
We endeavored to make it functionally on par with posix_env. This means we replicated the functionality of the thread pool and other things as precise as possible, including:
* Replicate posix logic using std:thread primitives.
* Implement all posix_env disk access functionality.
* Set `use_os_buffer=false` to disable OS disk buffering for WinWritableFile and WinRandomAccessFile.
* Replace `pread/pwrite` with `WriteFile/ReadFile` with `OVERLAPPED` structure.
* Use `SetFileInformationByHandle` to compensate absence of `fallocate`.

### In detail
Even though Windows provides its own efficient thread-pool implementation we chose to replicate posix logic using `std::thread` primitives. This allows anyone to quickly detect any changes within the posix source code and replicate them within windows env. This has proven to work very well. At the same time for anyone who wishes to replace the built-in thread-pool can do so using RocksDB stackable environments.

For disk access we implemented all of the functionality present within the posix_env which includes memory mapped files, random access, rate-limiter support etc.
The `use_os_buffer` flag on Posix platforms currently denotes disabling read-ahead log via `fadvise` mechanism. Windows does not have `fadvise` system call. What is more, it implements disk cache in a way that differs from Linux greatly. It’s not an uncommon practice on Windows to perform un-buffered disk access to gain control of the memory consumption. We think that in our use case this may also be a good configuration option at the expense of disk throughput. To compensate one may increase the configured in-memory cache size instead. Thus we have chosen  `use_os_buffer=false` to disable OS disk buffering for `WinWritableFile` and `WinRandomAccessFile`. The OS imposes restrictions on the alignment of the disk offsets, buffers used and the amount of data that is read/written when accessing files in un-buffered mode. When the option is true, the classes behave in a standard way. This allows to perform writes and reads in cases when un-buffered access does not make sense such as WAL and MANIFEST.

We have replaced `pread/pwrite` with `WriteFile/ReadFile` with `OVERLAPPED` structure so we can atomically seek to the position of the disk operation but still perform the operation synchronously. Thus we able to emulate that functionality of `pread/pwrite` reasonably well. The only difference is that the file pointer is not returned to its original position but that hardly matters given the random nature of access.

We used `SetFileInformationByHandle` both to truncate files after writing a full final page to disk and to pre-allocate disk space for faster I/O thus compensating for the absence of `fallocate` although some differences remain. For example, the pre-allocated space is not filled with zeros like on Linux, however, on a positive note, the end of file position is also not modified after pre-allocation.

RocksDB renames, copies and deletes files at will even though they may be opened with another handle at the same time. We had to relax and allow nearly all the concurrent access permissions possible.

## Thread-Local Storage
Thread-Local storage plays a significant role for RocksDB performance. Rather than creating a separate implementation we chose to create inline wrappers that forward `pthread_specific` calls to Windows `Tls` interfaces within `rocksdb::port` namespace. This leaves the existing meat of the logic in tact and unchanged and just as maintainable.

To mitigate the lack of thread local storage cleanup on thread-exit we added a limited amount of windows specific code within the same thread_local.cc file that injects a cleanup callback into a `"__tls"` structure within `".CRT$XLB"` data segment. This approach guarantees that the callback is invoked regardless of whether RocksDB used within an executable, standalone DLL or within another DLL.

## Jemalloc usage

When RocksDB is used with Jemalloc the latter needs to be initialized before any of the C++ globals or statics. To accomplish that we injected an initialization routine into `".CRT$XCT"` that is automatically invoked by the runtime before initializing static objects. je-uninit is queued to `atexit()`. 

The jemalloc redirecting `new/delete` global operators are used by the linker providing certain conditions are met. See build section in these notes.

## Stack Trace and Unhandled Exception Handler

We decided not to implement these two features because the hosting program as a rule has these two things in it.
We experienced no inconveniences debugging issues in the debugger or analyzing process dumps if need be and thus we did not
see this as a priority.

## Performance results
### Setup
All of the benchmarks are run on the same set of machines. Here are the details of the test setup:
* 2 Intel(R) Xeon(R) E5 2450 0 @ 2.10 GHz (total 16 cores)
* 2 XK0480GDQPH SSD Device, total 894GB free disk
* Machine has 128 GB of RAM
* Operating System: Windows Server 2012 R2 Datacenter
* 100 Million keys; each key is of size 10 bytes, each value is of size 800 bytes
* total database size is ~76GB
* The performance result is based on RocksDB 3.11.
* The parameters used, unless specified, were exactly the same as published in the GitHub Wiki page. 

### RocksDB on flash storage

#### Test 1. Bulk Load of keys in Random Order

Version 3.11 

* Total Run Time: 17.6 min
* Fillrandom: 5.480 micros/op 182465 ops/sec;  142.0 MB/s
* Compact: 486056544.000 micros/op 0 ops/sec

Version 3.10 

* Total Run Time: 16.2 min 
* Fillrandom: 5.018 micros/op 199269 ops/sec;  155.1 MB/s 
* Compact: 441313173.000 micros/op 0 ops/sec; 


#### Test 2. Bulk Load of keys in Sequential Order

Version 3.11 

* Fillseq: 4.944 micros/op 202k ops/sec;  157.4 MB/s

Version 3.10

* Fillseq: 4.105 micros/op 243.6k ops/sec;  189.6 MB/s 


#### Test 3. Random Write

Version 3.11 

* Unbuffered I/O enabled
* Overwrite: 52.661 micros/op 18.9k ops/sec;   14.8 MB/s

Version 3.10

* Unbuffered I/O enabled 
* Overwrite: 52.661 micros/op 18.9k ops/sec; 


#### Test 4. Random Read

Version 3.11 

* Unbuffered I/O enabled
* Readrandom: 15.716 micros/op 63.6k ops/sec; 49.5 MB/s 

Version 3.10

* Unbuffered I/O enabled 
* Readrandom: 15.548 micros/op 64.3k ops/sec; 


#### Test 5. Multi-threaded read and single-threaded write

Version 3.11

* Unbuffered I/O enabled
* Readwhilewriting: 25.128 micros/op 39.7k ops/sec; 

Version 3.10

* Unbuffered I/O enabled 
* Readwhilewriting: 24.854 micros/op 40.2k ops/sec; 


### RocksDB In Memory 

#### Test 1. Point Lookup

Version 3.11

80K writes/sec
* Write Rate Achieved: 40.5k write/sec;
* Readwhilewriting: 0.314 micros/op 3187455 ops/sec;  364.8 MB/s (715454999 of 715454999 found)

Version 3.10

* Write Rate Achieved:  50.6k write/sec 
* Readwhilewriting: 0.316 micros/op 3162028 ops/sec; (719576999 of 719576999 found) 


*10K writes/sec*

Version 3.11

* Write Rate Achieved: 5.8k/s write/sec
* Readwhilewriting: 0.246 micros/op 4062669 ops/sec;  464.9 MB/s (915481999 of 915481999 found)

Version 3.10

* Write Rate Achieved: 5.8k/s write/sec 
* Readwhilewriting: 0.244 micros/op 4106253 ops/sec; (927986999 of 927986999 found) 


#### Test 2. Prefix Range Query

Version 3.11

80K writes/sec
* Write Rate Achieved:  46.3k/s write/sec
* Readwhilewriting: 0.362 micros/op 2765052 ops/sec;  316.4 MB/s (611549999 of 611549999 found)

Version 3.10

* Write Rate Achieved: 45.8k/s write/sec 
* Readwhilewriting: 0.317 micros/op 3154941 ops/sec; (708158999 of 708158999 found) 

Version 3.11

10K writes/sec
* Write Rate Achieved: 5.78k write/sec
* Readwhilewriting: 0.269 micros/op 3716692 ops/sec;  425.3 MB/s (837401999 of 837401999 found)

Version 3.10

* Write Rate Achieved: 5.7k write/sec 
* Readwhilewriting: 0.261 micros/op 3830152 ops/sec; (863482999 of 863482999 found) 


We think that there is still big room to improve the performance, which will be an ongoing effort for us.

