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

func TestLoggerPrintln(t *testing.T) {
	logger := NewLogger("TEST")
	testLogSystem := &TestLogSystem{level: WarnLevel}
	AddLogSystem(testLogSystem)
	logger.Errorln("error")
	logger.Warnln("warn")
	logger.Infoln("info")
	logger.Debugln("debug")
	Flush()
	Reset()
	output := testLogSystem.Output
	if output != "[TEST] error\n[TEST] warn\n" {
		t.Error("Expected logger output '[TEST] error\\n[TEST] warn\\n', got ", testLogSystem.Output)
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
	Reset()
	output := testLogSystem.Output
	if output != "[TEST] error to { 2}\n[TEST] warn" {
		t.Error("Expected logger output '[TEST] error to { 2}\\n[TEST] warn', got ", testLogSystem.Output)
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
	Reset()
	output0 := testLogSystem0.Output
	output1 := testLogSystem1.Output
	if output0 != "[TEST] error\n" {
		t.Error("Expected logger 0 output '[TEST] error\\n', got ", testLogSystem0.Output)
	}
	if output1 != "[TEST] error\n[TEST] warn\n" {
		t.Error("Expected logger 1 output '[TEST] error\\n[TEST] warn\\n', got ", testLogSystem1.Output)
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
	Reset()
	contents, _ := ioutil.ReadFile(filename)
	output := string(contents)
	if output != "[TEST] error to test.log\n[TEST] warn\n" {
		t.Error("Expected contents of file 'test.log': '[TEST] error to test.log\\n[TEST] warn\\n', got ", output)
	} else {
		os.Remove(filename)
	}
}

func TestNoLogSystem(t *testing.T) {
	logger := NewLogger("TEST")
	logger.Warnln("warn")
	fmt.Println("1")
	Flush()
	Reset()
}
