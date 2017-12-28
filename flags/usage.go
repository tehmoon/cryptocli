package flags

import (
  "os"
  "flag"
  "../codec"
  "../inout"
  "../filter"
  "fmt"
)

type Usage struct {
  CommandLine string
  Other string
}

func PrintUsage(usage *Usage, codecList []codec.Codec, inoutList []inout.IO, filterList []filter.Initializer) (func ()) {
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

    if len(filter.FilterList) > 0 {
      fmt.Fprintln(os.Stderr, "\nFilters:")
      for _, f := range filterList {
        fmt.Fprintf(os.Stderr, "  %s\n\t%s\n", f.Name(), f.Description())
      }
    }

    if usage.Other != "" {
      fmt.Fprintf(os.Stderr, "\n%s\n", usage.Other)
    }
  }
}
