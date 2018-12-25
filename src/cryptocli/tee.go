package main

import (
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"github.com/google/shlex"
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

func (m *Tee) Init(global *GlobalFlags) (error) {
	m.pipeline = NewPipeline()
	m.teeIn, m.teeOut = NewPipeMessages()
	m.teeIn = m.pipeline.In(m.teeIn)
	m.teeOut = m.pipeline.Out(m.teeOut)

	words, err := shlex.Split(m.pipe)
	if err != nil {
		return errors.Wrap(err, "Error parsing pipe argument in tee module")
	}
	words = append([]string{"--",}, words...)

	mods := NewModules()
	root := pflag.NewFlagSet("tee", pflag.ContinueOnError)
	root.Parse(words)

	err = ParseRootRemainingArgs(mods, 0, root)
	if err != nil {
		return errors.Wrap(err, "Error parsing pipeline in tee module")
	}

	modules := mods.Modules()
	for i := range modules {
		module := modules[i]

		m.pipeline.Add(module)
	}

	err = m.pipeline.Init(&GlobalFlags{})
	if err != nil {
		return errors.Wrap(err, "Error initializing pipeline in tee module")
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
			SendMessage(payload, m.teeOut)
		}

		close(m.out)
		close(m.teeOut)
		m.sync.Done()
	}()

	go func() {
		for range m.teeIn {}
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
