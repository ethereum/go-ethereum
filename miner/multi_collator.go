type collatorWork {
    env *environment
    counter uint64
}

type collator struct {
    newWorkCh chan<- collatorWork
    workResultCh chan<-collatorWork
    // channel signalling collator loop should exit
    exitCh chan<-interface{}
    newHeadCh chan<-types.Header
    collatorImpl Collator
}

func (c *collator) mainLoop() {
    for {
    case newWork := <-c.newWorkCh:
        // pass a wrapped CollatorBlockState object to the collator
        // implementation.  collator calls Commit() to flush new work to the
        // result channel.
        // TODO...

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

func (m *MultiCollator) CollateBlock(work *environment) {
    m.responsiveCollatorCount = 0
    for c := range m.collators {
        select {
        case c.newWorkCh <- collatorWork{env: work.copy(), counter: m.counter}
            m.responsiveCollatorCount++
        }
    }
}


func (m *MultiCollator) collect(work *environment, cb workResult) {
    finishedCollators := 0
    for ; finishedCollators != m.responsiveCollatorCount; {
        for c := range m.collators {
            select {
            case response := m.workResultCh:
                // ignore collators responding from old work rounds
                if response.counter != m.counter {
                    continue
                }

                // ignore responses from collators that have already signalled they are done
                // TODO

                cb(response.work)
            default:
                continue
            }
        }
    }
}

// Collect sends the work request to all collators and blocks until all have 
// sent their result, or times out after x seconds
func (m *MultiCollator) Collect(work *environment) {

}

// StreamAndCommit sends the work request to all collator goroutines.
// It collects the results and calls worker.commit() on them as they come back
func (m *MultiCollator) StreamAndCommit(w *worker, work *environment) {

}

func (m *MultiCollator) NewHeadHook() {

}

func (m *Multicollator) SideChainHook() {

}
