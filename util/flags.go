package util

import (
  "../codec"
  "fmt"
  "flag"
  "os"
  "math/big"
  "github.com/pkg/errors"
  "strings"
)

type GlobalFlags struct {
  Chomp bool
  Encoders string
  Decoders string
  FromByteIn string
  FromByteOut string
  ToByteIn string
  ToByteOut string
}

var ErrBadFlag = errors.New("Bad flags")

func ParseFlags(set *flag.FlagSet, globalFlags *GlobalFlags) (*GlobalOptions) {
  globalOptions := newGlobalOptions()

  set.Parse(os.Args[2:])

  if globalFlags.Decoders != "" {
    codecs, err := codec.ParseAll(strings.Split(globalFlags.Decoders, ","))
    if err != nil {
      fmt.Println(err)
      os.Exit(2)
    }

    decoders := make([]codec.CodecDecoder, len(codecs))
    for i, decoder := range codecs {
      decoders[i] = decoder.Decoder()
    }

    globalOptions.Decoders = decoders
  }

  if globalFlags.Encoders != "" {
    codecs, err := codec.ParseAll(strings.Split(globalFlags.Encoders, ","))
    if err != nil {
      fmt.Println(err)
      os.Exit(2)
    }

    encoders := make([]codec.CodecEncoder, len(codecs))
    for i, encoder := range codecs {
      encoders[i] = encoder.Encoder()
    }

    globalOptions.Encoders = encoders
  }

  var ok bool

  globalOptions.FromByteIn, ok = parseBytePositionArgument(globalFlags.FromByteIn)
  if !ok {
    fmt.Println("Bad -from-byte-in number")
    os.Exit(2)
  }

  globalOptions.ToByteIn, ok = parseBytePositionArgument(globalFlags.ToByteIn)
  if !ok {
    fmt.Println("Bad -to-byte-in number")
    os.Exit(2)
  }

  globalOptions.FromByteOut, ok = parseBytePositionArgument(globalFlags.FromByteOut)
  if !ok {
    fmt.Println("Bad -from-byte-out number")
    os.Exit(2)
  }

  globalOptions.ToByteOut, ok = parseBytePositionArgument(globalFlags.ToByteOut)
  if !ok {
    fmt.Println("Bad -to-byte-out number")
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
