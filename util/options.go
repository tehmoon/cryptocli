package util

import (
  "../codec"
)

type GlobalOptions struct {
  Decoder codec.CodecDecoder
  Encoder codec.CodecEncoder
  FromByteIn uint64
  ToByteIn uint64
  FromByteOut uint64
  ToByteOut uint64
  Bs int
  Ibs int
  Obs int
}

func newGlobalOptions() (*GlobalOptions) {
  globalOptions := &GlobalOptions{}

  c, _ := codec.Parse("binary")
  globalOptions.Decoder = c.Decoder()
  globalOptions.Encoder = c.Encoder()

  return globalOptions
}
