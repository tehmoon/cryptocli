package main

import (
  "os"
  "./flags"
  "./codec"
  "./inout"
  "fmt"
  "io"
  "flag"
  "./command"
)

func main() {
  cmd := command.ParseCommand()
  globalFlags := flags.SetupFlags(flag.CommandLine)
  cmd.SetupFlags(flag.CommandLine)

  flag.CommandLine.Usage = flags.PrintUsage(cmd.Usage(), codec.CodecList, inout.InoutList)

  globalOptions := flags.ParseFlags(flag.CommandLine, globalFlags)
  err := cmd.ParseFlags()
  if err != nil {
    flag.CommandLine.Usage()
    os.Exit(2)
  }

  err = cmd.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, "Error initializing command: %s: %v", cmd.Name(), err)
  }

  done := make(chan struct{})
  byteCounterIn := newByteCounter(globalOptions.FromByteIn, globalOptions.ToByteIn)
  byteCounterOut := newByteCounter(globalOptions.FromByteOut, globalOptions.ToByteOut)

  var cmdOut io.Reader = cmd

  if globalOptions.TeeCmdOut != nil {
    err := globalOptions.TeeCmdOut.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error initializing tee command output: %v", err)
      os.Exit(1)
    }

    cmdOut = io.TeeReader(cmd, globalOptions.TeeCmdOut)
  }

  if globalOptions.TeeCmdIn != nil {
    err := globalOptions.TeeCmdIn.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error initializing tee command input: %v", err)
      os.Exit(1)
    }
  }

  go func() {
    err := globalOptions.Encoders[len(globalOptions.Encoders) - 1].Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err in init encoder: %v", err)
      os.Exit(1)
    }

    err = globalOptions.Output.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err initializing output: %v", err)
      os.Exit(1)
    }

    var lastReader io.Reader
    lastReader = globalOptions.Encoders[len(globalOptions.Encoders) - 1]

    if globalOptions.TeeOut != nil {
      err = globalOptions.TeeOut.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err initializing output: %v", err)
        os.Exit(1)
      }

      lastReader = io.TeeReader(lastReader, globalOptions.TeeOut)
    }

    _, err = io.Copy(globalOptions.Output, lastReader)
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err in reading encoder: %v", err)
      os.Exit(1)
    }

    globalOptions.Output.Chomp(globalFlags.Chomp)

    err = globalOptions.Output.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, "Err closing output: %v", err)
      os.Exit(1)
    }

    done <- struct{}{}
  }()

  var encoderReader codec.CodecEncoder
  encoderReader = globalOptions.Encoders[0]

  for _, encoder := range globalOptions.Encoders[1:] {
    go func(encoder codec.CodecEncoder, encoderReader codec.CodecEncoder) {
      err := encoder.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init encoder: %v", err)
        os.Exit(1)
      }

      _, err = io.Copy(encoder, encoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading reader: %v", err)
        os.Exit(1)
      }

      err = encoder.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(encoder, encoderReader)

    encoderReader = encoder
  }

  if globalOptions.FromByteOut != 0 || globalOptions.ToByteOut != 0 {
    go func() {
       _, err := io.Copy(byteCounterOut, cmdOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in command: %v", err)
        os.Exit(1)
      }

      err = byteCounterOut.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, byteCounterOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in byteCounterOut: %v", err)
        os.Exit(1)
      }

      err = encoderReader.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Encoders[0])
  } else {
    go func(encoderReader codec.CodecEncoder) {
       _, err = io.Copy(encoderReader, cmdOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in encoderReader: %v", err)
        os.Exit(1)
      }

      err = encoderReader.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Encoders[0])
  }

  var decoderReader codec.CodecDecoder
  decoderReader = globalOptions.Decoders[0]

  for _, decoder := range globalOptions.Decoders[1:] {
    go func(decoder codec.CodecDecoder, decoderReader codec.CodecDecoder) {
      err := decoderReader.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }

      _, err = io.Copy(decoder, decoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading decoder decoderReader: %v", err)
        os.Exit(1)
      }

      err = decoder.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(decoder, decoderReader)

    decoderReader = decoder
  }

  if globalOptions.FromByteIn != 0 || globalOptions.ToByteIn != 0 {
    go func() {
      err := decoderReader.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }
       _, err = io.Copy(byteCounterIn, decoderReader)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in decoder: %v", err)
        os.Exit(1)
      }

      err = byteCounterIn.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func() {
      var cmdIn io.Reader = byteCounterIn

      if globalOptions.TeeCmdIn != nil {
        cmdIn = io.TeeReader(cmdIn, globalOptions.TeeCmdIn)
      }

       _, err := io.Copy(cmd, cmdIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err reading in byteCounterIn: %v", err)
        os.Exit(1)
      }

      err = cmd.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  } else {
    go func(decoder codec.CodecDecoder) {
      err := decoder.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in init decoder: %v", err)
        os.Exit(1)
      }

      var cmdIn io.Reader = decoder

      if globalOptions.TeeCmdIn != nil {
        cmdIn = io.TeeReader(cmdIn, globalOptions.TeeCmdIn)
      }

      _, err = io.Copy(cmd, cmdIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, "Err in reading decoder decoderReader: %v", err)
        os.Exit(1)
      }

      err = cmd.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }(globalOptions.Decoders[len(globalOptions.Decoders) - 1])
  }

  go func() {
    err := globalOptions.Input.Init()
    if err != nil {
      if err == io.EOF {
        globalFlags.Chomp = true
        globalOptions.Decoders[0].Close()
        return
      }

      fmt.Fprintf(os.Stderr, "Error initializing input: %v", err)
      os.Exit(1)
    }

    var reader io.Reader

    reader = globalOptions.Input

    if globalOptions.TeeIn != nil {
      err := globalOptions.TeeIn.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, "Error initializing tee input: %v", err)
        os.Exit(1)
      }

      reader = io.TeeReader(globalOptions.Input, globalOptions.TeeIn)
    }

    _, err = io.Copy(globalOptions.Decoders[0], reader)
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error in decoding input: %v", err)
      os.Exit(1)
    }

    err = globalOptions.Input.Close()
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }

    if globalOptions.TeeIn != nil {
      err = globalOptions.TeeIn.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }

    err = globalOptions.Decoders[0].Close()
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }
  }()

  <- done
}
