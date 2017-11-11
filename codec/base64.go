package codec

import (
  "io"
  "github.com/pkg/errors"
  "encoding/base64"
)

var DefaultBase64 = &Base64{
  name: "base64",
  description: "base64 decode input and base64 encode output",
}

type Base64 struct{
  name string
  description string
}

func (codec Base64) Name() (string) {
  return codec.name
}

func (codec Base64) Description() (string) {
  return codec.description
}

type Base64Decoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  decoder io.Reader
}

type Base64Encoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  encoder io.WriteCloser
}

func (codec Base64) Decoder() (CodecDecoder) {
  dec := &Base64Decoder{
  }
  dec.pipeReader, dec.pipeWriter = io.Pipe()
  dec.decoder = base64.NewDecoder(base64.StdEncoding, dec.pipeReader)
  return dec
}

func (codec Base64) Encoder() (CodecEncoder) {
  enc := &Base64Encoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  enc.encoder = base64.NewEncoder(base64.StdEncoding, enc.pipeWriter)
  return enc
}

func (codec Base64Decoder) Init() (error) {
  return nil
}

func (dec *Base64Decoder) Read(p []byte) (int, error) {
  return dec.decoder.Read(p)
}

func (dec *Base64Decoder) Write(data []byte) (int, error) {
  return dec.pipeWriter.Write(data)
}

func (dec *Base64Decoder) Close() (error) {
  return dec.pipeWriter.Close()
}

func (ecn *Base64Encoder) Init() (error) {
  return nil
}

func (enc *Base64Encoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc *Base64Encoder) Write(data []byte) (int, error) {
  return enc.encoder.Write(data)
}

func (enc *Base64Encoder) Close() (error) {
  err := enc.encoder.Close()
  if err != nil {
    return errors.Wrap(err, "Error closing the base64 encoder")
  }

  return enc.pipeWriter.Close()
}
