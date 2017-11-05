package main

import (
  "os"
  "./commands/dd"
  "./commands/dgst"
  "./util"
  "./codec"
  "fmt"
  "io"
  "flag"
)

func main() {
  command := parseCommand()
  globalFlags := util.SetupFlags(flag.CommandLine)
  command.SetupFlags(flag.CommandLine)
  flag.CommandLine.Usage = func () {
    usage := command.Usage()
    fmt.Fprintf(os.Stderr, "Usage: %s [<Options>] %s\n\nOptions:\n", os.Args[0], usage.CommandLine)
    flag.PrintDefaults()
    if len(codec.CodecList) > 0 {
      fmt.Fprintln(os.Stderr, "Codecs:")
      for _, c := range codec.CodecList {
        fmt.Fprintf(os.Stderr, "  %s\n", c)
      }
    }
    if usage.Other != "" {
      fmt.Fprintf(os.Stderr, "\n%s\n", usage.Other)
    }
  }
  globalOptions := util.ParseFlags(flag.CommandLine, globalFlags)
  err := command.ParseFlags()
  if err != nil {
    flag.CommandLine.Usage()
    os.Exit(2)
  }

  done := make(chan struct{})
  byteCounterIn := newByteCounter(globalOptions.FromByteIn, globalOptions.ToByteIn)
  byteCounterOut := newByteCounter(globalOptions.FromByteOut, globalOptions.ToByteOut)

  go func() {
    _, err := io.Copy(os.Stdout, globalOptions.Encoder)
    if err != nil {
      fmt.Printf("Err in reading encoder: %v\n", err)
      return
    }

    stdoutFileInfo, _ := os.Stdout.Stat()
    if (! (stdoutFileInfo.Mode() & os.ModeCharDevice == 0) && ! globalFlags.Chomp) {
      fmt.Println()
    }

    done <- struct{}{}
  }()

  if globalOptions.FromByteOut != 0 || globalOptions.ToByteOut != 0 {
    go func() {
       _, err := io.Copy(byteCounterOut, command)
      if err != nil {
        fmt.Printf("Err reading in command: %v\n", err)
        os.Exit(1)
      }

      byteCounterOut.Close()
    }()

    go func() {
      err := globalOptions.Encoder.Init()
      if err != nil {
        fmt.Printf("Err in init encoder: %v\n", err)
        os.Exit(1)
      }
       _, err = io.Copy(globalOptions.Encoder, byteCounterOut)
      if err != nil {
        fmt.Printf("Err reading in byteCounterOut: %v\n", err)
        os.Exit(1)
      }

      globalOptions.Encoder.Close()
    }()
  } else {
    go func() {
      err := globalOptions.Encoder.Init()
      if err != nil {
        fmt.Printf("Err in init encoder: %v\n", err)
        os.Exit(1)
      }
      _, err = io.Copy(globalOptions.Encoder, command)
      if err != nil {
        fmt.Printf("Err in reading command: %v\n", err)
        os.Exit(1)
      }

      globalOptions.Encoder.Close()
    }()
  }

  if globalOptions.FromByteIn != 0 || globalOptions.ToByteIn != 0 {
    go func() {
      err := globalOptions.Decoder.Init()
      if err != nil {
        fmt.Printf("Err in init decoder: %v\n", err)
        os.Exit(1)
      }
       _, err = io.Copy(byteCounterIn, globalOptions.Decoder)
      if err != nil {
        fmt.Printf("Err reading in decoder: %v\n", err)
        os.Exit(1)
      }

      byteCounterIn.Close()
    }()

    go func() {
       _, err := io.Copy(command, byteCounterIn)
      if err != nil {
        fmt.Printf("Err reading in byteCounterIn: %v\n", err)
        os.Exit(1)
      }

      command.Close()
    }()
  } else {
    go func() {
      err := globalOptions.Decoder.Init()
      if err != nil {
        fmt.Printf("Err in init decoder: %v\n", err)
        os.Exit(1)
      }
      _, err = io.Copy(command, globalOptions.Decoder)
      if err != nil {
        fmt.Printf("Err in reading decoder: %v\n", err)
        os.Exit(1)
      }
      command.Close()
    }()
  }

  stdinFileInfo, _ := os.Stdin.Stat()
  if (stdinFileInfo.Mode() & os.ModeCharDevice == 0) {
    _, err := io.Copy(globalOptions.Decoder, os.Stdin)
    if err != nil {
      fmt.Printf("Error in decoding stdin: %v\n", err)
      os.Exit(1)
    }
    globalOptions.Decoder.Close()
    <- done
  }
}

var CommandList = []Command{
  dd.Command,
  dgst.Command,
}

func UsageCommand() {
  fmt.Fprintln(os.Stderr, "Commands:")
  for _, command := range CommandList {
    fmt.Fprintf(os.Stderr, "  %s:  %s\n", command.Name(), command.Description())
  }

  os.Exit(2)
}

func parseCommand() (CommandPipe) {
  if len(os.Args) == 1 {
    UsageCommand()
  }

  switch command := os.Args[1]; command {
    case dd.Command.Name():
      return dd.Command
    case dgst.Command.Name():
      return dgst.Command
    default:
      fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
      UsageCommand()
  }

  return nil
}
