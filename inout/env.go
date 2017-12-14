package inout

import (
  "net/url"
  "os"
  "bytes"
  "github.com/pkg/errors"
  "io"
)

var DefaultEnv = &Env{
  name: "env:",
  description: "Read and unset environment variable. Doesn't work for output",
}

type Env struct {
  description string
  name string
}

func (f Env) In(uri *url.URL) (Input) {
  return &EnvInput{
    env: uri.Opaque,
    name: "env-input",
  }
}

func (f Env) Out(uri *url.URL) (Output) {
  return &EnvOutput{
    name: "env-output",
  }
}

func (f Env) Name() (string) {
  return f.name
}

func (f Env) Description() (string) {
  return f.description
}

type EnvInput struct {
  env string
  reader io.Reader
  name string
}

func (in *EnvInput) Init() (error) {
  if in.env == "" {
    return errors.New("Env filetype is missing variable\n")
  }

  env := os.Getenv(in.env)
  in.reader = bytes.NewBufferString(env)

  if env != "" {
    err := os.Unsetenv(env)
    if err != nil {
      return errors.Wrapf(err, "Error unsetting env %s\n", env)
    }
  }

  return nil
}

func (in *EnvInput) Read(p []byte) (int, error) {
  return in.reader.Read(p)
}

func (in *EnvInput) Close() (error) {
  return nil
}

func (in EnvInput) Name() (string) {
  return in.name
}

type EnvOutput struct {
  name string
}

func (out *EnvOutput) Init() (error) {
  return errors.New("Inout module doesn't support output\n")
}

func (out *EnvOutput) Write(data []byte) (int, error) {
  return 0, io.EOF
}

func (out *EnvOutput) Close() (error) {
  return nil
}

func (out EnvOutput) Name() (string) {
  return out.name
}

func (out EnvOutput) Chomp(chomp bool) {}
