package command

import (
  "flag"
  "../flags"
  "runtime"
  "os"
  "os/exec"
  "github.com/pkg/errors"
  "io"
)

type Pipe struct {
  name string
  description string
  inputReader *io.PipeReader
  inputWriter *io.PipeWriter
  outputReader *io.PipeReader
  outputWriter *io.PipeWriter
  usage *flags.Usage
  flagSet *flag.FlagSet
  sync chan error
}

var DefaultPipe = &Pipe{
  name: "pipe",
  description: "Execute a command and attach stdin and stdout to the pipeline",
  usage: &flags.Usage{
    CommandLine: "<cmd> [args...]",
    Other: "",
  },
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

func (command *Pipe) Init() (error) {
  ok := pipeCheckOS()
  if ! ok {
    return errors.Errorf("Your operating sysytem: %s is not supported by the command pipe\n", runtime.GOOS)
  }

  command.inputReader, command.inputWriter = io.Pipe()
  command.outputReader, command.outputWriter = io.Pipe()
  command.sync = make(chan error)

  if len(command.flagSet.Args()) == 0 {
    command.inputWriter.Close()
    command.outputWriter.Close()

    return nil
  }

  cmd := exec.Command(command.flagSet.Args()[0], command.flagSet.Args()[1:]...) 

  stdin, err := cmd.StdinPipe()
  if err != nil {
    return errors.Wrap(err, "Unable to attach stdin")
  }

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return errors.Wrap(err, "Unable to attach stdout")
  }

  stderr, err := cmd.StderrPipe()
  if err != nil {
    return errors.Wrap(err, "Unalbe to attach stderr")
  }

  err = cmd.Start()
  if err != nil {
    return errors.Wrap(err, "Error starting command")
  }

  go func() {
    io.Copy(stdin, command.inputReader)
  }()

  go func() {
    io.Copy(command.outputWriter, stdout)
    command.outputWriter.Close()
  }()

  go func() {
    io.Copy(os.Stderr, stderr)
  }()

  go func() {
    command.sync <- cmd.Wait()
  }()

  return nil
}

func (command Pipe) Read(p []byte) (int, error) {
  return command.outputReader.Read(p)
}

func (command Pipe) Write(data []byte) (int, error) {
  return command.inputWriter.Write(data)
}

func (command Pipe) Close() (error) {
  err := <- command.sync
  command.inputWriter.Close()

  return err
}

func (command Pipe) Usage() (*flags.Usage) {
  return command.usage
}

func (command Pipe) Name() (string) {
  return command.name
}

func (command Pipe) Description() (string) {
  return command.description
}

func (command *Pipe) SetupFlags(set *flag.FlagSet) {
  command.flagSet = set
}

func (command *Pipe) ParseFlags() (error) {
  return nil
}
