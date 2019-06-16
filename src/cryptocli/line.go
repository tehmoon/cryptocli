package main

import (
	"bufio"
	"sync"
	"github.com/spf13/pflag"
	"log"
	"github.com/tehmoon/errors"
)

func init() {
	MODULELIST.Register("line", "Produce messages per lines", NewLine)
}

type Line struct {
	in chan *Message
	out chan *Message
	wg *sync.WaitGroup
	nl bool
}

func (m Line) Init(global *GlobalFlags) (error) {
	return nil
}

func (m Line) Start() {
	m.wg.Add(1)

	go func() {
		defer close(m.out)
		defer m.wg.Done()

		scanner := bufio.NewScanner(NewMessageReader(m.in))

		for scanner.Scan() {
			b := scanner.Bytes()
			tmp := make([]byte, len(b))
			copy(tmp, b)

			if m.nl {
				tmp = append(tmp, '\n')
			}

			SendMessage(tmp, m.out)
		}

		err := scanner.Err()
		if err != nil {
			err = errors.Wrap(err, "Error reading line")
			log.Println(err.Error())
		}
	}()
}

func (m Line) Wait() {
	m.wg.Wait()

	for range m.in {}
}

func (m *Line) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Line) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func NewLine() (Module) {
	return &Line{
		wg: &sync.WaitGroup{},
	}
}

func (m *Line) SetFlagSet(fs *pflag.FlagSet) {
	fs.BoolVar(&m.nl, "new-line", false, "Append a new line to each message")
}
