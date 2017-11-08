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
    err := globalOptions.Encoders[len(globalOptions.Encoders) - 1].Init()
    if err != nil {
      fmt.Printf("Err in init encoder: %v\n", err)
      return
    }
    _, err = io.Copy(os.Stdout, globalOptions.Encoders[len(globalOptions.Encoders) - 1])
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

  var encoderReader codec.CodecEncoder
  encoderReader = globalOptions.Encoders[0]

  for _, encoder := range globalOptions.Encoders[1:] {
    go func(encoder codec.CodecEncoder, encoderReader codec.CodecEncoder) {
      err := encoder.Init()
      if err != nil {
        fmt.Printf("Err in init encoder: %v\n", err)
        os.Exit(1)
      }

      _, err = io.Copy(encoder, encoderReader)
      if err != nil {
        fmt.Printf("Err in reading reader: %v\n", err)
        os.Exit(1)
      }

      encoder.Close()
    }(encoder, encoderReader)

    encoderReader = encoder
  }

  if globalOptions.FromByteOut != 0 || globalOptions.ToByteOut != 0 {
    go func() {
       _, err := io.Copy(byteCounterOut, command)
      if err != nil {
        fmt.Printf("Err reading in command: %v\n", err)
        os.Exit(1)
      }

      byteCounterOut.Close()
    }()

    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, byteCounterOut)
      if err != nil {
        fmt.Printf("Err reading in byteCounterOut: %v\n", err)
        os.Exit(1)
      }

      encoderReader.Close()
    }(globalOptions.Encoders[0])
  } else {
    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, command)
      if err != nil {
        fmt.Printf("Err reading in byteCounterOut: %v\n", err)
        os.Exit(1)
      }

      encoderReader.Close()
    }(globalOptions.Encoders[0])
  }

  var decoderReader codec.CodecDecoder
  decoderReader = globalOptions.Decoders[0]

  for _, decoder := range globalOptions.Decoders[1:] {
    go func(decoder codec.CodecDecoder, decoderReader codec.CodecDecoder) {
      err := decoderReader.Init()
      if err != nil {
        fmt.Printf("Err in init decoder: %v\n", err)
        os.Exit(1)
      }

      _, err = io.Copy(decoder, decoderReader)
      if err != nil {
        fmt.Printf("Err in reading decoder decoderReader: %v\n", err)
        os.Exit(1)
      }

      decoder.Close()
    }(decoder, decoderReader)

    decoderReader = decoder
  }

  if globalOptions.FromByteIn != 0 || globalOptions.ToByteIn != 0 {
    go func() {
      err := decoderReader.Init()
      if err != nil {
        fmt.Printf("Err in init decoder: %v\n", err)
        os.Exit(1)
      }
       _, err = io.Copy(byteCounterIn, decoderReader)
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
    go func(decoder codec.CodecDecoder) {
      err := decoder.Init()
      if err != nil {
        fmt.Printf("Err in init decoder: %v\n", err)
        os.Exit(1)
      }
      _, err = io.Copy(command, decoder)
      if err != nil {
        fmt.Printf("Err in reading decoder decoderReader: %v\n", err)
        os.Exit(1)
      }

      command.Close()
    }(globalOptions.Decoders[len(globalOptions.Decoders) - 1])
  }

  stdinFileInfo, _ := os.Stdin.Stat()
  if (stdinFileInfo.Mode() & os.ModeCharDevice == 0) {
    _, err := io.Copy(globalOptions.Decoders[0], os.Stdin)
    if err != nil {
      fmt.Printf("Error in decoding stdin: %v\n", err)
      os.Exit(1)
    }
    globalOptions.Decoders[0].Close()
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
