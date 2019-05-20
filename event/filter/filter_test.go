package filter

import "testing"

func TestFilters(t *testing.T) {
	var success bool
	var failure bool

	fm := New()
	fm.Start()
	fm.Install(Generic{
		Str1: "hello",
		Fn: func(data interface{}) {
			success = data.(bool)
		},
	})
	fm.Install(Generic{
		Str1: "hello1",
		Str2: "hello",
		Fn: func(data interface{}) {
			failure = true
		},
	})
	fm.Notify(Generic{Str1: "hello"}, true)
	fm.Stop()

	if !success {
		t.Error("expected 'hello' to be posted")
	}

	if failure {
		t.Error("hello1 was triggered")
	}
}
