package common

import (
	"flag"
	"fmt"
	"os"

	"github.com/rakyll/globalconf"
)

// Config struct
type ConfigManager struct {
	ExecPath string
	Debug    bool
	Diff     bool
	DiffType string
	Paranoia bool
	VmType   int

	conf *globalconf.GlobalConf
}

// Read config
//
// Initialize Config from Config File
func ReadConfig(ConfigFile string, Datadir string, EnvPrefix string) *ConfigManager {
	if !FileExist(ConfigFile) {
		// create ConfigFile if it does not exist, otherwise
		// globalconf will panic when trying to persist flags.
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
	cfg := &ConfigManager{ExecPath: Datadir, Debug: true, conf: g, Paranoia: true}
	return cfg
}

// provides persistence for flags
func (c *ConfigManager) Save(key string, value interface{}) {
	f := &flag.Flag{Name: key, Value: newConfValue(value)}
	c.conf.Set("", f)
}

func (c *ConfigManager) Delete(key string) {
	c.conf.Delete("", key)
}

// private type implementing flag.Value
type confValue struct {
	value string
}

// generic constructor to allow persising non-string values directly
func newConfValue(value interface{}) *confValue {
	return &confValue{fmt.Sprintf("%v", value)}
}

func (self confValue) String() string     { return self.value }
func (self confValue) Set(s string) error { self.value = s; return nil }
