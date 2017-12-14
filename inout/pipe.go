package inout

import (
  "os"
  "net/url"
  "io"
  "github.com/pkg/errors"
  "runtime"
  "os/exec"
)

var (
  DefaultPipe = &Pipe{
    name: "pipe:",
    description: "Run a command in a sub shell. Either write to the command's stdin or read from its stdout.",
  }
  ErrInoutPipeBadOS = errors.Errorf("Operating system: %s is not supported by inout pipe\n", runtime.GOOS)
)

type Pipe struct {
  name string
  description string
}

func (p Pipe) In(uri *url.URL) (Input) {
  command := uri.Opaque
  if command == "" {
    command = uri.Path
  }

  input := &PipeInput{
    command: command,
    name: "pipe-input",
    sync: make(chan error),
  }

  input.pipeReader, input.pipeWriter = io.Pipe()

  return input
}

func (p Pipe) Name() (string) {
  return p.name
}

func (p Pipe) Description() (string) {
  return p.description
}

func (p Pipe) Out(uri *url.URL) (Output) {
  command := uri.Opaque
  if command == "" {
    command = uri.Path
  }

  output := &PipeOutput{
    command: command,
    name: "pipe-output",
    sync: make(chan error),
  }

  output.pipeReader, output.pipeWriter = io.Pipe()

  return output
}

type PipeInput struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  command string
  cmd *exec.Cmd
  name string
  sync chan error
}

type PipeOutput struct {
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  command string
  cmd *exec.Cmd
  name string
  sync chan error
}

func pipeCheckOS() (bool) {
  switch runtime.GOOS {
    case "linux":
    case "darwin":
    case "openbsd":
    default:
      return false
  }

  return true
}

func pipeGetShell() (string) {
  shell := os.Getenv("SHELL")
  if shell == "" {
    shell = "/bin/sh"
  }

  return shell
}

func pipeStartAndAttachStdin(command string, args... string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
  cmd := exec.Command(command, args...)

  stdin, err := cmd.StdinPipe()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error creating stdin pipe")
  }

  stderr, err := cmd.StderrPipe()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error creating stderr pipe")
  }

  err = cmd.Start()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error starting command")
  }

  return cmd, stdin, stderr, err
}

func pipeStartAndAttachStdout(command string, args... string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
  cmd := exec.Command(command, args...)

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error creating stdout pipe")
  }

  stderr, err := cmd.StderrPipe()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error creating stderr pipe")
  }

  err = cmd.Start()
  if err != nil {
    return nil, nil, nil, errors.Wrap(err, "Error starting command")
  }

  return cmd, stdout, stderr, err
}

func (in *PipeInput) Init() (error) {
  if in.command == "" {
    return errors.New("No command to pipe to stdout\n")
  }

  shell := pipeGetShell()

  ok := pipeCheckOS()
  if ! ok {
    return ErrInoutPipeBadOS
  }

  cmd, stdout, stderr, err := pipeStartAndAttachStdout(shell, "-c", in.command)
  if err != nil {
    return err
  }

  in.cmd = cmd

  go func() {
    io.Copy(os.Stderr, stderr)
  }()

  go func() {
    io.Copy(in.pipeWriter, stdout)

    in.pipeWriter.Close()
    in.sync <- in.cmd.Wait()
  }()

  return nil
}

func (in *PipeInput) Read(p []byte) (int, error) {
  return in.pipeReader.Read(p)
}

func (in *PipeInput) Close() (error) {
  return <- in.sync
}

func (in PipeInput) Name() (string) {
  return in.name
}

func (out PipeOutput) Init() (error) {
  if out.command == "" {
    return errors.New("No command to pipe to stdout\n")
  }

  shell := pipeGetShell()

  ok := pipeCheckOS()
  if ! ok {
    return ErrInoutPipeBadOS
  }

  cmd, stdin, stderr, err := pipeStartAndAttachStdin(shell, "-c", out.command)
  if err != nil {
    return err
  }

  out.cmd = cmd

  go func() {
    io.Copy(os.Stderr, stderr)
  }()

  go func() {
    io.Copy(stdin, out.pipeReader)
    out.sync <- out.cmd.Wait()
  }()

  return nil
}

func (out PipeOutput) Write(data []byte) (int, error) {
  return out.pipeWriter.Write(data)
}

func (out PipeOutput) Close() (error) {
  err := out.pipeWriter.Close()
  if err != nil {
    return err
  }

  return nil
}

func (out PipeOutput) Name() (string) {
  return out.name
}

func (out PipeOutput) Chomp(chomp bool) {}
