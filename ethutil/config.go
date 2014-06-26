package ethutil

import (
	"flag"
	"fmt"
	"github.com/rakyll/globalconf"
	"os"
	"runtime"
)

// Config struct
type config struct {
	Db Database

	ExecPath     string
	Debug        bool
	Ver          string
	ClientString string
	Pubkey       []byte
	Identifier   string

	conf *globalconf.GlobalConf
}

var Config *config

// Read config
//
// Initialize Config from Config File
func ReadConfig(ConfigFile string, Datadir string, Identifier string, EnvPrefix string) *config {
	if Config == nil {
		// create ConfigFile if does not exist, otherwise globalconf panic when trying to persist flags
		_, err := os.Stat(ConfigFile)
		if err != nil && os.IsNotExist(err) {
			fmt.Printf("config file '%s' doesn't exist, creating it\n", ConfigFile)
			os.Create(ConfigFile)
		}
		g, err := globalconf.NewWithOptions(&globalconf.Options{
			Filename:  ConfigFile,
			EnvPrefix: EnvPrefix,
		})
		if err != nil {
			fmt.Println(err)
		} else {
			g.ParseAll()
		}
		Config = &config{ExecPath: Datadir, Debug: true, Ver: "0.5.15", conf: g, Identifier: Identifier}
		Config.SetClientString("Ethereum(G)")
	}
	return Config
}

// Set client string
//
func (c *config) SetClientString(str string) {
	os := runtime.GOOS
	cust := c.Identifier
	Config.ClientString = fmt.Sprintf("%s/v%s/%s/%s/Go", str, c.Ver, cust, os)
}

func (c *config) SetIdentifier(id string) {
	c.Identifier = id
	c.Set("id", id)
}

// provides persistence for flags
func (c *config) Set(key, value string) {
	f := &flag.Flag{Name: key, Value: &confValue{value}}
	c.conf.Set("", f)
}

type confValue struct {
	value string
}

func (self confValue) String() string     { return self.value }
func (self confValue) Set(s string) error { self.value = s; return nil }
