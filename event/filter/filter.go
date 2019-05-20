package filter

import "reflect"

type Filter interface {
	Compare(Filter) bool
	Trigger(data interface{})
}

type FilterEvent struct {
	filter Filter
	data   interface{}
}

type Filters struct {
	id       int
	watchers map[int]Filter
	ch       chan FilterEvent

	quit chan struct{}
}

func New() *Filters {
	return &Filters{
		ch:       make(chan FilterEvent),
		watchers: make(map[int]Filter),
		quit:     make(chan struct{}),
	}
}

func (self *Filters) Start() {
	go self.loop()
}

func (self *Filters) Stop() {
	close(self.quit)
}

func (self *Filters) Notify(filter Filter, data interface{}) {
	self.ch <- FilterEvent{filter, data}
}

func (self *Filters) Install(watcher Filter) int {
	self.watchers[self.id] = watcher
	self.id++

	return self.id - 1
}

func (self *Filters) Uninstall(id int) {
	delete(self.watchers, id)
}

func (self *Filters) loop() {
out:
	for {
		select {
		case <-self.quit:
			break out
		case event := <-self.ch:
			for _, watcher := range self.watchers {
				if reflect.TypeOf(watcher) == reflect.TypeOf(event.filter) {
					if watcher.Compare(event.filter) {
						watcher.Trigger(event.data)
					}
				}
			}
		}
	}
}

func (self *Filters) Match(a, b Filter) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(b) && a.Compare(b)
}

func (self *Filters) Get(i int) Filter {
	return self.watchers[i]
}
