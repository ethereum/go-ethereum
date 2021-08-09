type collatorWork {
    env *environment
    counter uint64
    interrupt *int32
}

type collatorBlockState struct {
    work collatorWork
    c *collator
}

func (w *collatorWork) Copy() collatorWork {
    newEnv := w.copy()
    return collatorWork{
        env: newEnv,
        counter: w.counter,
        interrupt: w.interrupt,
    }
}

func (b *collatorBlockState) AddTransactions() {

}

func (bs *collatorBlockState) Commit() {
    bs.c.workResultch <- bs.work
}

type collator struct {
    newWorkCh chan<- collatorWork
    workResultCh chan<-collatorWork
    // channel signalling collator loop should exit
    exitCh chan<-interface{}
    newHeadCh chan<-types.Header
    collateBlockImpl BlockCollator
}

func (c *collator) mainLoop() {
    for {
    case newWork := <-c.newWorkCh:
        // pass a wrapped CollatorBlockState object to the collator
        // implementation.  collator calls Commit() to flush new work to the
        // result channel.
        // TODO...
        c.collateBlockImpl(collatorBlockState{newWork, c})

        // signal to the exitCh that the collator is done
        // computing this work.
        // TODO...
        c.workResultCh <- collatorWork{nil, newWork.counter}
        fallthrough
    case newHead := <-newHeadCh:
        fallthrough
    case <-c.exitCh:
        // TODO any cleanup needed?
        break
    default:
        fallthrough
    }
}

type MultiCollator struct {
    workResultCh
    counter uint64
    responsiveCollatorCount uint
    collators []collator
}

func (m *MultiCollator) Start() {
    for c := range m.collators {
        go c.mainLoop()
    }
}

func (m *MultiCollator) Stop() {
    for c := range m.collators {
        select {
        case c.exitCh
        }
    }
}

func (m *MultiCollator) CollateBlock(work *environment, interrupt *int32) {
    if m.counter == math.Uint64Max {
        m.counter = 0
    } else {
        m.counter++
    }
    m.responsiveCollatorCount = 0
    m.interrupt = interrupt
    for c := range m.collators {
        select {
        case c.newWorkCh <- collatorWork{env: work.copy(), counter: m.counter, interrupt: interrupt}
            m.responsiveCollatorCount++
        }
    }
}


func (m *MultiCollator) Collect(work *environment, cb workResult) {
    finishedCollators := []uint{}
    shouldAdjustRecommitDown := true
    for {
        if finishedCollators == m.responsiveCollatorcount {
            break
        }

        if interrupt != nil && atomic.LoadInt32(interrupt) != commitInterruptNone {
            // TODO: if the interrupt was recommit, signal to the worker to adjust up
            shouldAdjustRecommitDown = false
        }

        for i, c := range m.collators {
            select {
            case response := m.workResultCh:
                // ignore collators responding from old work rounds
                if response.counter != m.counter {
                    continue
                }

                // ignore responses from collators that have already signalled they are done
                shouldIgnore := false
                for _, finishedCollator := range finishedCollators {
                    if i == finishedCollator {
                        shouldIgnore = true
                    }
                }
                if shouldIgnore {
                    break
                }

                // nil for work signals the collator won't send back any more blocks for this round
                if response.work == nil {
                    finishedCollators = append(finishedCollators, i)
                } else {
                    cb(response.work)
                }
            default:
                continue
            }
        }
    }

    if shouldAdjustRecommitDown {
        // TODO signal to worker to adjust recommit interval down
    }
}

func (m *MultiCollator) NewHeadHook() {

}

func (m *Multicollator) SideChainHook() {

}
