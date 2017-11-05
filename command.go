package main

import (
  "io"
  "./util"
  "flag"
)

type Command interface {
  Name() (string)
  Description() (string)
  SetupFlags(*flag.FlagSet)
  ParseFlags() (error)
  Usage() (*util.Usage)
}

type CommandPipe interface {
  Command
  io.ReadWriteCloser
}
