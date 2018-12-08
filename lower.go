package main

import (
	"sync"
	"github.com/spf13/pflag"
)

func init() {
	MODULELIST.Register("lower", "Lowercase all ascii characters", NewLower)
}

type Lower struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
}

func (m Lower) Init(global *GlobalFlags) (error) {
	return nil
}

func (m Lower) Start() {
	m.wg.Add(1)

	go startLower(m.in, m.out, m.wg)
}

func (m Lower) Wait() {
	m.wg.Wait()
}

func NewLower() (Module) {
	return &Lower{
		wg: &sync.WaitGroup{},
	}
}

func (m *Lower) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Lower) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func startLower(in, out chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		for i, b := range message.Payload {
			if b > 64 && b < 91 {
				message.Payload[i] = b + 32
			}
		}

		out <- message
	}

	close(out)
	wg.Done()
}

func (m *Lower) SetFlagSet(fs *pflag.FlagSet) {}
