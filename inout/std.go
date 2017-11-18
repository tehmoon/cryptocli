package inout

import (
  "net/url"
  "fmt"
  "os"
  "io"
)

var DefaultStd Std = Std{}

type Std struct{}

func (s Std) In(uri *url.URL) (Input) {
  return &Stdin{
    name: "stdin",
  }
}

func (s Std) Out(uri *url.URL) (Output) {
  return &Stdout{
    chomp: false,
    name: "stdout",
  }
}

func (s Std) Name() (string) {
  return ""
}

func (s Std) Description() (string) {
  return ""
}

type Stdin struct{
  name string
}

type Stdout struct{
  chomp bool
  name string
}

func (in Stdin) Init() (error) {
  fi, _ := os.Stdin.Stat()
  if (fi.Mode() & os.ModeCharDevice != 0) {
    return io.EOF
  }

  return nil
}

func (in Stdin) Read(p []byte) (int, error) {
  return os.Stdin.Read(p)
}

func (in Stdin) Close() (error) {
  return nil
}

func (in Stdin) Name() (string) {
  return in.name
}

func (out Stdout) Write(data []byte) (int, error) {
  return os.Stdout.Write(data)
}

func (out *Stdout) Init(chomp bool) (error) {
  out.chomp = chomp

  return nil
}

func (out Stdout) Close() (error) {
  fi, _ := os.Stdout.Stat()
  if (! (fi.Mode() & os.ModeCharDevice == 0) && ! out.chomp) {
    fmt.Println()
  }

  return nil
}

func (out Stdout) Name() (string) {
  return out.name
}
