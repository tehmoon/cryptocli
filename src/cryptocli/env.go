package main

import (
	"sync"
	"github.com/spf13/pflag"
	"github.com/tehmoon/errors"
	"os"
)

func init() {
	MODULELIST.Register("env", "Read an environment variable", NewEnv)
}

type Env struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	v string
}

func (m Env) Init(global *GlobalFlags) (error) {
	if m.v == "" {
		return errors.Errorf("Flag %q must be specified in module init", "var")
	}

	return nil
}

func (m Env) Start() {
	m.wg.Add(1)

	go func() {
		SendMessage([]byte(os.Getenv(m.v)), m.out)

		close(m.out)

		m.wg.Done()
	}()
}

func (m Env) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Env) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Env) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewEnv() (Module) {
	return &Env{
		wg: &sync.WaitGroup{},
	}
}

func (m *Env) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.v, "var", "", "Variable to read from")
}
