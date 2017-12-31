package inout

import (
  "net/url"
  "io"
  "github.com/tehmoon/errors"
  "bytes"
)

var DefaultAscii = &Ascii{
  name: "ascii:",
  description: "Decode ascii value and use it for input. Doesn't work for output",
}

type Ascii struct {
  description string
  name string
}

func (h Ascii) In(uri *url.URL) (Input) {
  return &AsciiInput{
    asciiString: uri.Opaque,
    name: "ascii-input",
  }
}

func (h Ascii) Out(uri *url.URL) (Output) {
  return &AsciiOutput{
    name: "ascii-output",
  }
}

func (h Ascii) Name() (string) {
  return h.name
}

func (h Ascii) Description() (string) {
  return h.description
}

type AsciiInput struct {
  name string
  asciiString string
  reader io.Reader
}

func (in *AsciiInput) Init() (error) {
  in.reader = bytes.NewBufferString(in.asciiString)

  return nil
}

func (in AsciiInput) Read(p []byte) (int, error) {
  return in.reader.Read(p)
}

func (in AsciiInput) Close() (error) {
  return nil
}

func (in AsciiInput) Name() (string) {
  return in.name
}

type AsciiOutput struct {
  name string
}

func (out AsciiOutput) Init() (error) {
  return errors.New("Ascii module doesn't support output\n")
}

func (out AsciiOutput) Write(data []byte) (int, error) {
  return 0, io.EOF
}

func (out AsciiOutput) Close() (error) {
  return nil
}

func (out AsciiOutput) Name() (string) {
  return out.name
}

func (out AsciiOutput) Chomp(chomp bool) {}
