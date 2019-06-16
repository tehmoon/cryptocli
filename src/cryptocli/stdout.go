package main

import (
	"sync"
	"os"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

var stdoutMutex = struct{sync.Mutex; Init bool}{Init: false,}

func init() {
	MODULELIST.Register("stdout", "Writes to stdout", NewStdout)
}

type Stdout struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
}

func (m Stdout) Init(global *GlobalFlags) (error) {
	stdoutMutex.Lock()
	defer stdoutMutex.Unlock()
	if stdoutMutex.Init {
		return errors.New("Module \"stdout\" cannot be added more than once")
	}

	stdoutMutex.Init = true

	return nil
}

func (m Stdout) Start() {
	m.sync.Add(1)

	go func() {
		for message := range m.in {
			os.Stdout.Write(message.Payload)
			os.Stdout.Sync()
		}

		close(m.out)
		m.sync.Done()
	}()
}

func (m Stdout) Wait() {
	m.sync.Wait()
}

func (m *Stdout) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Stdout) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewStdout() (Module) {
	return &Stdout{
		sync: &sync.WaitGroup{},
	}
}

func (m Stdout) SetFlagSet(fs *pflag.FlagSet) {}
