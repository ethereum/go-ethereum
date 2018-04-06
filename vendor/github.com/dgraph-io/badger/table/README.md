# BenchmarkRead

```
$ go test -bench Read$ -count 3

Size of table: 105843444
BenchmarkRead-8   	3	 343846914 ns/op
BenchmarkRead-8   	3	 351790907 ns/op
BenchmarkRead-8   	3	 351762823 ns/op
```

Size of table is 105,843,444 bytes, which is ~101M.

The rate is ~287M/s which matches our read speed. This is using mmap.

To read a 64M table, this would take ~0.22s, which is negligible.

```
$ go test -bench BenchmarkReadAndBuild -count 3

BenchmarkReadAndBuild-8   	       1	2341034225 ns/op
BenchmarkReadAndBuild-8   	       1	2346349671 ns/op
BenchmarkReadAndBuild-8   	       1	2364064576 ns/op
```

The rate is ~43M/s. To build a ~64M table, this would take ~1.5s. Note that this
does NOT include the flushing of the table to disk. All we are doing above is
to read one table (mmaped) and write one table in memory.

The table building takes 1.5-0.22 ~ 1.3s.

If we are writing out up to 10 tables, this would take 1.5*10 ~ 15s, and ~13s
is spent building the tables.

When running populate, building one table in memory tends to take ~1.5s to ~2.5s
on my system. Where does this overhead come from? Let's investigate the merging.

Below, we merge 5 tables. The total size remains unchanged at ~101M.

```
$ go test -bench ReadMerged -count 3
BenchmarkReadMerged-8   	       1	1321190264 ns/op
BenchmarkReadMerged-8   	       1	1296958737 ns/op
BenchmarkReadMerged-8   	       1	1314381178 ns/op
```

The rate is ~76M/s. To build a 64M table, this would take ~0.84s. The writing
takes ~1.3s as we saw above. So in total, we expect around 0.84+1.3 ~ 2.1s.
This roughly matches what we observe when running populate. There might be
some additional overhead due to the concurrent writes going on, in flushing the
table to disk. Also, the tables tend to be slightly bigger than 64M/s.