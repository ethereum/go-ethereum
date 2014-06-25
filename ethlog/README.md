## Features

- packages use tagged logger sending log messages to shared (process-wide) logging engine
- log writers (interface ethlog.LogSystem) can be added to the logging engine by wrappers/guis/clients
- shared logging engine dispatching to multiple log systems
- log level can be set separately per log system
- async logging thread: logging IO does not block main thread 
- log messages are synchronously stringified to avoid incorrectly logging of changed states 
- log level enum: ethlog.LogLevel:   Silence, ErrorLevel, WarnLevel, InfoLevel, DebugLevel, DebugDetailLevel

## Usage

In an ethereum component package:

    import "github.com/ethereum/eth-go/ethlog"

    // package-wide logger using tag
    var logger = ethlog.NewLogger("TAG")

Logger provides named Printf and Println style methods for all loglevels

    logger.Infoln("this is info") # > [TAG] This is info
    logger.Infof("this %v is info", object) # > [TAG] This object is info

Ethereum wrappers should register log systems conforming to ethlog.LogSystem

    import "github.com/ethereum/eth-go/ethlog"
    
    type CustomLogWriter struct {
      logLevel ethlog.LogLevel
    }

    func (t *TestLogSystem) SetLogLevel(i LogLevel) {
      t.level = i
    }

    func (t *TestLogSystem) GetLogLevel() LogLevel {
      return t.level
    }

    func (c *CustomLogWriter) Printf(format string, v...interface{}) {
      //....
    }

    func (c *CustomLogWriter) Println(v...interface{}) {
      //....
    }

    ethlog.AddLogWriter(&CustomLogWriter{})

ethlog also provides constructors for that wrap io.Writers into a standard logger with a settable level:

    filename := "test.log"
    file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
    fileLogSystem := NewStdLogSystem(file, 0, WarnLevel)
    AddLogSystem(fileLogSystem)
    stdOutLogSystem := NewStdLogSystem(os.Stdout, 0, WarnLevel)
    AddLogSystem(stdOutLogSystem)




