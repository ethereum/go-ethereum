package ethutil

import (
	"log"
	"os"
	"os/user"
	"path"
)

type LogType byte

const (
	LogTypeStdIn = 1
	LogTypeFile  = 2
)

// Config struct isn't exposed
type config struct {
	Db Database

	Log      Logger
	ExecPath string
	Debug    bool
	Ver      string
	Pubkey   []byte
	Seed     bool
}

var Config *config

// Read config doesn't read anything yet.
func ReadConfig(base string) *config {
	if Config == nil {
		usr, _ := user.Current()
		path := path.Join(usr.HomeDir, base)

		//Check if the logging directory already exists, create it if not
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Debug logging directory %s doesn't exist, creating it", path)
				os.Mkdir(path, 0777)
			}
		}

		Config = &config{ExecPath: path, Debug: true, Ver: "0.2.2"}
		Config.Log = NewLogger(LogFile|LogStd, 0)
	}

	return Config
}

type LoggerType byte

const (
	LogFile = 0x1
	LogStd  = 0x2
)

type Logger struct {
	logSys   []*log.Logger
	logLevel int
}

func NewLogger(flag LoggerType, level int) Logger {
	var loggers []*log.Logger

	flags := log.LstdFlags | log.Lshortfile

	if flag&LogFile > 0 {
		file, err := os.OpenFile(path.Join(Config.ExecPath, "debug.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			log.Panic("unable to create file logger", err)
		}

		log := log.New(file, "[ETH]", flags)

		loggers = append(loggers, log)
	}
	if flag&LogStd > 0 {
		log := log.New(os.Stdout, "[ETH]", flags)
		loggers = append(loggers, log)
	}

	return Logger{logSys: loggers, logLevel: level}
}

func (log Logger) Debugln(v ...interface{}) {
	if log.logLevel != 0 {
		return
	}

	for _, logger := range log.logSys {
		logger.Println(v...)
	}
}

func (log Logger) Debugf(format string, v ...interface{}) {
	if log.logLevel != 0 {
		return
	}

	for _, logger := range log.logSys {
		logger.Printf(format, v...)
	}
}
