package ethutil

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path"
	"runtime"
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
	Identifier   string
}

const defaultConf = `
id = ""
port = 30303
upnp = true
maxpeer = 10
rpc = false
rpcport = 8080
`

var Config *config

func ApplicationFolder(base string) string {
	usr, _ := user.Current()
	p := path.Join(usr.HomeDir, base)

	if len(base) > 0 {
		//Check if the logging directory already exists, create it if not
		_, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("Debug logging directory %s doesn't exist, creating it\n", p)
				os.Mkdir(p, 0777)

			}
		}

		iniFilePath := path.Join(p, "conf.ini")
		_, err = os.Stat(iniFilePath)
		if err != nil && os.IsNotExist(err) {
			file, err := os.Create(iniFilePath)
			if err != nil {
				fmt.Println(err)
			} else {
				assetPath := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ethereum", "go-ethereum", "ethereal", "assets")
				file.Write([]byte(defaultConf + "\nasset_path = " + assetPath))
			}
		}
	}

	return p
}

// Read config
//
// Initialize the global Config variable with default settings
func ReadConfig(base string, logTypes LoggerType, id string) *config {
	if Config == nil {
		path := ApplicationFolder(base)

		Config = &config{ExecPath: path, Debug: true, Ver: "0.5.0 RC11"}
		Config.Identifier = id
		Config.Log = NewLogger(logTypes, LogLevelDebug)
		Config.SetClientString("/Ethereum(G)")
	}

	return Config
}

// Set client string
//
func (c *config) SetClientString(str string) {
	id := runtime.GOOS
	if len(c.Identifier) > 0 {
		id = c.Identifier
	}
	Config.ClientString = fmt.Sprintf("%s nv%s/%s", str, c.Ver, id)
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

func (log *Logger) Fatal(v ...interface{}) {
	if log.logLevel > LogLevelInfo {
		return
	}

	for _, logger := range log.logSys {
		logger.Println(v...)
	}

	os.Exit(1)
}
