package dgst

import (
  "io"
  "flag"
  "crypto/sha256"
  "crypto/sha512"
  "hash"
  "../../flags"
)

type Dgst struct {
  name string
  description string
  hash hash.Hash
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  flagSet *flag.FlagSet
  usage *flags.Usage
}

var Command *Dgst

func init() {
  command := &Dgst{
    name: "dgst",
    description: "Hash the content of stdin",
    usage: &flags.Usage{
      CommandLine: "<hash algorithm>",
      Other: "Hash Algorithms:\n  sha256\n  sha512",
    },
  }
  command.pipeReader, command.pipeWriter = io.Pipe()

  Command = command
}

func (command Dgst) Usage() (*flags.Usage) {
  return command.usage
}

func (command Dgst) Name() (string) {
  return command.name
}

func (command Dgst) Description() (string) {
  return command.description
}

func (command Dgst) Read(p []byte) (int, error) {
  return command.pipeReader.Read(p)
}

func (command Dgst) Write(data []byte) (int, error) {
  return command.hash.Write(data)
}

func (command Dgst) Close() (error) {
  _, err := command.pipeWriter.Write(command.hash.Sum(nil))
  if err != nil {
    return err
  }

  return command.pipeWriter.Close()
}

func (command *Dgst) SetupFlags(set *flag.FlagSet) {
  command.flagSet = set
}

func (command *Dgst) ParseFlags() (error) {
  hashFunction := ""

  if command.flagSet.Parsed() {
    hashFunction = command.flagSet.Arg(0)
  }

  switch hashFunction {
    case "sha256":
      command.hash = sha256.New()
    case "sha512":
      command.hash = sha512.New()
    case "":
      command.hash = sha512.New()
    default:
      return flags.ErrBadFlag
  }

  return nil
}
