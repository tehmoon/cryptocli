package codec

import (
  "compress/gzip"
  "io"
)

var DefaultGzip = &Gzip{
  name: "gzip",
  description: "gzip compress output and gzip decompress input",
}

type Gzip struct{
  name string
  description string
}

func (codec Gzip) Name() (string) {
  return codec.name
}

func (codec Gzip) Description() (string) {
  return codec.description
}

type GzipDecoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  gzipReader *gzip.Reader
}

type GzipEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  gzipWriter *gzip.Writer
}

func (codec Gzip) Decoder() (CodecDecoder) {
  dec := &GzipDecoder{}
  dec.pipeReader, dec.pipeWriter = io.Pipe()
  return dec
}

func (codec Gzip) Encoder() (CodecEncoder) {
  enc := &GzipEncoder{}
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  enc.gzipWriter = gzip.NewWriter(enc.pipeWriter)
  return enc
}

func (dec *GzipDecoder) Read(p []byte) (int, error) {
  return dec.gzipReader.Read(p)
}

func (dec *GzipDecoder) Write(data []byte) (int, error) {
  return dec.pipeWriter.Write(data)
}

func (dec *GzipDecoder) Init() (error) {
  var err error
  dec.gzipReader, err = gzip.NewReader(dec.pipeReader)
  return err
}

func (dec *GzipDecoder) Close() (error) {
  if dec.gzipReader != nil {
    err := dec.gzipReader.Close()
    if err != nil {
      return err
    }
  }
  return dec.pipeWriter.Close()
}

func (enc GzipEncoder) Init() (error) {
  return nil
}

func (enc *GzipEncoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc *GzipEncoder) Write(data []byte) (int, error) {
  return enc.gzipWriter.Write(data)
}

func (enc *GzipEncoder) Close() (error) {
  err := enc.gzipWriter.Close()
  if err != nil {
    return err
  }

  return enc.pipeWriter.Close()
}
