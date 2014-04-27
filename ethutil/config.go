package ethutil

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
)

// Log types available
type LogType byte

const (
	LogTypeStdIn = 1
	LogTypeFile  = 2
)

// Config struct
type config struct {
	Db Database

	Log          *Logger
	ExecPath     string
	Debug        bool
	Ver          string
	ClientString string
	Pubkey       []byte
	Seed         bool
}

var Config *config

// Read config
//
// Initialize the global Config variable with default settings
func ReadConfig(base string) *config {
	if Config == nil {
		usr, _ := user.Current()
		path := path.Join(usr.HomeDir, base)

		if len(base) > 0 {
			//Check if the logging directory already exists, create it if not
			_, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					log.Printf("Debug logging directory %s doesn't exist, creating it\n", path)
					os.Mkdir(path, 0777)
				}
			}
		}

		Config = &config{ExecPath: path, Debug: true, Ver: "0.5"}
		Config.Log = NewLogger(LogFile|LogStd, LogLevelDebug)
		Config.SetClientString("/Ethereum(G)")
	}

	return Config
}

// Set client string
//
func (c *config) SetClientString(str string) {
	Config.ClientString = fmt.Sprintf("%s nv%s/%s", str, c.Ver, runtime.GOOS)
}

type LoggerType byte

const (
	LogFile = 0x1
	LogStd  = 0x2
)

type LogSystem interface {
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

type Logger struct {
	logSys   []LogSystem
	logLevel int
}

func NewLogger(flag LoggerType, level int) *Logger {
	var loggers []LogSystem

	flags := log.LstdFlags

	if flag&LogFile > 0 {
		file, err := os.OpenFile(path.Join(Config.ExecPath, "debug.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			log.Panic("unable to create file logger", err)
		}

		log := log.New(file, "", flags)

		loggers = append(loggers, log)
	}
	if flag&LogStd > 0 {
		log := log.New(os.Stdout, "", flags)
		loggers = append(loggers, log)
	}

	return &Logger{logSys: loggers, logLevel: level}
}

func (log *Logger) AddLogSystem(logger LogSystem) {
	log.logSys = append(log.logSys, logger)
}

const (
	LogLevelDebug = iota
	LogLevelInfo
)

func (log *Logger) Debugln(v ...interface{}) {
	if log.logLevel != LogLevelDebug {
		return
	}

	for _, logger := range log.logSys {
		logger.Println(v...)
	}
}

func (log *Logger) Debugf(format string, v ...interface{}) {
	if log.logLevel != LogLevelDebug {
		return
	}

	for _, logger := range log.logSys {
		logger.Printf(format, v...)
	}
}

func (log *Logger) Infoln(v ...interface{}) {
	if log.logLevel > LogLevelInfo {
		return
	}

	fmt.Println(len(log.logSys))
	for _, logger := range log.logSys {
		logger.Println(v...)
	}
}

func (log *Logger) Infof(format string, v ...interface{}) {
	if log.logLevel > LogLevelInfo {
		return
	}

	for _, logger := range log.logSys {
		logger.Printf(format, v...)
	}
}
