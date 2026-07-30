# PBT flat state: measurements

## Superseded — read this first

The in-memory numbers below were **measured against a Go map**
(`rawdb.NewMemoryDatabase()`) and the justification given for them was wrong in
two ways. Both are corrected here; the table is kept because the before/after it
records is still the only measurement of the flat-off arm, which stopped being
measurable once flat state was unlocked.

- It claimed the numbers were explained by the tree doing "~29 vs 1" lookups.
  That describes the *flat-off* trie walk. The headline row compared MPT-flat
  against PBT-flat-**on**, where both do a single flat lookup — the mechanism
  cited does not exist in the arm being measured.
- It claimed "a real disk widens the read gap". Adding a fixed I/O cost to both
  arms compresses a ratio toward 1.0; the direction was asserted, not derived.

**The real-database numbers are in the next section and they differ materially.**

## Real database — pebble, 1M accounts

Measured 2026-07-29, branch `pbt` @ `f30c047ac`. Apple M4 Pro, darwin/arm64,
Go 1.24. Harness `core/state/pbt_bench_test.go`: pebble on a temp directory,
1,000,000 accounts with two storage slots each (one header stem, one overflow
bucket), `Compact` before measuring so reads come from SSTables rather than the
memtable. Pebble's block cache cannot be disabled — `minCache` floors it at
16 MB — so the state is sized an order of magnitude past it. Setup takes minutes;
this is **not a CI check** and there is deliberately no `Test*` in the file.

| Benchmark | MPT | PBT | PBT / MPT |
|---|---|---|---|
| account read | 6166 ns/op, 1038 B, 16 allocs | 6599 ns/op, 1029 B, 17 allocs | **1.07×** |
| storage read | 12544 ns/op, 1891 B, 24 allocs | 12723 ns/op, 1979 B, 27 allocs | **1.01×** |
| commit (1000 mutations) | 94.6 ms/op, 14.8 MB, 137k allocs | 159.9 ms/op, 17.7 MB, 258k allocs | **1.69×** |

Reproduce:

```
go test ./core/state/ -run XXX -bench BenchmarkStateAccountRead -benchtime 2000x -timeout 90m
go test ./core/state/ -run XXX -bench BenchmarkStateStorageRead -benchtime 2000x -timeout 90m
go test ./core/state/ -run XXX -bench BenchmarkStateCommit      -benchtime 200x  -timeout 90m
```

**Reads are at parity; commit is not.** 1.07× and 1.01× are near enough to call
level. Commit at 1.69× is outside the ≤1.5× gate — and it is the one number
where the tree's shape shows up clearly: nearly twice the allocations for the
same thousand mutations, which is the wider node records being built and
written.

**What changed against the in-memory run.** Absolute read times are roughly six
times higher, because real I/O dominates what was previously a CPU measurement,
and the read ratio moved *against* the tree: from 0.97× (marginally ahead) to
1.07× (marginally behind). So the earlier "reaches MPT cold-read parity" claim
does not survive as stated — it is close to parity, not past it.

**The commit benchmark was also wrong, independently of the backend.** It opened
state from the same root every iteration, stacking `b.N` sibling layers on one
parent — a shape block processing never produces, and one where the layer tree
grows without ever flattening. Chaining each iteration onto the previous root
lets the database cap and flush on its normal schedule, which is why the
absolute cost tripled (28.8 ms → 94.6 ms for merkle) while the ratio fell
(1.84× → 1.69×). The earlier figure was measuring an artefact.

One caveat on that row: the harness sets `NoAsyncFlush` for determinism, so
flushes are synchronous where a real node overlaps them with execution. Both
arms pay it equally, so the ratio holds, but the absolute milliseconds are
pessimistic against production.

---

# Appendix: the superseded in-memory run (before the unlock)

Captured 2026-07-29 on branch `pbt` @ `35fe89ed4`, before flat state is enabled
for the binary tree. **The `PBT flat-off` column stops existing once the
generation marker no longer blocks flat state**, which is why it is recorded
here rather than measured later.

Harness: `core/state/pbt_bench_test.go`. Reproduce with

```
go test ./core/state/ -run XXX -bench 'BenchmarkStateAccountRead|BenchmarkStateStorageRead' -benchtime 3000x -count=3
go test ./core/state/ -run XXX -bench 'BenchmarkStateCommit' -benchtime 20x -count=3
```

Machine: Apple M4 Pro, darwin/arm64, Go 1.24.

## Numbers

Medians of three runs. 20,000 accounts, two storage slots each (one in the
binary tree's header stem, one in its overflow bucket).

| Benchmark | MPT | PBT flat-off | PBT flat-on | flat-on vs MPT |
|---|---|---|---|---|
| account read | 1020 ns/op, 911 B, 14 allocs | 6281 ns/op, 5147 B, 79 allocs | 992 ns/op, 847 B, 13 allocs | **0.97×** |
| storage read | 2186 ns/op, 1615 B, 20 allocs | 10579 ns/op, 7527 B, 125 allocs | 2065 ns/op, 1551 B, 19 allocs | **0.94×** |
| commit (1000 mutations) | 6.46 ms/op, 8.09 MB, 86.2k allocs | 12.95 ms/op, 11.96 MB, 188.7k allocs | 8.28 ms/op, 8.95 MB, 138.1k allocs | **1.28×** |

Flat-on measured after the unlock, same harness and machine.

## Result against the gate

| Gate | Target | Measured | |
|---|---|---|---|
| account read | ≥ MPT parity | 0.97× | met |
| storage read | ≥ MPT parity | 0.94× | met |
| commit | ≤ 1.5× MPT | 1.28× | met |

Reads improved 6.3× and 5.1×, landing marginally ahead of the merkle trie on
both time and allocations. Commit improved 1.6× — it was outside the gate at
2.0× before this work and is inside it now, which is the read cost coming out
of the commit path along with everything else.

## Reading these honestly

**What they measure.** Lookup count and CPU, not device I/O. Both arms run over
an in-memory key-value store with deliberately tiny clean caches (1 KB each), so
reads reach the trie rather than being absorbed by RAM. A real disk would widen
the read gap, not narrow it — the binary tree's extra cost is extra lookups.

**Why the read gap is larger than the ~1.7× in `ANALYSIS.md`.** That figure came
from an external bare-metal run with production-sized caches and a different
workload, where many reads never reached the trie at all. Here almost every read
does, which is the comparison the unlock is meant to change. The two numbers are
not in conflict; they bound the same effect from different ends.

**The commit number is read-inflated.** `BenchmarkStateCommit` mutates accounts,
which requires loading them first — so the binary tree's slow read path is
inside the measurement. That is realistic (block processing reads before it
writes), but it means the 2.0× is not purely commit-side cost, and flat state
should improve this column too rather than only the read ones.

**Allocation counts are the clearest signal.** 14 → 79 allocations for a single
account read is the trie walk showing up directly, and it is the part flat state
removes.

## Gate

From the plan: PBT with flat state on should reach MPT cold-read parity, and
commit should stay within 1.5× MPT. Note the commit arm is **already at 2.0×**
before any of this work, so either flat state brings it inside the gate or the
gate needs restating against a measurement rather than an aspiration.
