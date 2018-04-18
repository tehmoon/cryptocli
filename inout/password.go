package inout

import (
  "fmt"
  "net/url"
  "os"
  "golang.org/x/crypto/ssh/terminal"
  "github.com/pkg/errors"
  "io"
)

var DefaultPassword = &Password{
  name: "password:",
  description: "Reads a line of input from a terminal without local echo",
}

type Password struct {
  description string
  name string
}

func (f Password) In(uri *url.URL) (Input) {
  return &PasswordInput{
    name: "password-input",
  }
}

func (f Password) Out(uri *url.URL) (Output) {
  return &PasswordOutput{
    name: "password-output",
  }
}

func (f Password) Name() (string) {
  return f.name
}

func (f Password) Description() (string) {
  return f.description
}

type PasswordInput struct {
  word string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  name string
}

func (in *PasswordInput) Init() (error) {
  fd0 := int(os.Stdin.Fd())

  if ! terminal.IsTerminal(fd0) {
    return errors.New("password-in is not in a terminal")
  }

  fmt.Fprintf(os.Stderr, "Enter password: ")

  buf, err := terminal.ReadPassword(fd0)
  if err != nil {
  return errors.Wrap(err, "error reading password")
  }

  os.Stderr.Write([]byte{'\n',})

  in.pipeReader, in.pipeWriter = io.Pipe()

  go func(buff []byte, writer *io.PipeWriter) {
    _, err := writer.Write(buf)
     writer.CloseWithError(err)
  }(buf, in.pipeWriter)

  return nil
}

func (in *PasswordInput) Read(p []byte) (int, error) {
  return in.pipeReader.Read(p)
}

func (in *PasswordInput) Close() (error) {
  return in.pipeWriter.Close()
}

func (in PasswordInput) Name() (string) {
  return in.name
}

type PasswordOutput struct {
  name string
}

func (out *PasswordOutput) Init() (error) {
  return errors.New("Inout module doesn't support output\n")
}

func (out *PasswordOutput) Write(data []byte) (int, error) {
  return 0, io.EOF
}

func (out *PasswordOutput) Close() (error) {
  return nil
}

func (out PasswordOutput) Name() (string) {
  return out.name
}

func (out PasswordOutput) Chomp(chomp bool) {}
