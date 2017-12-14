package command

import (
  "io"
  "../flags"
  "os"
  "flag"
  "fmt"
)

type Command interface {
  Name() (string)
  Description() (string)
  SetupFlags(*flag.FlagSet)
  ParseFlags() (error)
  Init() (error)
  Usage() (*flags.Usage)
}

type CommandPipe interface {
  Command
  io.ReadWriteCloser
}

func ParseCommand() (CommandPipe) {
  if len(os.Args) == 1 {
    UsageCommand()
  }

  switch command := os.Args[1]; command {
    case DefaultDd.Name():
      return DefaultDd
    case DefaultDgst.Name():
      return DefaultDgst
    case DefaultPbkdf2.Name():
      return DefaultPbkdf2
    case DefaultScrypt.Name():
      return DefaultScrypt
    default:
      fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
      UsageCommand()
  }

  return nil
}

var CommandList = []Command{
  DefaultDd,
  DefaultDgst,
  DefaultPbkdf2,
  DefaultScrypt,
}

func UsageCommand() {
  fmt.Fprintln(os.Stderr, "Commands:")
  for _, command := range CommandList {
    fmt.Fprintf(os.Stderr, "  %s:  %s\n", command.Name(), command.Description())
  }

  os.Exit(2)
}
