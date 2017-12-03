package flags

import (
  "../codec"
  "fmt"
  "flag"
  "os"
  "math/big"
  "github.com/pkg/errors"
  "strings"
  "../inout"
)

type GlobalFlags struct {
  Chomp bool
  Encoders string
  Decoders string
  FromByteIn string
  FromByteOut string
  ToByteIn string
  ToByteOut string
  In string
  Out string
  TeeIn string
  TeeCmdIn string
  TeeCmdOut string
  TeeOut string
}

var ErrBadFlag = errors.New("Bad flags\n")

func ParseFlags(set *flag.FlagSet, globalFlags *GlobalFlags) (*GlobalOptions) {
  globalOptions := newGlobalOptions()

  set.Parse(os.Args[2:])

  var err error

  if globalFlags.TeeIn != "" {
    globalOptions.TeeIn, err = inout.ParseOutput(globalFlags.TeeIn)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v", err)
      os.Exit(2)
    }
  }
  if globalFlags.TeeCmdIn != "" {
    globalOptions.TeeCmdIn, err = inout.ParseOutput(globalFlags.TeeCmdIn)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v", err)
      os.Exit(2)
    }
  }
  if globalFlags.TeeCmdOut != "" {
    globalOptions.TeeCmdOut, err = inout.ParseOutput(globalFlags.TeeCmdOut)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v", err)
      os.Exit(2)
    }
  }
  if globalFlags.TeeOut != "" {
    globalOptions.TeeOut, err = inout.ParseOutput(globalFlags.TeeOut)
    if err != nil {
      fmt.Fprintf(os.Stderr, "%v", err)
      os.Exit(2)
    }
  }

  globalOptions.Input, err = inout.ParseInput(globalFlags.In)
  if err != nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(2)
  }

  globalOptions.Output, err = inout.ParseOutput(globalFlags.Out)
  if err != nil {
    fmt.Fprintf(os.Stderr, "%v", err)
    os.Exit(2)
  }

  if globalFlags.Decoders != "" {
    cvs, err := codec.ParseAll(strings.Split(globalFlags.Decoders, ","))
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing decoders. Err: %v", err)
      os.Exit(2)
    }

    decoders := make([]codec.CodecDecoder, len(cvs))
    for i, cv := range cvs {
      dec := cv.Codec.Decoder(cv.Values)
      if dec == nil {
        fmt.Fprintf(os.Stderr, "Codec %s doesn't support decoding\n", cv.Codec.Name())
        os.Exit(2)
      }

      decoders[i] = dec
    }

    globalOptions.Decoders = decoders
  }

  if globalFlags.Encoders != "" {
    cvs, err := codec.ParseAll(strings.Split(globalFlags.Encoders, ","))
    if err != nil {
      fmt.Fprintf(os.Stderr, "Error parsing encoders. Err: %v", err)
      os.Exit(2)
    }

    encoders := make([]codec.CodecEncoder, len(cvs))
    for i, cv := range cvs {
      enc := cv.Codec.Encoder(cv.Values)
      if enc == nil {
        fmt.Fprintf(os.Stderr, "Codec %s doesn't support encoding\n", cv.Codec.Name())
        os.Exit(2)
      }

      encoders[i] = enc
    }

    globalOptions.Encoders = encoders
  }

  var ok bool

  globalOptions.FromByteIn, ok = parseBytePositionArgument(globalFlags.FromByteIn)
  if !ok {
    fmt.Fprintln(os.Stderr, "Bad -from-byte-in number")
    os.Exit(2)
  }

  globalOptions.ToByteIn, ok = parseBytePositionArgument(globalFlags.ToByteIn)
  if !ok {
    fmt.Fprintln(os.Stderr, "Bad -to-byte-in number")
    os.Exit(2)
  }

  globalOptions.FromByteOut, ok = parseBytePositionArgument(globalFlags.FromByteOut)
  if !ok {
    fmt.Fprintln(os.Stderr, "Bad -from-byte-out number")
    os.Exit(2)
  }

  globalOptions.ToByteOut, ok = parseBytePositionArgument(globalFlags.ToByteOut)
  if !ok {
    fmt.Fprintln(os.Stderr, "Bad -to-byte-out number")
    os.Exit(2)
  }

  if globalFlags.ToByteIn[0] == '+' {
    globalOptions.ToByteIn += globalOptions.FromByteIn
  }

  if globalFlags.ToByteOut[0] == '+' {
    globalOptions.ToByteOut += globalOptions.FromByteOut
  }

  return globalOptions
}

func SetupFlags(set *flag.FlagSet) (*GlobalFlags) {
  globalFlags := &GlobalFlags{}

  set.BoolVar(&globalFlags.Chomp, "chomp", false, "Get rid of the last \\n when not in pipe")
  set.StringVar(&globalFlags.Decoders, "decoders", "binary", "Set a list of codecs separated by ',' to decode input that will be process in the order given")
  set.StringVar(&globalFlags.Encoders, "encoders", "binary", "Set a list of codecs separated by ',' to encode output that will be process in the order given")
  set.StringVar(&globalFlags.FromByteIn, "from-byte-in", "0", "Skip the first x bytes of stdin. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10")
  set.StringVar(&globalFlags.ToByteIn, "to-byte-in", "0", "Stop at byte x of stdin.  Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-in")
  set.StringVar(&globalFlags.FromByteOut, "from-byte-out", "0", "Skip the first x bytes of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10")
  set.StringVar(&globalFlags.ToByteOut, "to-byte-out", "0", "Stop at byte x of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-out")
  set.StringVar(&globalFlags.In, "in", "", "Input <fileType> method")
  set.StringVar(&globalFlags.Out, "out", "", "Output <fileType> method")
  set.StringVar(&globalFlags.TeeIn, "tee-in", "", "Copy output before -encoders to <fileType>")
  set.StringVar(&globalFlags.TeeCmdIn, "tee-cmd-in", "", "Copy output after -decoders and before <command> to <fileType>")
  set.StringVar(&globalFlags.TeeCmdOut, "tee-cmd-out", "", "Copy output after <command> and before -encoders to <fileType>")
  set.StringVar(&globalFlags.TeeOut, "tee-out", "", "Copy output after -encoders to <fileType>")

  return globalFlags
}

func parseBytePositionArgument(mark string) (uint64, bool) {
  if mark[0] == '+' {
    mark = mark[1:]
  }

  i, ok := new(big.Int).SetString(mark, 0)
  if ! ok {
    return 0, false
  }

  cmp := new(big.Int).SetUint64((1 << 64) - 1).Cmp(i)
  if cmp < 0 {
    return 0, false
  }

  return i.Uint64(), true
}
