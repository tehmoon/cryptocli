package flags

import (
  "../codec"
  "../inout"
  "../filter"
)

type GlobalOptions struct {
  Decoders []codec.CodecDecoder
  Encoders []codec.CodecEncoder
  FromByteIn uint64
  ToByteIn uint64
  FromByteOut uint64
  ToByteOut uint64
  Input inout.Input
  Output inout.Output
  Bs int
  Ibs int
  Obs int
  FiltersIn []filter.Filter
  FiltersCmdIn []filter.Filter
  FiltersCmdOut []filter.Filter
  FiltersOut []filter.Filter
  TeeIn inout.Output
  TeeCmdIn inout.Output
  TeeCmdOut inout.Output
  TeeOut inout.Output
  Chomp bool
}

func newGlobalOptions() (*GlobalOptions) {
  globalOptions := &GlobalOptions{
    FiltersIn: make([]filter.Filter, 0),
    FiltersCmdIn: make([]filter.Filter, 0),
    FiltersCmdOut: make([]filter.Filter, 0),
    FiltersOut: make([]filter.Filter, 0),
  }

  cv, _ := codec.Parse("binary")
  globalOptions.Decoders = []codec.CodecDecoder{cv.Codec.Decoder(cv.Values),}
  globalOptions.Encoders = []codec.CodecEncoder{cv.Codec.Encoder(cv.Values),}

  return globalOptions
}
