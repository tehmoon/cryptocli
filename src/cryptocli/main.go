package main

import (
	"log"
	"github.com/tehmoon/errors"
	"os"
	"fmt"
	"runtime"
)

// default version when not used by the build script
var VERSION = "dev"

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetOutput(os.Stderr)
}

func main() {
	flags, err := ParseFlags()
	if err != nil {
		err = errors.Wrap(err, "Error parsing flags")
		log.Println(err.Error())
		os.Exit(3)
	}

	if flags.Version {
		fmt.Fprintf(os.Stderr, "%s %s\n", VERSION, runtime.Version())
		os.Exit(0)
	}

	pipeline := NewPipeline()

	buff := flags.Global.MaxConcurrentStreams
	in, out := make(chan *Message, buff), make(chan *Message, buff)

//	if flags.Global.Std {
//		pipeline.Add(NewStdin())
//		flags.Modules.Register(NewStdout())
//	}

	modules := flags.Modules.Modules()
	if len(modules) == 0 {
		return
	}

	for i := range modules {
		module := modules[i]

		pipeline.Add(module)
	}

	err = pipeline.Init(in, out, &flags.Global)
	if err != nil {
		log.Fatal(err)
	}

	RelayMessages(out, in)
}
