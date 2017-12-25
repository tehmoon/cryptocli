package codec

import (
  "compress/gzip"
  "io"
  "net/url"
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
  reader *io.PipeReader
}

type GzipEncoder struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  writer *gzip.Writer
}

func (codec Gzip) Decoder(values url.Values) (CodecDecoder) {
  dec := &GzipDecoder{}
  return dec
}

func (codec Gzip) Encoder(values url.Values) (CodecEncoder) {
  enc := &GzipEncoder{}
  return enc
}

func (dec *GzipDecoder) Read(p []byte) (int, error) {
  return dec.reader.Read(p)
}

func (dec *GzipDecoder) Write(data []byte) (int, error) {
  return dec.pipeWriter.Write(data)
}

func (dec *GzipDecoder) Init() (error) {
  var err error

  dec.pipeReader, dec.pipeWriter = io.Pipe()
  reader, writer := io.Pipe()
  dec.reader = reader

  go func() {
    r, err := gzip.NewReader(dec.pipeReader)
    if err != nil {
      writer.CloseWithError(err)
      return
    }

    io.Copy(writer, r)
    r.Close()
    writer.Close()
  }()

  return err
}

func (dec *GzipDecoder) Close() (error) {
  return dec.pipeWriter.Close()
}

func (enc *GzipEncoder) Init() (error) {
  enc.pipeReader, enc.pipeWriter = io.Pipe()
  enc.writer = gzip.NewWriter(enc.pipeWriter)

  return nil
}

func (enc *GzipEncoder) Read(p []byte) (int, error) {
  return enc.pipeReader.Read(p)
}

func (enc *GzipEncoder) Write(data []byte) (int, error) {
  return enc.writer.Write(data)
}

func (enc *GzipEncoder) Close() (error) {
  err := enc.writer.Close()
  if err != nil {
    return err
  }

  return enc.pipeWriter.Close()
}
