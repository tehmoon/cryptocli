package main

import (
	"log"
	"github.com/tehmoon/errors"
	"os"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stderr)
}

func main() {
	flags, err := ParseFlags()
	if err != nil {
		log.Println(errors.Wrap(err, "Error parsing flags"))
		os.Exit(3)
	}

	pipeline := NewPipeline()

	in, out := NewPipeMessages()
	in = pipeline.In(in)
	out = pipeline.Out(out)

	if flags.Global.Std {
		pipeline.Add(NewStdin())
		flags.Modules.Register(NewStdout())
	}

	modules := flags.Modules.Modules()
	if len(modules) == 0 {
		return
	}

	for i := range modules {
		module := modules[i]

		pipeline.Add(module)
	}

	err = pipeline.Init(&flags.Global)
	if err != nil {
		log.Fatal(err)
	}

	pipeline.Start()
	RelayMessages(out, in)
	pipeline.Wait()
}
