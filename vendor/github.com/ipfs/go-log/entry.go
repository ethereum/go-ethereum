package log

type entry struct {
	loggables []Loggable
	system    string
	event     string
}
