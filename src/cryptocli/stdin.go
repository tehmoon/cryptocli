package main

import (
	"sync"
	"os"
	"log"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

var stdinMutex = struct{sync.Mutex; Init bool}{Init: false,}

func init() {
	MODULELIST.Register("stdin", "Reads from stdin", NewStdin)
}

type Stdin struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
}

func (m *Stdin) Init(global *GlobalFlags) (error) {
	stdinMutex.Lock()
	defer stdinMutex.Unlock()
	if stdinMutex.Init {
		return errors.New("Module \"stdin\" cannot be added more than once")
	}

	stdinMutex.Init = true

	return nil
}

func (m Stdin) Start() {
	m.sync.Add(1)

	// Cancel will tell stdin to stop reading and close the out channel
	cancel := make(chan struct{}, 0)

	go stdinStartOut(m.out, cancel)
	go stdinStartIn(m.in, cancel, m.sync)
}

func (m Stdin) Wait() {
	m.sync.Wait()
}

func (m *Stdin) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Stdin) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewStdin() (Module) {
	return &Stdin{
		sync: &sync.WaitGroup{},
	}
}

func stdinStartOutRead(write chan *Message, closed *StdinCloseSync) {
	err := ReadBytesStep(os.Stdin, func(payload []byte) (bool) {
		if closed.Closed {
			return false
		}

		SendMessage(payload, write)
		return true
	})
	if err != nil {
		log.Println(errors.Wrap(err, "Error copying stdin"))
	}

	closed.Lock()
	closed.Closed = true
	close(write)
	closed.Unlock()
}

type StdinCloseSync struct {
	sync.Mutex
	Closed bool
}

func stdinStartOut(write chan *Message, cancel chan struct{}) {
	// Closed makes sure it is safe to close the channel.
	// Otherwise it panics.
	closed := &StdinCloseSync{
		Closed: false,
	}

	go stdinStartOutRead(write, closed)

	<- cancel

	closed.Lock()
	if ! closed.Closed {
		closed.Closed = true
		close(write)
	}
	closed.Unlock()
}

func stdinStartIn(read chan *Message, cancel chan struct{}, wg *sync.WaitGroup) {
	for range read {}

	cancel <- struct{}{}

	wg.Done()
}

func (m *Stdin) SetFlagSet(fs *pflag.FlagSet) {}
