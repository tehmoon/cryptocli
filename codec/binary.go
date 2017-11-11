package codec

import (
  "io"
)

var DefaultBinary = &Binary{
  name: "binary",
  description: "Do nothing in input and nothing in output",
}

type Binary struct {
  name string
  description string
}

func (codec Binary) Name() (string) {
  return codec.name
}

func (codec Binary) Description() (string) {
  return codec.description
}

type BinaryDecoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

type BinaryEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func (codec Binary) Decoder() (CodecDecoder) {
  dec := &BinaryDecoder{}
  dec.pipeReader, dec.pipeWriter = io.Pipe()
  return dec
}

func (codec Binary) Encoder() (CodecEncoder) {
  enc := &BinaryEncoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  return enc
}

func (dec BinaryDecoder) Init() (error) {
  return nil
}

func (dec *BinaryDecoder) Read(p []byte) (int, error) {
  return dec.pipeReader.Read(p)
}

func (dec *BinaryDecoder) Write(data []byte) (int, error) {
  return dec.pipeWriter.Write(data)
}

func (dec *BinaryDecoder) Close() (error) {
  return dec.pipeWriter.Close()
}

func (enc BinaryEncoder) Init() (error) {
  return nil
}

func (enc *BinaryEncoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc *BinaryEncoder) Write(data []byte) (int, error) {
  return enc.pipeWriter.Write(data)
}

func (enc *BinaryEncoder) Close() (error) {
  return enc.pipeWriter.Close()
}
