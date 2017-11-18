package flags

import (
  "../codec"
  "../inout"
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
  Tee inout.Output
}

func newGlobalOptions() (*GlobalOptions) {
  globalOptions := &GlobalOptions{}

  c, _ := codec.Parse("binary")
  globalOptions.Decoders = []codec.CodecDecoder{c.Decoder(),}
  globalOptions.Encoders = []codec.CodecEncoder{c.Encoder(),}

  return globalOptions
}
