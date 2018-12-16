package main

import (
	"sync"
	"github.com/spf13/pflag"
)

func init() {
	MODULELIST.Register("upper", "Uppercase all ascii characters", NewUpper)
}

type Upper struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
}

func (m Upper) Init(global *GlobalFlags) (error) {
	return nil
}

func (m Upper) Start() {
	m.wg.Add(1)

	go startUpper(m.in, m.out, m.wg)
}

func (m Upper) Wait() {
	m.wg.Wait()
}

func (m *Upper) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Upper) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewUpper() (Module) {
	return &Upper{
		wg: &sync.WaitGroup{},
	}
}

func startUpper(in, out chan *Message, wg *sync.WaitGroup) {
	for message := range in {
		for i, b := range message.Payload {
			if b > 96 && b < 123 {
				message.Payload[i] = b - 32
			}
		}

		out <- message
	}

	close(out)
	wg.Done()
}

func (m *Upper) SetFlagSet(fs *pflag.FlagSet) {}
