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
	line bool
}

func (m *Stdin) Init(global *GlobalFlags) (error) {
	stdinMutex.Lock()
	defer stdinMutex.Unlock()
	if stdinMutex.Init {
		return errors.New("Module \"stdin\" cannot be added more than once")
	}

	stdinMutex.Init = true

	if global.Line {
		m.line = true
	}

	return nil
}

func (m Stdin) Start() {
	m.sync.Add(2)

	go stdinStartOut(m.out, m.line, m.sync)
	go stdinStartIn(m.in, m.sync)
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

func stdinStartOut(write chan *Message, line bool, wg *sync.WaitGroup) {
	var err error

	cb := func(payload []byte) {
		SendMessage(payload, write)
	}

	if line {
		err = ReadDelimStep(os.Stdin, '\n', cb)
	} else {
		err = ReadBytesStep(os.Stdin, cb)
	}

	if err != nil {
		log.Println(errors.Wrap(err, "Error copying stdin"))
	}

	close(write)

	wg.Done()
}

func stdinStartIn(read chan *Message, wg *sync.WaitGroup) {
	for range read {}

	log.Println("Press c^d to close stdin")

	wg.Done()
}

func (m *Stdin) SetFlagSet(fs *pflag.FlagSet) {
	fs.BoolVar(&m.line, "line", false, "Read lines from the stdin")
}
