package command

import (
  "io"
  "flag"
  "../flags"
)

type Dd struct {
  name string
  description string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  usage *flags.Usage
}

var DefaultDd = &Dd{
  name: "dd",
  description: "Copy input to output like the dd tool.",
  usage: &flags.Usage{
    CommandLine: "",
    Other: "",
  },
}

func (command *Dd) Init() (error) {
  command.pipeReader, command.pipeWriter = io.Pipe()

  return nil
}

func (command Dd) Usage() (*flags.Usage) {
  return command.usage
}

func (command Dd) Name() (string) {
  return command.name
}

func (command Dd) Description() (string) {
  return command.description
}

func (command Dd) Read(p []byte) (int, error) {
  return command.pipeReader.Read(p)
}

func (command Dd) Write(data []byte) (int, error) {
  return command.pipeWriter.Write(data)
}

func (command Dd) Close() (error) {
  return command.pipeWriter.Close()
}

func (command Dd) SetupFlags(set *flag.FlagSet) {}
func (command Dd) ParseFlags() (error) {
  return nil
}
