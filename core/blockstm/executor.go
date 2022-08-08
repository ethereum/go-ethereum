package blockstm

import (
	"container/heap"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ExecResult struct {
	err      error
	ver      Version
	txIn     TxnInput
	txOut    TxnOutput
	txAllOut TxnOutput
}

type ExecTask interface {
	Execute(mvh *MVHashMap, incarnation int) error
	MVReadList() []ReadDescriptor
	MVWriteList() []WriteDescriptor
	MVFullWriteList() []WriteDescriptor
	Sender() common.Address
	Settle()
}

type ExecVersionView struct {
	ver    Version
	et     ExecTask
	mvh    *MVHashMap
	sender common.Address
}

func (ev *ExecVersionView) Execute() (er ExecResult) {
	er.ver = ev.ver
	if er.err = ev.et.Execute(ev.mvh, ev.ver.Incarnation); er.err != nil {
		return
	}

	er.txIn = ev.et.MVReadList()
	er.txOut = ev.et.MVWriteList()
	er.txAllOut = ev.et.MVFullWriteList()

	return
}

type ErrExecAbortError struct {
	Dependency int
}

func (e ErrExecAbortError) Error() string {
	if e.Dependency >= 0 {
		return fmt.Sprintf("Execution aborted due to dependency %d", e.Dependency)
	} else {
		return "Execution aborted"
	}
}

type IntHeap []int

func (h IntHeap) Len() int           { return len(h) }
func (h IntHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h IntHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *IntHeap) Push(x any) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(int))
}

func (h *IntHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]

	return x
}

// A thread safe priority queue
type SafePriorityQueue struct {
	m     sync.Mutex
	queue *IntHeap
	data  map[int]interface{}
}

func NewSafePriorityQueue(capacity int) *SafePriorityQueue {
	q := make(IntHeap, 0, capacity)

	return &SafePriorityQueue{
		m:     sync.Mutex{},
		queue: &q,
		data:  make(map[int]interface{}, capacity),
	}
}

func (pq *SafePriorityQueue) Push(v int, d interface{}) {
	pq.m.Lock()

	heap.Push(pq.queue, v)
	pq.data[v] = d

	pq.m.Unlock()
}

func (pq *SafePriorityQueue) Pop() interface{} {
	pq.m.Lock()
	defer pq.m.Unlock()

	v := heap.Pop(pq.queue).(int)

	return pq.data[v]
}

func (pq *SafePriorityQueue) Len() int {
	return pq.queue.Len()
}

type ParallelExecutionResult struct {
	TxIO  *TxnInputOutput
	Stats *[][]uint64
	Deps  *DAG
}

const numGoProcs = 2
const numSpeculativeProcs = 16

// Max number of pre-validation to run per loop
const preValidateLimit = 5

// Max number of times a transaction (t) can be executed before its dependency is resolved to its previous tx (t-1)
const maxIncarnation = 2

