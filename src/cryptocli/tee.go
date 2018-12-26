package main

import (
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
)

func init() {
	MODULELIST.Register("tee", "Create a new one way pipeline to copy the data over", NewTee)
}

type Tee struct {
	in chan *Message
	out chan *Message
	teeIn chan *Message
	teeOut chan *Message
	sync *sync.WaitGroup
	pipe string
	pipeline *Pipeline
}

func (m *Tee) SetFlagSet(fs *pflag.FlagSet) {
	fs.StringVar(&m.pipe, "pipe", "", "Pipeline definition")
}

func (m *Tee) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *Tee) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *Tee) Init(global *GlobalFlags) (err error) {
	m.teeIn, m.teeOut, m.pipeline, err = InitPipeline(m.pipe, &GlobalFlags{})
	if err != nil {
		return errors.Wrap(err, "Error creating pipeline in tee module")
	}

	return nil
}

func (m Tee) Start() {
	m.sync.Add(2)
	m.pipeline.Start()

	go func() {
		for message := range m.in {
			payload := make([]byte, len(message.Payload))
			copy(payload, message.Payload)

			SendMessage(message.Payload, m.out)
			SendMessage(payload, m.teeIn)
		}

		close(m.out)
		close(m.teeIn)
		m.sync.Done()
	}()

	go func() {
		for range m.teeOut {}
		m.sync.Done()
	}()
}

func (m Tee) Wait() {
	m.pipeline.Wait()
	m.sync.Wait()

	for range m.in {}
}

func NewTee() (Module) {
	return &Tee{
		sync: &sync.WaitGroup{},
	}
}
