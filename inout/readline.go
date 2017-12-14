package inout

import (
  "net/url"
  "os"
  "bufio"
  "github.com/pkg/errors"
  "io"
)

var DefaultReadline = &Readline{
  name: "readline:",
  description: "Read lines from stdin until WORD is reached.",
}

type Readline struct {
  description string
  name string
}

func (f Readline) In(uri *url.URL) (Input) {
  return &ReadlineInput{
    word: uri.Opaque,
    name: "readline-input",
  }
}

func (f Readline) Out(uri *url.URL) (Output) {
  return &ReadlineOutput{
    name: "readline-output",
  }
}

func (f Readline) Name() (string) {
  return f.name
}

func (f Readline) Description() (string) {
  return f.description
}

type ReadlineInput struct {
  word string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  name string
}

func (in *ReadlineInput) Init() (error) {
  if in.word == "" {
    return errors.New("Readline filetype is missing WORD\n")
  }

  in.pipeReader, in.pipeWriter = io.Pipe()
  reader := bufio.NewReader(os.Stdin)

  go func() {
    var (
      err error
      line string
    )

    for {
      line, err = reader.ReadString('\n')
      if err != nil {
        break
      }

      if line[:len(line) -1 ] == in.word {
        break
      }

      _, err = in.pipeWriter.Write([]byte(line))
      if err != nil {
        break
      }
    }

    in.pipeWriter.CloseWithError(err)
  }()

  return nil
}

func (in *ReadlineInput) Read(p []byte) (int, error) {
  return in.pipeReader.Read(p)
}

func (in *ReadlineInput) Close() (error) {
  return in.pipeWriter.Close()
}

func (in ReadlineInput) Name() (string) {
  return in.name
}

type ReadlineOutput struct {
  name string
}

func (out *ReadlineOutput) Init() (error) {
  return errors.New("Inout module doesn't support output\n")
}

func (out *ReadlineOutput) Write(data []byte) (int, error) {
  return 0, io.EOF
}

func (out *ReadlineOutput) Close() (error) {
  return nil
}

func (out ReadlineOutput) Name() (string) {
  return out.name
}

func (out ReadlineOutput) Chomp(chomp bool) {}
