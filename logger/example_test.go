package logger

import "os"

func ExampleLogger() {
	logger := NewLogger("TAG")
	logger.Infoln("so awesome")            // prints [TAG] so awesome
	logger.Infof("this %q is raw", "coin") // prints [TAG] this "coin" is raw
}

func ExampleLogSystem() {
	filename := "test.log"
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	fileLog := NewStdLogSystem(file, 0, WarnLevel)
	AddLogSystem(fileLog)

	stdoutLog := NewStdLogSystem(os.Stdout, 0, WarnLevel)
	AddLogSystem(stdoutLog)

	NewLogger("TAG").Warnln("reactor meltdown") // writes to both logs
}
