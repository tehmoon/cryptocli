package main

import (
	"bytes"
	"github.com/tehmoon/errors"
	"github.com/google/shlex"
	"github.com/spf13/pflag"
	"sync"
)

type Pipeline struct {
	modules []Module
}

func NewPipeline() (*Pipeline) {
	return &Pipeline{
		modules: make([]Module, 0),
	}
}

func (p *Pipeline) Add(m Module) {
	p.modules = append(p.modules, m)
}

func (p *Pipeline) Parse(cl string) (error) {
	words, err := shlex.Split(cl)
	if err != nil {
		return errors.Wrap(err, "Error parsing pipeline command line")
	}
	words = append([]string{"--",}, words...)

	mods := NewModules()
	root := pflag.NewFlagSet("root", pflag.ContinueOnError)
	root.Parse(words)

	err = ParseRootRemainingArgs(mods, 0, root)
	if err != nil {
		return errors.Wrap(err, "Error parsing pipeline modules")
	}

	modules := mods.Modules()
	for i := range modules {
		module := modules[i]

		p.Add(module)
	}

	return nil
}

func (p Pipeline) Init(pipeIn, pipeOut chan *Message, global *GlobalFlags) (err error) {
	if len(p.modules) == 0 {
		return nil
	}

	if len(p.modules) == 1 {
		err = p.modules[0].Init(pipeIn, pipeOut, global)
		if err != nil {
			return errors.Wrapf(err, "Error in module number %d", 1)
		}

		return nil
	}

	// -1 since we start this at len of 2
	buff := 1
	chans := make([]chan *Message, len(p.modules) - 1)
	for i := range chans {
		chans[i] = make(chan *Message, buff)
	}

	for i := range p.modules {
		module := p.modules[i]

		if i == 0 {
			err = module.Init(pipeIn, chans[i], global)
			if err != nil {
				return errors.Wrapf(err, "Error in module number %d", i)
			}

			continue
		}

		if i == len(p.modules) - 1 {
			err = module.Init(chans[i - 1], pipeOut, global)
			if err != nil {
				return errors.Wrapf(err, "Error in module number %d", i)
			}

			continue
		}

		err = module.Init(chans[i - 1], chans[i], global)
		if err != nil {
			return errors.Wrapf(err, "Error in module number %d", i)
		}
	}

	return nil
}

func WriteToPipeline(pipe string, data []byte) error {
	in, out, _, err := InitPipeline(pipe, &GlobalFlags{
		MultiStreams: false,
		MaxConcurrentStreams: 1,
	})
	if err != nil {
		return errors.Wrap(err, "Error starting pipeline")
	}

	mc := NewMessageChannel()
	out <- &Message{
		Type: MessageTypeChannel,
		Interface: mc.Callback,
	}

	message, opened := <- in
	if ! opened {
		close(mc.Channel)
		close(out)
		return errors.New("Pipeline is empty!")
	}

	wg := &sync.WaitGroup{}

	LOOP: for {
		switch message.Type {
			case MessageTypeTerminate:
				close(mc.Channel)
				wg.Wait()
				out <- message
				break LOOP
			case MessageTypeChannel:
				cb, ok := message.Interface.(MessageChannelFunc)
				if ok {
					wg.Add(1)
					go func() {
						defer wg.Done()
						mc.Start(nil)
						_, inc := cb()

						wg.Add(1)
						go DrainChannel(inc, wg)

						mc.Channel <- data
						close(mc.Channel)
					}()

					wg.Wait()

					out <- &Message{
						Type: MessageTypeTerminate,
					}
					break LOOP
				}
		}
	}

	wg.Wait()
	<- in
	close(out)

	return nil
}

func ReadAllPipeline(pipe string) ([]byte, error) {
	in, out, _, err := InitPipeline(pipe, &GlobalFlags{})
	if err != nil {
		return nil, errors.Wrap(err, "Error starting pipeline")
	}

	buff := bytes.NewBuffer(nil)

	mc := NewMessageChannel()

	out <- &Message{
		Type: MessageTypeChannel,
		Interface: mc.Callback,
	}

	message, opened := <- in
	if ! opened {
		close(mc.Channel)
		close(out)
		return nil, errors.New("Pipeline is empty!")
	}

	LOOP: for {
		switch message.Type {
			case MessageTypeTerminate:
				close(mc.Channel)
				out <- message
				break LOOP
			case MessageTypeChannel:
				cb, ok := message.Interface.(MessageChannelFunc)
				if ok {
					mc.Start(nil)
					_, inc := cb()

					for payload := range inc {
						buff.Write(payload)
					}

					close(mc.Channel)
					out <- &Message{
						Type: MessageTypeTerminate,
					}
					break LOOP
				}
		}
	}

	<- in
	close(out)

	return buff.Bytes(), nil
}

// TOOD: refact
//// Create a pipeline and initialize it.
//// Returns both sides of the pipeline and the pipeline itself.
//// You will also have to call Start() and Wait()
func InitPipeline(pipe string, global *GlobalFlags) (in, out chan *Message, pipeline *Pipeline, err error) {
	pipeline = NewPipeline()
	buff := 1
	in, out = make(chan *Message, buff), make(chan *Message, buff)

	err = pipeline.Parse(pipe)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Error parsing pipeline")
	}

	err = pipeline.Init(in, out, global)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Error initializing pipeline")
	}

	// we reverse at the end because the end of the pipeline is
	// the input for another module
	return out, in, pipeline, nil
}
