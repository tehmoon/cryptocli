package main

import (
  "io"
  "./flags"
  "flag"
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
