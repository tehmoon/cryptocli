package main

import (
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("null", "Discard all incoming data", NewNull)
}

type Null struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
}

func (m Null) Init(global *GlobalFlags) (error) {
	return nil
}

func (m Null) Start() {
	m.sync.Add(1)

	go func() {
		for range m.in {}
		close(m.out)
		m.sync.Done()
	}()
}

func (m Null) Wait() {
	m.sync.Wait()
}

func (m *Null) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Null) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewNull() (Module) {
	return &Null{
		sync: &sync.WaitGroup{},
	}
}

func (m *Null) SetFlagSet(fs *pflag.FlagSet) {}
