package inout

import (
  "os"
  "net/url"
  "io"
  "github.com/tehmoon/errors"
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
    Env: os.Environ(),
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
    Env: os.Environ(),
  }

  output.pipeReader, output.pipeWriter = io.Pipe()

  return output
}

type PipeInput struct {
  Env []string
  pipeReader *io.PipeReader
  pipeWriter *io.PipeWriter
  command string
  cmd *exec.Cmd
  name string
  sync chan error
  doneReading chan struct{}
}

type PipeOutput struct {
  Env []string
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

func pipeStartAndAttachStdin(env []string, command string, args... string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, error) {
  cmd := exec.Command(command, args...)
  cmd.Env = env

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

func pipeStartAndAttachStdout(env []string, command string, args... string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
  cmd := exec.Command(command, args...)
  cmd.Env = env

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

  in.doneReading = make(chan struct{})
  in.sync = make(chan error)
  shell := pipeGetShell()

  ok := pipeCheckOS()
  if ! ok {
    return ErrInoutPipeBadOS
  }

  cmd, stdout, stderr, err := pipeStartAndAttachStdout(in.Env, shell, "-c", in.command)
  if err != nil {
    return err
  }

  in.cmd = cmd

  go func() {
    io.Copy(os.Stdout, stderr)
  }()

  go func() {
    var e error

    _, err := io.Copy(in.pipeWriter, stdout)
    if err != nil {
      e = errors.WrapErr(e, err)
    }

    err = in.pipeWriter.Close()
    if err != nil {
      e = errors.WrapErr(e, err)
    }

    err = in.cmd.Wait()
    if err != nil {
      e = errors.WrapErr(e, err)
    }

    <- in.doneReading
    in.sync <- e
  }()

  return nil
}

func (in *PipeInput) Read(p []byte) (int, error) {
  i, err := in.pipeReader.Read(p)
  if err != nil {
    in.doneReading <- struct{}{}
  }

  return i, err
}

func (in *PipeInput) Close() (error) {
  return <- in.sync
}

func (in PipeInput) Name() (string) {
  return in.name
}

func (out *PipeOutput) Init() (error) {
  if out.command == "" {
    return errors.New("No command to pipe to stdout\n")
  }

  out.sync = make(chan error)
  shell := pipeGetShell()

  ok := pipeCheckOS()
  if ! ok {
    return ErrInoutPipeBadOS
  }

  cmd, stdin, stderr, err := pipeStartAndAttachStdin(out.Env, shell, "-c", out.command)
  if err != nil {
    return err
  }

  out.cmd = cmd

  go func() {
    io.Copy(os.Stderr, stderr)
  }()

  go func() {
    io.Copy(stdin, out.pipeReader)
    stdin.Close()
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

  return <- out.sync
}

func (out PipeOutput) Name() (string) {
  return out.name
}

func (out PipeOutput) Chomp(chomp bool) {}
