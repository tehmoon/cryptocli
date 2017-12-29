package inout

import (
  "net/url"
  "io"
)

var DefaultNull = &Null{
  name: "null:",
  description: "Behaves like /dev/null on *nix system",
}

type Null struct {
  description string
  name string
}

func (n Null) In(uri *url.URL) (Input) {
  return &NullInput{
    name: "null-input",
  }
}

func (n Null) Out(uri *url.URL) (Output) {
  return &NullOutput{
    name: "null-output",
  }
}

func (n Null) Name() (string) {
  return n.name
}

func (n Null) Description() (string) {
  return n.description
}

type NullInput struct {
  name string
}

func (in NullInput) Init() (error) {
  return nil
}

func (in NullInput) Read(p []byte) (int, error) {
  return 0, io.EOF
}

func (in NullInput) Close() (error) {
  return nil
}

func (in NullInput) Name() (string) {
  return in.name
}

type NullOutput struct {
  name string
}

func (out NullOutput) Init() (error) {
  return nil
}

func (out NullOutput) Write(data []byte) (int, error) {
  return len(data), nil
}

func (out NullOutput) Chomp(chomp bool) {
}

func (out NullOutput) Close() (error) {
  return nil
}

func (out NullOutput) Name() (string) {
  return out.name
}
