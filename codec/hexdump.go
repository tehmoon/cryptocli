package codec

import (
  "io"
  "encoding/hex"
)

var DefaultHexdump = &Hexdump{
  name: "hexdump",
  description: "Encode output to hexdump -c. Doesn't support decoding",
}

type Hexdump struct {
  name string
  description string
}

func (codec Hexdump) Name() (string) {
  return codec.name
}

func (codec Hexdump) Description() (string) {
  return codec.description
}

type HexdumpEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  dumper io.WriteCloser
}

func (codec Hexdump) Decoder() (CodecDecoder) {
  return nil
}

func (codec Hexdump) Encoder() (CodecEncoder) {
  enc := &HexdumpEncoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  enc.dumper = hex.Dumper(enc.pipeWriter)

  return enc
}

func (encoder *HexdumpEncoder) Init() (error) {
  return nil
}

func (encoder *HexdumpEncoder) Write(data []byte) (int, error) {
  return encoder.dumper.Write(data)
}

func (encoder *HexdumpEncoder) Read(p []byte) (int, error) {
  return encoder.pipeReader.Read(p)
}

func (encoder *HexdumpEncoder) Close() (error) {
  err := encoder.dumper.Close()
  if err != nil {
    return err
  }

  return encoder.pipeWriter.Close()
}
