package pgeth

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"runtime/debug"

	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/mattn/go-colorable"
	"gopkg.in/yaml.v2"
)

type PluginToolkit struct {
	Node    *node.Node
	Backend ethapi.Backend
	Logger  log.Logger
}

type PluginDetails struct {
	Name   string                 `yaml:"name"`
	Config map[string]interface{} `yaml:"config"`
}

type Plugin struct {
	Details PluginDetails
	Start   func(*PluginToolkit, map[string]interface{}, context.Context, chan (error))
	Version func()
}

type PluginEngineConfig struct {
	Node    *node.Node
	Backend ethapi.Backend
}

type PluginEngine struct {
	node    *node.Node
	backend ethapi.Backend
	logger  log.Logger
}

func NewEngine(cfg *PluginEngineConfig) *PluginEngine {
	logger := log.New()

	var ostream log.Handler
	output := colorable.NewColorableStdout()
	ostream = log.StreamHandler(output, log.PgethFormat(true))
	logger.SetHandler(ostream)
	return &PluginEngine{
		node:    cfg.Node,
		backend: cfg.Backend,
		logger:  logger,
	}
}

func (p *PluginEngine) Version(ctx context.Context) error {
	pluginDirectory := os.Getenv("PGETH_DIRECTORY")

	if len(pluginDirectory) == 0 {
		p.logger.Warn("Skipping plugin engine startup: PGETH_DIRECTORY is empty")
		return nil
	}

	p.logger.Info("Starting plugin engine")

	files, err := ioutil.ReadDir(pluginDirectory)
	if err != nil {
		return err
	}

	plugins := []*Plugin{}

	for _, file := range files {
		if file.IsDir() {
			newPlugin, err := p.loadPluginDetailsFromDirectory(filepath.Join(pluginDirectory, file.Name()))
			if err != nil {
				return err
			}
			plugins = append(plugins, newPlugin)
		}
	}

	errChan := make(chan error)

	for _, plug := range plugins {
		plug.Version()
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errChan:
				p.logger.Error(err.Error())
			}
		}
	}()

	return nil
}

func (p *PluginEngine) Start(ctx context.Context) error {
	pluginDirectory := os.Getenv("PGETH_DIRECTORY")

	if len(pluginDirectory) == 0 {
		p.logger.Warn("Skipping plugin engine startup: PGETH_DIRECTORY is empty")
		return nil
	}

	p.logger.Info("Starting plugin engine")

	files, err := ioutil.ReadDir(pluginDirectory)
	if err != nil {
		return err
	}

	plugins := []*Plugin{}

	for _, file := range files {
		if file.IsDir() {
			newPlugin, err := p.loadPluginDetailsFromDirectory(filepath.Join(pluginDirectory, file.Name()))
			if err != nil {
				return err
			}
			plugins = append(plugins, newPlugin)
		}
	}

	var toolkit *PluginToolkit = &PluginToolkit{
		Node:    p.node,
		Backend: p.backend,
		Logger:  p.logger,
	}

	errChan := make(chan error)

	for _, plug := range plugins {
		go func(_plug *Plugin) {
			defer func() {
				if err := recover(); err != nil {
					p.logger.Error("Plugin crashed", "error", err, "plugin", _plug.Details.Name)
					// logs entire trace
					debug.PrintStack()
				}
			}()
			_plug.Start(toolkit, _plug.Details.Config, ctx, errChan)
		}(plug)
		p.logger.Info(fmt.Sprintf("Starting \"%s\" plugin", plug.Details.Name))
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-errChan:
				p.logger.Error(err.Error())
			}
		}
	}()

	return nil
}

func (p *PluginEngine) loadPluginDetailsFromDirectory(pluginDirectoryPath string) (*Plugin, error) {
	p.logger.Info(fmt.Sprintf("Loading plugin from %s", pluginDirectoryPath))

	var plug Plugin

	yamlFile, err := ioutil.ReadFile(filepath.Join(pluginDirectoryPath, "config.yaml"))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, &plug.Details)
	if err != nil {
		return nil, err
	}

	p.logger.Info(fmt.Sprintf("Loading config for \"%s\" = %+v", plug.Details.Name, plug.Details.Config))

	pluginSo, err := plugin.Open(filepath.Join(pluginDirectoryPath, "plugin.so"))
	if err != nil {
		panic(err)
	}

	startFunc, err := pluginSo.Lookup("Start")
	if err != nil {
		panic(err)
	}

	versionFunc, err := pluginSo.Lookup("Version")
	if err != nil {
		panic(err)
	}

	plug.Start = startFunc.(func(*PluginToolkit, map[string]interface{}, context.Context, chan (error)))
	plug.Version = versionFunc.(func())

	return &plug, nil
}
