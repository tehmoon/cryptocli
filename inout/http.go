package inout

import (
  "net/url"
  "net/http"
  "io"
)

var DefaultHttp = &Http{
  name: "http://",
  description: "Get http url or post the output to https. Max redirects count is 3. Will fail if scheme changes.",
}

type Http struct{
  description string
  name string
}

func (h Http) In(uri *url.URL) (Input) {
  in := &HttpInput{
    uri: uri,
    name: "http-input",
  }

  in.pipeReader, in.pipeWriter = io.Pipe()

  return in
}

func (h Http) Out(uri *url.URL) (Output) {
  out := &HttpOutput{
    uri: uri,
    name: "http-output",
    sync: make(chan error),
  }

  out.pipeReader, out.pipeWriter = io.Pipe()

  return out
}

func (h Http) Name() (string) {
  return h.name
}

func (h Http) Description() (string) {
  return h.description
}

type HttpInput struct {
  uri *url.URL
  name string
  body io.ReadCloser
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

type HttpOutput struct {
  uri *url.URL
  name string
  sync chan error
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
}

func (in HttpInput) Name() (string) {
  return in.name
}

func (in *HttpInput) Init() (error) {
  body, err := readHTTP(*in.uri, &http.Client{CheckRedirect: httpRedirectPolicy(),})
  if err != nil {
    return err
  }

  in.body = body

  go func() {
    io.Copy(in.pipeWriter, in.body)

    in.pipeWriter.Close()
  }()

  return nil
}

func (in *HttpInput) Close() (error) {
  return in.body.Close()
}

func (in *HttpInput) Read(p []byte) (int, error) {
  return in.pipeReader.Read(p)
}

func (out HttpOutput) Init() (error) {
  writer, err := writeHTTP(*out.uri, &http.Client{CheckRedirect: httpRedirectPolicy(),}, out.sync)
  if err != nil {
    return err
  }

  go func() {
    io.Copy(writer, out.pipeReader)
    writer.Close()
  }()

  return nil
}

func (out HttpOutput) Write(data []byte) (int, error) {
  return out.pipeWriter.Write(data)
}

func (out HttpOutput) Close() (error) {
  out.pipeWriter.Close()

  return <- out.sync
}

func (out HttpOutput) Name() (string) {
  return out.name
}

func (out HttpOutput) Chomp(chomp bool) {}