// nolint: gocognit
// A stateless executor that executes transactions in parallel
func ExecuteParallel(tasks []ExecTask, profile bool) (ParallelExecutionResult, error) {
	if len(tasks) == 0 {
		return ParallelExecutionResult{MakeTxnInputOutput(len(tasks)), nil, nil}, nil
	}

	// Stores the execution statistics for each task
	stats := make([][]uint64, 0, len(tasks))
	statsMutex := sync.Mutex{}

	// Channel for tasks that should be prioritized
	chTasks := make(chan ExecVersionView, len(tasks))

	// Channel for speculative tasks
	chSpeculativeTasks := make(chan struct{}, len(tasks))

	// A priority queue that stores speculative tasks
	specTaskQueue := NewSafePriorityQueue(len(tasks))

	// Channel to signal that the result of a transaction could be written to storage
	chSettle := make(chan int, len(tasks))

	// Channel to signal that a transaction has finished executing
	chResults := make(chan struct{}, len(tasks))

	// A priority queue that stores the transaction index of results, so we can validate the results in order
	resultQueue := NewSafePriorityQueue(len(tasks))

	// A wait group to wait for all settling tasks to finish
	var settleWg sync.WaitGroup

	// An integer that tracks the index of last settled transaction
	lastSettled := -1

	// For a task that runs only after all of its preceding tasks have finished and passed validation,
	// its result will be absolutely valid and therefore its validation could be skipped.
	// This map stores the boolean value indicating whether a task satisfy this condition ( absolutely valid).
	skipCheck := make(map[int]bool)

	for i := 0; i < len(tasks); i++ {
		skipCheck[i] = false
	}

	// Execution tasks stores the state of each execution task
	execTasks := makeStatusManager(len(tasks))

	// Validate tasks stores the state of each validation task
	validateTasks := makeStatusManager(0)

	// Stats for debugging purposes
	var cntExec, cntSuccess, cntAbort, cntTotalValidations, cntValidationFail int

	diagExecSuccess := make([]int, len(tasks))
	diagExecAbort := make([]int, len(tasks))

	// Initialize MVHashMap
	mvh := MakeMVHashMap()

	// Stores the inputs and outputs of the last incardanotion of all transactions
	lastTxIO := MakeTxnInputOutput(len(tasks))

	// Tracks the incarnation number of each transaction
	txIncarnations := make([]int, len(tasks))

	// A map that stores the estimated dependency of a transaction if it is aborted without any known dependency
	estimateDeps := make(map[int][]int, len(tasks))

	for i := 0; i < len(tasks); i++ {
		estimateDeps[i] = make([]int, 0)
	}

	// A map that records whether a transaction result has been speculatively validated
	preValidated := make(map[int]bool, len(tasks))

	begin := time.Now()

	workerWg := sync.WaitGroup{}
	workerWg.Add(numSpeculativeProcs + numGoProcs)

	// Launch workers that execute transactions
	for i := 0; i < numSpeculativeProcs+numGoProcs; i++ {
		go func(procNum int) {
			defer workerWg.Done()

			doWork := func(task ExecVersionView) {
				start := time.Duration(0)
				if profile {
					start = time.Since(begin)
				}

				res := task.Execute()

				if res.err == nil {
					mvh.FlushMVWriteSet(res.txAllOut)
				}

				resultQueue.Push(res.ver.TxnIndex, res)
				chResults <- struct{}{}

				if profile {
					end := time.Since(begin)

					stat := []uint64{uint64(res.ver.TxnIndex), uint64(res.ver.Incarnation), uint64(start), uint64(end), uint64(procNum)}

					statsMutex.Lock()
					stats = append(stats, stat)
					statsMutex.Unlock()
				}
			}

			if procNum < numSpeculativeProcs {
				for range chSpeculativeTasks {
					doWork(specTaskQueue.Pop().(ExecVersionView))
				}
			} else {
				for task := range chTasks {
					doWork(task)
				}
			}
		}(i)
	}

	// Launch a worker that settles valid transactions
	settleWg.Add(len(tasks))

	go func() {
		for t := range chSettle {
			tasks[t].Settle()
			settleWg.Done()
		}
	}()

	// bootstrap first execution
	tx := execTasks.takeNextPending()
	if tx != -1 {
		cntExec++

		chTasks <- ExecVersionView{ver: Version{tx, 0}, et: tasks[tx], mvh: mvh, sender: tasks[tx].Sender()}
	}

	// Before starting execution, going through each task to check their explicit dependencies (whether they are coming from the same account)
	prevSenderTx := make(map[common.Address]int)

	for i, t := range tasks {
		if tx, ok := prevSenderTx[t.Sender()]; ok {
			execTasks.addDependencies(tx, i)
			execTasks.clearPending(i)
		}

		prevSenderTx[t.Sender()] = i
	}

	var res ExecResult

	var err error

	// Start main validation loop
	// nolint:nestif
	for range chResults {
		res = resultQueue.Pop().(ExecResult)
		tx := res.ver.TxnIndex

		if res.err == nil {
			lastTxIO.recordRead(tx, res.txIn)

			if res.ver.Incarnation == 0 {
				lastTxIO.recordWrite(tx, res.txOut)
				lastTxIO.recordAllWrite(tx, res.txAllOut)
			} else {
				if res.txAllOut.hasNewWrite(lastTxIO.AllWriteSet(tx)) {
					validateTasks.pushPendingSet(execTasks.getRevalidationRange(tx + 1))
				}

				prevWrite := lastTxIO.AllWriteSet(tx)

				// Remove entries that were previously written but are no longer written

				cmpMap := make(map[Key]bool)

				for _, w := range res.txAllOut {
					cmpMap[w.Path] = true
				}

				for _, v := range prevWrite {
					if _, ok := cmpMap[v.Path]; !ok {
						mvh.Delete(v.Path, tx)
					}
				}

				lastTxIO.recordWrite(tx, res.txOut)
				lastTxIO.recordAllWrite(tx, res.txAllOut)
			}

			validateTasks.pushPending(tx)
			execTasks.markComplete(tx)
			diagExecSuccess[tx]++
			cntSuccess++

			execTasks.removeDependency(tx)
		} else if execErr, ok := res.err.(ErrExecAbortError); ok {

			addedDependencies := false

			if execErr.Dependency >= 0 {
				l := len(estimateDeps[tx])
				for l > 0 && estimateDeps[tx][l-1] > execErr.Dependency {
					execTasks.removeDependency(estimateDeps[tx][l-1])
					estimateDeps[tx] = estimateDeps[tx][:l-1]
					l--
				}
				if txIncarnations[tx] < maxIncarnation {
					addedDependencies = execTasks.addDependencies(execErr.Dependency, tx)
				} else {
					addedDependencies = execTasks.addDependencies(tx-1, tx)
				}
			} else {
				estimate := 0

				if len(estimateDeps[tx]) > 0 {
					estimate = estimateDeps[tx][len(estimateDeps[tx])-1]
				}
				addedDependencies = execTasks.addDependencies(estimate, tx)
				newEstimate := estimate + (estimate+tx)/2
				if newEstimate >= tx {
					newEstimate = tx - 1
				}
				estimateDeps[tx] = append(estimateDeps[tx], newEstimate)
			}

			execTasks.clearInProgress(tx)
			if !addedDependencies {
				execTasks.pushPending(tx)
			}
			txIncarnations[tx]++
			diagExecAbort[tx]++
			cntAbort++
		} else {
			err = res.err
			break
		}

		// do validations ...
		maxComplete := execTasks.maxAllComplete()

		var toValidate []int

		for validateTasks.minPending() <= maxComplete && validateTasks.minPending() >= 0 {
			toValidate = append(toValidate, validateTasks.takeNextPending())
		}

		for i := 0; i < len(toValidate); i++ {
			cntTotalValidations++

			tx := toValidate[i]

			if skipCheck[tx] || ValidateVersion(tx, lastTxIO, mvh) {
				validateTasks.markComplete(tx)
			} else {
				cntValidationFail++
				diagExecAbort[tx]++
				for _, v := range lastTxIO.AllWriteSet(tx) {
					mvh.MarkEstimate(v.Path, tx)
				}
				// 'create validation tasks for all transactions > tx ...'
				validateTasks.pushPendingSet(execTasks.getRevalidationRange(tx + 1))
				validateTasks.clearInProgress(tx) // clear in progress - pending will be added again once new incarnation executes

				addedDependencies := false
				if txIncarnations[tx] >= maxIncarnation {
					addedDependencies = execTasks.addDependencies(tx-1, tx)
				}

				execTasks.clearComplete(tx)
				if !addedDependencies {
					execTasks.pushPending(tx)
				}

				preValidated[tx] = false
				txIncarnations[tx]++
			}
		}

		preValidateCount := 0
		invalidated := []int{}

		i := sort.SearchInts(validateTasks.pending, maxComplete+1)

		for i < len(validateTasks.pending) && preValidateCount < preValidateLimit {
			tx := validateTasks.pending[i]

			if !preValidated[tx] {
				cntTotalValidations++

				if !ValidateVersion(tx, lastTxIO, mvh) {
					cntValidationFail++
					diagExecAbort[tx]++

					invalidated = append(invalidated, tx)

					if execTasks.checkComplete(tx) {
						execTasks.clearComplete(tx)
					}

					if !execTasks.checkInProgress(tx) {
						for _, v := range lastTxIO.AllWriteSet(tx) {
							mvh.MarkEstimate(v.Path, tx)
						}

						validateTasks.pushPendingSet(execTasks.getRevalidationRange(tx + 1))

						addedDependencies := false
						if txIncarnations[tx] >= maxIncarnation {
							addedDependencies = execTasks.addDependencies(tx-1, tx)
						}

						if !addedDependencies {
							execTasks.pushPending(tx)
						}
					}

					txIncarnations[tx]++

					preValidated[tx] = false
				} else {
					preValidated[tx] = true
				}
				preValidateCount++
			}

			i++
		}

		for _, tx := range invalidated {
			validateTasks.clearPending(tx)
		}

		// Settle transactions that have been validated to be correct and that won't be re-executed again
		maxValidated := validateTasks.maxAllComplete()

		for lastSettled < maxValidated {
			lastSettled++
			if execTasks.checkInProgress(lastSettled) || execTasks.checkPending(lastSettled) || execTasks.blockCount[lastSettled] >= 0 {
				lastSettled--
				break
			}
			chSettle <- lastSettled
		}

		if validateTasks.countComplete() == len(tasks) && execTasks.countComplete() == len(tasks) {
			log.Debug("blockstm exec summary", "execs", cntExec, "success", cntSuccess, "aborts", cntAbort, "validations", cntTotalValidations, "failures", cntValidationFail, "#tasks/#execs", fmt.Sprintf("%.2f%%", float64(len(tasks))/float64(cntExec)*100))
			break
		}

		// Send the next immediate pending transaction to be executed
		if execTasks.minPending() != -1 && execTasks.minPending() == maxValidated+1 {
			nextTx := execTasks.takeNextPending()
			if nextTx != -1 {
				cntExec++

				skipCheck[nextTx] = true

				chTasks <- ExecVersionView{ver: Version{nextTx, txIncarnations[nextTx]}, et: tasks[nextTx], mvh: mvh, sender: tasks[nextTx].Sender()}
			}
		}

		// Send speculative tasks
		for execTasks.peekPendingGE(maxValidated+3) != -1 || len(execTasks.inProgress) == 0 {
			// We skip the next transaction to avoid the case where they all have conflicts and could not be
			// scheduled for re-execution immediately even when it's their time to run, because they are already in
			// speculative queue.
			nextTx := execTasks.takePendingGE(maxValidated + 3)

			if nextTx == -1 {
				nextTx = execTasks.takeNextPending()
			}

			if nextTx != -1 {
				cntExec++

				task := ExecVersionView{ver: Version{nextTx, txIncarnations[nextTx]}, et: tasks[nextTx], mvh: mvh, sender: tasks[nextTx].Sender()}

				specTaskQueue.Push(nextTx, task)
				chSpeculativeTasks <- struct{}{}
			}
		}
	}

	close(chTasks)
	close(chSpeculativeTasks)
	workerWg.Wait()
	close(chResults)
	settleWg.Wait()
	close(chSettle)

	var dag DAG
	if profile {
		dag = BuildDAG(*lastTxIO)
	}

	return ParallelExecutionResult{lastTxIO, &stats, &dag}, err
}
