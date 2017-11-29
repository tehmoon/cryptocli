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

  cv, _ := codec.Parse("binary")
  globalOptions.Decoders = []codec.CodecDecoder{cv.Codec.Decoder(cv.Values),}
  globalOptions.Encoders = []codec.CodecEncoder{cv.Codec.Encoder(cv.Values),}

  return globalOptions
}
