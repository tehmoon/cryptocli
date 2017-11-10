package codec

import (
  "io"
  "fmt"
  "encoding/hex"
)

var DefaultHex = &Hex{
  name: "hex",
  description: "hex encode output and hex decode input",
}

type Hex struct {
  name string
  description string
}

func (codec Hex) Name() (string) {
  return codec.name
}

func (codec Hex) Description() (string) {
  return codec.description
}

type HexDecoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

type HexEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func (codec Hex) Decoder() (CodecDecoder) {
  dec := &HexDecoder{}
  dec.pipeReader, dec.pipeWriter = io.Pipe()
  return dec
}

func (codec Hex) Encoder() (CodecEncoder) {
  enc := &HexEncoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  return enc
}

func (codec HexDecoder) Init() (error) {
  return nil
}

func (dec *HexDecoder) Read(p []byte) (int, error) {
  return dec.pipeReader.Read(p)
}

func (dec *HexDecoder) Write(data []byte) (int, error) {
  buff := make([]byte, hex.DecodedLen(len(data)))
  _, err := hex.Decode(buff, data)
  if err != nil {
    return 0, err
  }

  i, err := dec.pipeWriter.Write(buff)
  return i * 2, err
}

func (dec *HexDecoder) Close() (error) {
  return dec.pipeWriter.Close()
}

func (enc HexEncoder) Init() (error) {
  return nil
}

func (enc *HexEncoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc *HexEncoder) Write(data []byte) (int, error) {
  fmt.Fprintf(enc.pipeWriter, "%x", data)

  return len(data), nil
}

func (enc *HexEncoder) Close() (error) {
  return enc.pipeWriter.Close()
}
