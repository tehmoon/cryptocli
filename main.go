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
  "./pipeline"
  "github.com/tehmoon/errors"
  "./filter"
)

func main() {
  cmd := command.ParseCommand()
  globalFlags := flags.SetupFlags(flag.CommandLine)
  cmd.SetupFlags(flag.CommandLine)

  flag.CommandLine.Usage = flags.PrintUsage(cmd.Usage(), codec.CodecList, inout.InoutList, filter.FilterList)

  globalOptions := flags.ParseFlags(flag.CommandLine, globalFlags)
  err := cmd.ParseFlags(globalOptions)
  if err != nil {
    fmt.Fprintln(os.Stderr, err.Error())
    flag.CommandLine.Usage()
    os.Exit(2)
  }

  err = cmd.Init()
  if err != nil {
    fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error initializing command: %s", cmd.Name()).Error())
    os.Exit(1)
  }

  done := make(chan struct{})
  byteCounterIn := newByteCounter(globalOptions.FromByteIn, globalOptions.ToByteIn)
  byteCounterOut := newByteCounter(globalOptions.FromByteOut, globalOptions.ToByteOut)

  filtersIn := pipeline.New()
  filtersCmdIn := pipeline.New()
  filtersCmdOut := pipeline.New()
  filtersOut := pipeline.New()

  if len(globalOptions.FiltersIn) != 0 {
    for _, f := range globalOptions.FiltersIn {
      filtersIn.Add(f)
    }
  }

  if len(globalOptions.FiltersCmdIn) != 0 {
    for _, f := range globalOptions.FiltersCmdIn {
      filtersCmdIn.Add(f)
    }
  }

  if len(globalOptions.FiltersCmdOut) != 0 {
    for _, f := range globalOptions.FiltersCmdOut {
      filtersCmdOut.Add(f)
    }
  }

  if len(globalOptions.FiltersOut) != 0 {
    for _, f := range globalOptions.FiltersOut {
      filtersOut.Add(f)
    }
  }

  err = filtersIn.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error intializing filters-in").Error())
    os.Exit(1)
  }

  err = filtersCmdIn.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error intializing filters-cmd-in").Error())
    os.Exit(1)
  }

  err = filtersCmdOut.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error intializing filters-cmd-out").Error())
    os.Exit(1)
  }

  err = filtersOut.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error intializing filters-out").Error())
    os.Exit(1)
  }

  go func() {
    _, err := io.Copy(filtersCmdOut, cmd)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error reading from command").Error())
      os.Exit(1)
    }

    err = filtersCmdOut.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error closing command").Error())
      os.Exit(1)
    }
  }()

  var cmdOut io.Reader = filtersCmdOut

  if globalOptions.TeeCmdOut != nil {
    err := globalOptions.TeeCmdOut.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error initializing tee command output").Error())
      os.Exit(1)
    }

    cmdOut = io.TeeReader(filtersCmdOut, globalOptions.TeeCmdOut)
  }

  if globalOptions.TeeCmdIn != nil {
    err := globalOptions.TeeCmdIn.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error initializing tee command input").Error())
      os.Exit(1)
    }
  }

  encoders := pipeline.New()
  for _, enc := range globalOptions.Encoders {
    err := encoders.Add(enc)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in setting up pipeline -encoders").Error())
      os.Exit(1)
    }
  }

  err = encoders.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in intializing pipeline -encoders").Error())
    os.Exit(1)
  }

  go func() {
    _, err := io.Copy(filtersOut, encoders)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in reading from -encoders").Error())
      os.Exit(1)
    }

    err = filtersOut.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in closing -filters-out").Error())
      os.Exit(1)
    }
  }()

  go func() {
    err = globalOptions.Output.Init()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err initializing output").Error())
      os.Exit(1)
    }

    var lastReader io.Reader = filtersOut

    if globalOptions.TeeOut != nil {
      err = globalOptions.TeeOut.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err initializing output").Error())
        os.Exit(1)
      }

      lastReader = io.TeeReader(lastReader, globalOptions.TeeOut)
    }

    _, err = io.Copy(globalOptions.Output, lastReader)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in reading encoder").Error())
      os.Exit(1)
    }

    globalOptions.Output.Chomp(globalOptions.Chomp)

    err = globalOptions.Output.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err closing output").Error())
      os.Exit(1)
    }

    done <- struct{}{}
  }()

  if globalOptions.FromByteOut != 0 || globalOptions.ToByteOut != 0 {
    go func() {
       _, err := io.Copy(byteCounterOut, cmdOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err reading in command").Error())
        os.Exit(1)
      }

      err = byteCounterOut.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func() {
       _, err = io.Copy(encoders, byteCounterOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err reading in byteCounterOut: %v").Error())
        os.Exit(1)
      }

      err = encoders.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  } else {
    go func() {
       _, err = io.Copy(encoders, cmdOut)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err reading in encoderReader").Error())
        os.Exit(1)
      }

      err = encoders.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  }

  decoders := pipeline.New()
  for _, dec := range globalOptions.Decoders {
    err := decoders.Add(dec)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in setting up pipeline decoders").Error())
      os.Exit(1)
    }
  }

  err = decoders.Init()
  if err != nil {
    fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in initializing pipeline decoders").Error())
    os.Exit(1)
  }

  if globalOptions.FromByteIn != 0 || globalOptions.ToByteIn != 0 {
    go func() {
       _, err = io.Copy(byteCounterIn, decoders)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error reading from -decoders").Error())
        os.Exit(1)
      }

      err = byteCounterIn.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()

    go func() {
      _, err := io.Copy(filtersCmdIn, byteCounterIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error reading from -decoders").Error())
        os.Exit(1)
      }

      err = filtersCmdIn.Close()
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error closing -filters-cmd-in").Error())
        os.Exit(1)
      }
    }()

    go func() {
      var cmdIn io.Reader = filtersCmdIn

      if globalOptions.TeeCmdIn != nil {
        cmdIn = io.TeeReader(cmdIn, globalOptions.TeeCmdIn)
      }

       _, err := io.Copy(cmd, cmdIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err reading in byteCounterIn").Error())
        os.Exit(1)
      }

      err = cmd.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  } else {
    go func() {
      _, err := io.Copy(filtersCmdIn, decoders)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error reading from -decoders").Error())
        os.Exit(1)
      }

      err = filtersCmdIn.Close()
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error closing -filters-cmd-in").Error())
        os.Exit(1)
      }
    }()

    go func() {
      var cmdIn io.Reader = filtersCmdIn

      if globalOptions.TeeCmdIn != nil {
        cmdIn = io.TeeReader(cmdIn, globalOptions.TeeCmdIn)
      }

      _, err = io.Copy(cmd, cmdIn)
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Err in reading decoder decoderReader").Error())
        os.Exit(1)
      }

      err = cmd.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }()
  }

  go func() {
    err := globalOptions.Input.Init()
    if err != nil {
      if err == io.EOF {
        globalFlags.Chomp = true
        decoders.Close()
        return
      }

      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error initializing input").Error())
      os.Exit(1)
    }

    _, err = io.Copy(filtersIn, globalOptions.Input)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in reading -in").Error())
      os.Exit(1)
    }

    err = globalOptions.Input.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in closing -in").Error())
      os.Exit(1)
    }

    err = filtersIn.Close()
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in closing filters-in").Error())
      os.Exit(1)
    }
  }()

  go func() {
    var reader io.Reader = filtersIn

    if globalOptions.TeeIn != nil {
      err := globalOptions.TeeIn.Init()
      if err != nil {
        fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error initializing tee input").Error())
        os.Exit(1)
      }

      reader = io.TeeReader(filtersIn, globalOptions.TeeIn)
    }

    _, err = io.Copy(decoders, reader)
    if err != nil {
      fmt.Fprintf(os.Stderr, errors.Wrap(err, "Error in decoding input").Error())
      os.Exit(1)
    }

    if globalOptions.TeeIn != nil {
      err = globalOptions.TeeIn.Close()
      if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
      }
    }

    err = decoders.Close()
    if err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
    }
  }()

  <- done
}
