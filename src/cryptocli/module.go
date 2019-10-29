package main

import (
	"sync"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"fmt"
	"os"
)

var MODULELIST = NewModuleList()

type Module interface {
	// Initialize module's configuration, throwing errors
	// if there is something wrong
	// Init should not be blocking
	Init(in, out chan *Message, flags *GlobalFlags) (err error)

	SetFlagSet(flags *pflag.FlagSet, args []string)
}

type Modules struct {
	sync.Mutex
	modules []Module
}

func (m *Modules) Register(module Module) {
	m.Lock()
	defer m.Unlock()

	m.modules = append(m.modules, module)
}

func (m *Modules) Modules() ([]Module) {
	m.Lock()
	defer m.Unlock()

	modules := make([]Module, len(m.modules))
	for i := range m.modules {
		module := m.modules[i]
		modules[i] = module
	}

	return modules
}

func NewModules() (*Modules) {
	return &Modules{
		modules: make([]Module, 0),
	}
}

type ModuleList struct {
	sync.Mutex
	modules map[string]ModuleListInfo
}

func (m *ModuleList) Register(name, desc string, f func() (Module)) {
	m.Lock()
	defer m.Unlock()

	_, found := m.modules[name]
	if found {
		fmt.Fprintf(os.Stderr, "Module %q is already in the list\n", name)
		os.Exit(1)
	}

	module := ModuleListInfo{
		F: f,
		ShortDescription: desc,
	}

	m.modules[name] = module
}

func (m *ModuleList) Find(name string) (Module, error) {
	m.Lock()
	defer m.Unlock()

	info, found := m.modules[name]
	if ! found {
		return nil, errors.New("Module not found")
	}

	return info.F(), nil
}

func (m *ModuleList) Help() (string) {
	m.Lock()
	defer m.Unlock()

	message := "List of all modules:"

	for name := range m.modules {
		module := m.modules[name]

		message += fmt.Sprintf("\n\t%s: %s", name, module.ShortDescription)
	}

	return message
}

type ModuleListInfo struct {
	F func() (Module)
	ShortDescription string
}

func NewModuleList() (*ModuleList) {
	return &ModuleList{
		modules: make(map[string]ModuleListInfo),
	}
}
