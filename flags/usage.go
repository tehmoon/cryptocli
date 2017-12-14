package flags

import (
  "os"
  "flag"
  "../codec"
  "../inout"
  "fmt"
)

type Usage struct {
  CommandLine string
  Other string
}

func PrintUsage(usage *Usage, codecList []codec.Codec, inoutList []inout.IO) (func ()) {
  return func() {
    fmt.Fprintf(os.Stderr, "Usage: %s [<Options>] %s\n\nOptions:\n", os.Args[0], usage.CommandLine)
    flag.PrintDefaults()
    if len(codecList) > 0 {
      fmt.Fprintln(os.Stderr, "\nCodecs:")
      for _, c := range codec.CodecList {
        fmt.Fprintf(os.Stderr, "  %s\n\t%s\n", c.Name(), c.Description())
      }
    }

    if len(inout.InoutList) > 0 {
      fmt.Fprintln(os.Stderr, "\nFileTypes:")
      for _, i := range inoutList {
        fmt.Fprintf(os.Stderr, "  %s\n\t%s\n", i.Name(), i.Description())
      }
    }

    if usage.Other != "" {
      fmt.Fprintf(os.Stderr, "\n%s\n", usage.Other)
    }
  }
}
