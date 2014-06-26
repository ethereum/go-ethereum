package ethlog

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

type TestLogSystem struct {
	Output string
	level  LogLevel
}

func (t *TestLogSystem) Println(v ...interface{}) {
	t.Output += fmt.Sprintln(v...)
}

func (t *TestLogSystem) Printf(format string, v ...interface{}) {
	t.Output += fmt.Sprintf(format, v...)
}

func (t *TestLogSystem) SetLogLevel(i LogLevel) {
	t.level = i
}

func (t *TestLogSystem) GetLogLevel() LogLevel {
	return t.level
}

func quote(s string) string {
	return fmt.Sprintf("'%s'", s)
}

func TestLoggerPrintln(t *testing.T) {
	logger := NewLogger("TEST")
	testLogSystem := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem)
	logger.Errorln("error")
	logger.Warnln("warn")
	logger.Infoln("info")
	logger.Debugln("debug")
	Flush()
	output := testLogSystem.Output
	fmt.Println(quote(output))
	if output != "[TEST] error\n[TEST] warn\n" {
		t.Error("Expected logger output '[TEST] error\\n[TEST] warn\\n', got ", quote(testLogSystem.Output))
	}
}

func TestLoggerPrintf(t *testing.T) {
	logger := NewLogger("TEST")
	testLogSystem := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem)
	logger.Errorf("error to %v\n", *testLogSystem)
	logger.Warnf("warn")
	logger.Infof("info")
	logger.Debugf("debug")
	Flush()
	output := testLogSystem.Output
	fmt.Println(quote(output))
	if output != "[TEST] error to { 2}\n[TEST] warn" {
		t.Error("Expected logger output '[TEST] error to { 2}\\n[TEST] warn', got ", quote(testLogSystem.Output))
	}
}

func TestMultipleLogSystems(t *testing.T) {
	logger := NewLogger("TEST")
	testLogSystem0 := &TestLogSystem{level: ErrorLevel}
	testLogSystem1 := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem0)
	AddLogSystem(testLogSystem1)
	logger.Errorln("error")
	logger.Warnln("warn")
	Flush()
	output0 := testLogSystem0.Output
	output1 := testLogSystem1.Output
	if output0 != "[TEST] error\n" {
		t.Error("Expected logger 0 output '[TEST] error\\n', got ", quote(testLogSystem0.Output))
	}
	if output1 != "[TEST] error\n[TEST] warn\n" {
		t.Error("Expected logger 1 output '[TEST] error\\n[TEST] warn\\n', got ", quote(testLogSystem1.Output))
	}
}

func TestFileLogSystem(t *testing.T) {
	logger := NewLogger("TEST")
	filename := "test.log"
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	testLogSystem := NewStdLogSystem(file, 0, WarnLevel)
	AddLogSystem(testLogSystem)
	logger.Errorf("error to %s\n", filename)
	logger.Warnln("warn")
	Flush()
	contents, _ := ioutil.ReadFile(filename)
	output := string(contents)
	fmt.Println(quote(output))
	if output != "[TEST] error to test.log\n[TEST] warn\n" {
		t.Error("Expected contents of file 'test.log': '[TEST] error to test.log\\n[TEST] warn\\n', got ", quote(output))
	} else {
		os.Remove(filename)
	}
}

func TestNoLogSystem(t *testing.T) {
	logger := NewLogger("TEST")
	logger.Warnln("warn")
	Flush()
}
