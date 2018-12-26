package main

import (
	"github.com/tehmoon/errors"
	"github.com/google/shlex"
	"github.com/spf13/pflag"
	"bytes"
)

type Pipeline struct {
	modules []PipelineModule
	in chan *Message
	out chan *Message
}

type PipelineModule struct {
	In chan *Message
	Out chan *Message
	Module Module
}

func NewPipeline() (*Pipeline) {
	return &Pipeline{
		modules: make([]PipelineModule, 0),
	}
}

func (p *Pipeline) Add(m Module) {
	in, out := NewPipeMessages()
	p.modules = append(p.modules, PipelineModule{
		In: in,
		Out: out,
		Module: m,
	})
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

func (p *Pipeline) In(in chan *Message) (chan *Message) {
	p.in = in

	return in
}

func (p *Pipeline) Out(out chan *Message) (chan *Message) {
	p.out = out

	return out
}

func (p Pipeline) Init(global *GlobalFlags) (error) {
	if len(p.modules) == 0 {
		go RelayMessages(p.in, p.out)

		return nil
	}

	for i := range p.modules {
		module := p.modules[i]

		in, out := module.In, module.Out
		in = module.Module.In(in)
		out = module.Module.Out(out)

		err := module.Module.Init(global)
		if err != nil {
			return errors.Wrapf(err, "Error in module number %d", i)
		}

		if i == 0 {
			go RelayMessages(p.in, in)
		}

		if i == len(p.modules) - 1 {
			go RelayMessages(out, p.out)
			continue
		}

		go RelayMessages(out, p.modules[i + 1].In)
	}

	return nil
}

func (p Pipeline) Start() {
	for i := range p.modules {
		module := p.modules[i]

		module.Module.Start()
	}
}

func (p Pipeline) Wait() {
	for i := range p.modules {
		module := p.modules[i]

		module.Module.Wait()
	}
}

func ReadAllPipeline(pipe string) ([]byte, error) {
	in, out, pipeline, err := InitPipeline(pipe, &GlobalFlags{})
	if err != nil {
		return nil, errors.Wrap(err, "Error starting pipeline")
	}

	buff := bytes.NewBuffer(nil)

	go func() {
		for message := range out {
			buff.Write(message.Payload)
		}

		close(in)
	}()

	pipeline.Start()
	pipeline.Wait()

	return buff.Bytes(), nil
}

// Create a pipeline and initialize it.
// Returns both sides of the pipeline and the pipeline itself.
// You will also have to call Start() and Wait()
func InitPipeline(pipe string, global *GlobalFlags) (in, out chan *Message, pipeline *Pipeline, err error) {
	pipeline = NewPipeline()
	in, out = NewPipeMessages()
	in = pipeline.In(in)
	out = pipeline.Out(out)

	err = pipeline.Parse(pipe)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Error parsing pipeline")
	}

	err = pipeline.Init(global)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Error initializing pipeline")
	}

	return in, out, pipeline, nil
}
